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
	master       map[uint16]*DataPacket
	destinations map[uint16][]net.UDPAddr //holds the info about the destinations unicast or multicast
	multicast    map[uint16]bool          //stores if an universe should be send out as multicast
	bind         string                   //stores the string with the binding information
	cid          [16]byte                 //the global cid for all packets
	sourceName   string                   //the global source name for all packets
}

//NewTransmitter creates a new Transmitter object and returns it. Only use one object for one
//network interface. bind is a string like "192.168.2.34" or "". It is used for binding the udpconnection.
//In most cases an empty string will be sufficient. The caller is responsible for closing!
//If you want to use multicast, you have to provide a binding string on some operation systems (eg Windows).
func NewTransmitter(binding string, cid [16]byte, sourceName string) (Transmitter, error) {
	//create tranmsitter:
	tx := Transmitter{
		universes:    make(map[uint16]chan [512]byte),
		master:       make(map[uint16]*DataPacket),
		destinations: make(map[uint16][]net.UDPAddr),
		multicast:    make(map[uint16]bool),
		bind:         "",
		cid:          cid,
		sourceName:   sourceName,
	}
	//create a udp address for testing, if the given bind address is possible
	addr, err := net.ResolveUDPAddr("udp", binding)
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
	ServerAddr, err := net.ResolveUDPAddr("udp", t.bind)
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
	masterPacket.SetData(make([]byte, 512)) //set 0 data
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
		serv.Close()
	}()

	return ch, nil
}

//IsActivated checks if the given universe was activated and returns true if this is the case
func (t *Transmitter) IsActivated(universe uint16) bool {
	if _, ok := t.universes[universe]; ok {
		return true
	}
	return false
}

//GetActivated returns a slice with all activated universes
func (t *Transmitter) GetActivated() (list []uint16) {
	list = make([]uint16, 0)
	for univ := range t.universes {
		list = append(list, univ)
	}
	return
}

//SetMulticast is for setting wether or not a universe should be send out via multicast.
//Keep in mind, that on some operating systems you have to provide a bind address.
func (t *Transmitter) SetMulticast(universe uint16, multicast bool) {
	t.multicast[universe] = multicast
}

//IsMulticast returns wether or not multicast is turned on for the given universe. true: on
func (t *Transmitter) IsMulticast(universe uint16) bool {
	return t.multicast[universe]
}

//SetDestinations sets a slice of destinations for the universe that is used for sending out.
//So multiple destinations are supported. Note: the existing slice will be overwritten!
//If you want no unicasting, just set an empty slice. If there is a string that could not be
//converted to an ip-address, this one is left out and an error slice will be returned,
//but the indices of the errors are not the same as the string indices on which the errors happened.
func (t *Transmitter) SetDestinations(universe uint16, destinations []string) []error {
	newDest := make([]net.UDPAddr, 0)
	errs := make([]error, 0)

	for _, dest := range destinations {
		if dest == "" {
			continue // continue if the string is empty
		}
		addr, err := net.ResolveUDPAddr("udp", dest+":5568")
		if err != nil {
			errs = append(errs, err)
			continue
		}
		newDest = append(newDest, *addr)
	}
	t.destinations[universe] = newDest

	if len(errs) == 0 {
		return nil
	}
	return errs
}

//Destinations returns all destinations that have been set via SetDestinations. Note: the returned
//slice contains deep copys and no change will affect the internal slice.
func (t *Transmitter) Destinations(universe uint16) []net.UDPAddr {
	new := make([]net.UDPAddr, len(t.destinations[universe]))
	copy(new, t.destinations[universe])
	return new
}

//handles sending and sequence numbering
func (t *Transmitter) sendOut(server *net.UDPConn, universe uint16) {
	//only send if the universe was activated
	if _, ok := t.master[universe]; !ok {
		return
	}
	//increase seqeunce number
	packet := t.master[universe]
	packet.SequenceIncr()
	//check if we have to transmitt via multicast
	if t.multicast[universe] {
		server.WriteToUDP(packet.getBytes(), generateMulticast(universe))
	}
	//for every destination, send out
	for _, dest := range t.destinations[universe] {
		server.WriteToUDP(packet.getBytes(), &dest)
	}
}

func generateMulticast(universe uint16) *net.UDPAddr {
	addr, _ := net.ResolveUDPAddr("udp", calcMulticastAddr(universe)+":5568")
	return addr
}
