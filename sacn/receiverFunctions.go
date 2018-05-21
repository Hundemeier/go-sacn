package sacn

import (
	"net"
	"time"

	"golang.org/x/net/ipv4"
)

//Set the timout according to the E1.31 protocol
const timeoutMs = 2500

//ReceiverSocket is used to listen on a network interface for sACN data.
//All the data that is arrived, and was activated is send to the DataChan.
//All errors from all universes are send through the ErrChan.
//This Receiver checks for out-of-order packets and sorts out packets with too low priority.
//Note: if there are two sources with the same highest priority, there will be send a
//"sources exceeded" error in the error channel.
//Furthermore: through the channel only changed data will be send.
//So the sequence numbers may not be in order.
type ReceiverSocket struct {
	DataChan     chan DataPacket
	ErrChan      chan ReceiveError
	socket       *ipv4.PacketConn
	stopListener chan struct{}
	activated    map[uint16]struct{}
	//a map that stores, wether a universe is activated for listening or not. It is used like a set
	//se methods: isActive, setActive
	lastDatas          map[uint16]*lastData
	multicastInterface *net.Interface // the interface that is used for joining multicast groups
}

type lastData struct {
	sources     map[[16]byte]source
	lastTime    time.Time
	lastSequ    byte
	lastDMXdata []byte
}

//ReceiveError contains the universe from which the error occured and the error itself
type ReceiveError struct {
	Universe uint16
	Error    error
}

/*
ActivateUniverse activates a universe for listening, so every data is send through the dataChannel
if multicast is true, the corresponding multicast group will be joined, if you provided an interface
in the `NewReceiverSocket`.
*/
func (r *ReceiverSocket) ActivateUniverse(universe uint16, multicast bool) {
	//activate universe and set the lastData object for the handling
	r.setActive(universe, true)
	r.lastDatas[universe] = &lastData{
		sources:  make(map[[16]byte]source),
		lastTime: time.Now(),
	}
	if multicast {
		r.socket.JoinGroup(r.multicastInterface, calcMulticastUDPAddr(universe))
	}
}

//DeactivateUniverse deactivates a universe from listening and no further data will be send through the
//data channel. If `universe` was not activated, nothing will happen.
func (r *ReceiverSocket) DeactivateUniverse(universe uint16) {
	r.setActive(universe, false)
	delete(r.lastDatas, universe)
	r.socket.LeaveGroup(r.multicastInterface, calcMulticastUDPAddr(universe))
}

//Close will close the open udp socket and close the data and error channel.
//If you want to receive again, create a new ReceiverSocket object. Do not call close twice!
func (r *ReceiverSocket) Close() {
	close(r.stopListener) // stop the running listener on the socket, because we will close the socket
	close(r.DataChan)
	close(r.ErrChan)
}

func (r *ReceiverSocket) isActive(universe uint16) bool {
	if _, ok := r.activated[universe]; ok {
		return true
	}
	return false
}

func (r *ReceiverSocket) setActive(universe uint16, active bool) {
	if active {
		r.activated[universe] = struct{}{}
	} else {
		delete(r.activated, universe)
	}
}

//GetAllActive returns a slice with all active universes
func (r *ReceiverSocket) GetAllActive() []uint16 {
	tmp := make([]uint16, 0)
	for key := range r.activated {
		tmp = append(tmp, key)
	}
	return tmp
}

func equalData(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func errToCh(universe uint16, err error, ch chan ReceiveError) {
	if err != nil {
		ch <- ReceiveError{
			Universe: universe,
			Error:    err,
		}
	}
}
