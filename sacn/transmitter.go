package sacn

import (
	"fmt"
	"net"
	"time"
)

//Transmitter : This struct is for managing the transmitting of sACN data.
//It handles all channels and overwatches what universes are already used.
type Transmitter struct {
	universes map[uint16]chan [512]byte
	//master stores the master DataPacket for all univereses. Its the last send out packet
	master      map[uint16]*DataPacket
	destination map[uint16]*net.UDPAddr //holds the info about the destinations unicast or multicast
	bind        string                  //stores the string with the binding information
	cid         [16]byte                //the global cid for all packets
	sourceName  string                  //the global source name for all packets
}

//NewTransmitter creates a new Transmitter object and returns it. Only use one object for one
//network interface. bind is a string like "192.168.2.34" or "". It is used for binding the udpconnection.
//In most cases an emtpy string will be sufficient. The caller is responsible for closing!
//If you want to use multicast, you have to provide a binding string on some operation systems (eg Windows).
func NewTransmitter(binding string, cid [16]byte, sourceName string) (Transmitter, error) {
	//create tranmsitter:
	tx := Transmitter{
		universes:   make(map[uint16]chan [512]byte),
		master:      make(map[uint16]*DataPacket),
		destination: make(map[uint16]*net.UDPAddr),
		bind:        "",
		cid:         cid,
		sourceName:  sourceName,
	}
	//create a udp address for testing, if the given bind address is possible
	addr, err := net.ResolveUDPAddr("udp", binding+":5568")
	if err != nil {
		return tx, err
	}
	serv, err := net.ListenUDP("udp", addr)
	serv.Close()
	if err != nil {
		return tx, err
	}
	//if everything is ok, set the bind address string
	tx.bind = binding
	return tx, nil
}

//Activate starts sending out DMX data on the given universe. It returns a channel that accepts
//byte slices and transmittes them to the unicast or multicast destination.
//If you want to deactivate the universe, simply close the channel.
func (t *Transmitter) Activate(universe uint16) (chan<- [512]byte, error) {
	//check if the universe is already activated
	if t.IsActivated(universe) {
		return nil, fmt.Errorf("the given universe %v is already activated", universe)
	}
	//create udp socket
	ServerAddr, err := net.ResolveUDPAddr("udp", t.bind+":5568")
	if err != nil {
		return nil, err
	}
	serv, err := net.ListenUDP("udp", ServerAddr)
	if err != nil {
		return nil, err
	}

	ch := make(chan [512]byte)
	t.universes[universe] = ch
	//init master packet
	masterPacket := NewDataPacket()
	masterPacket.SetCID(t.cid)
	masterPacket.SetSourceName(t.sourceName)
	masterPacket.SetUniverse(universe)
	t.master[universe] = &masterPacket

	//make goroutine that sends out every second a "keep alive" packet
	go func() {
		for {
			//if we have no master packet,break the loop
			if _, ok := t.master[universe]; !ok {
				break
			}
			t.sendOut(serv, universe)
			time.Sleep(time.Second * 1)
		}
	}()

	go func() {
		for i := range ch {
			t.master[universe].SetData(i[:])
			t.sendOut(serv, universe)
		}
		//if the channel was closed we send a last packet with stream terminated bit set
		t.master[universe].SetStreamTerminated(true)
		t.sendOut(serv, universe)
		//if the channel was closed, we deactivate the universe
		delete(t.master, universe)
		delete(t.universes, universe)
		fmt.Println("Test: serv.close")
		serv.Close()
	}()

	return ch, nil
}

//IsActivated checks if the given universe was activetd and returns true if this is the case
func (t *Transmitter) IsActivated(universe uint16) bool {
	if _, ok := t.universes[universe]; ok {
		return true
	}
	return false
}

//SetDestination sets a destination in form of an ip-address or "multicast" to an universe.
//eg: "192.168.2.34" or "multicast"
func (t *Transmitter) SetDestination(universe uint16, destination string) error {
	var dest *net.UDPAddr
	if destination == "multicast" {
		dest = generateMulticast(universe)
	} else {
		addr, err := net.ResolveUDPAddr("udp", destination+":5568")
		if err != nil {
			return err
		}
		dest = addr
	}
	t.destination[universe] = dest
	return nil
}

//handles sending and sequence numbering
func (t *Transmitter) sendOut(server *net.UDPConn, universe uint16) {
	//only send if the universe was activated
	if _, ok := t.master[universe]; !ok {
		return
	}
	//if no destination is available use multicast
	var target *net.UDPAddr
	if t.destination[universe] == nil {
		target = generateMulticast(universe)
	} else {
		target = t.destination[universe]
	}
	//increase seqeunce number
	packet := t.master[universe]
	packet.SequenceIncr()
	server.WriteToUDP(packet.getBytes(), target)
}

func generateMulticast(universe uint16) *net.UDPAddr {
	addr, _ := net.ResolveUDPAddr("udp", calcMulticastAddr(universe)+":5568")
	return addr
}
