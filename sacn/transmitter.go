package sacn

import (
	"errors"
	"net"
	"time"
)

//Transmitter : This struct is for managing the transmitting of sACN data.
//It handles all channels and overwatches what universes are already used.
type Transmitter struct {
	universes map[uint16]chan []byte
	//master stores the master DataPacket for all univereses. Its the last send out packet
	master      map[uint16]*DataPacket
	destination map[uint16]*net.UDPAddr //holds the info about the destinations unicast or multicast
	udp         *net.UDPConn            //stores the udp connection to use
	cid         [16]byte                //the global cid for all packets
	sourceName  string                  //the global source name for all packets
}

//NewTransmitter creates a new Transmitter object and returns it. Only use one object for one
//network interface. bind is a string like "192.168.2.34" or "". It is used for binding the udpconnection.
//In most cases an emtpy string will be sufficient. The caller is responsible for closing!
func NewTransmitter(bind string, cid [16]byte, sourceName string) (Transmitter, error) {
	//create a udp socket
	ServerAddr, err := net.ResolveUDPAddr("udp", bind+":5568")
	if err != nil {
		return Transmitter{}, err
	}
	ServerConn, err := net.ListenUDP("udp", ServerAddr)
	if err != nil {
		return Transmitter{}, err
	}
	return Transmitter{
		universes:   make(map[uint16]chan []byte),
		master:      make(map[uint16]*DataPacket),
		destination: make(map[uint16]*net.UDPAddr),
		udp:         ServerConn,
		cid:         cid,
		sourceName:  sourceName,
	}, nil
}

//Close closes the udp socket that was used.
//Note: all channels will be closed, so if you want to transmitt something after this call, a panic will occur
func (t *Transmitter) Close() error {
	//close all channels
	for _, v := range t.universes {
		close(v)
	}
	//close udp socket
	if t.udp != nil {
		return t.udp.Close()
	}
	return errors.New("no UDP socket could be closed, because it was nil")
}

//Activate starts sending out DMX data on the given universe. It returns a channel that accepts
//byte slices and transmittes them to the unicast or multicast destination.
//If you want to deactivate the universe, simply close the channel.
func (t *Transmitter) Activate(universe uint16) (chan<- []byte, error) {
	if t.udp == nil { //check if a udp socket is provided
		return nil, errors.New("no UDP socket could be used")
	}
	ch := make(chan []byte)
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
			t.sendOut(universe)
			time.Sleep(time.Second * 1)
		}
	}()

	go func() {
		for i := range ch {
			if len(i) <= 512 {
				t.master[universe].SetData(i)
				t.sendOut(universe)
			}
		}
		//if the channel was closed we send a last packet with stream terminated bit set
		t.master[universe].SetStreamTerminated(true)
		t.sendOut(universe)
		//if the channel was closed, we deactivate the universe
		delete(t.master, universe)
		delete(t.universes, universe)
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
	if !t.IsActivated(universe) {
		return errors.New("could not assign destination to universe: universe is not activated")
	}
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
func (t *Transmitter) sendOut(universe uint16) {
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
	t.udp.WriteToUDP(packet.getBytes(), target)
}

func generateMulticast(universe uint16) *net.UDPAddr {
	addr, _ := net.ResolveUDPAddr("udp", calcMulticastAddr(universe)+":5568")
	return addr
}
