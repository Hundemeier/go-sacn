package sacn

import (
	"net"
	"time"

	"golang.org/x/net/ipv4"
)

//Set the timout according to the E1.31 protocol
const timeoutMs = 2500

//ReceiverSocket is used to listen on a network interface for sACN data.
//The OnChangeCallback is used for changed DMX data. So if a source or priority changed,
//this callback will not be invoked if not the DMX data has changed.
//This Receiver checks for out-of-order packets and sorts out packets with too low priority.
type ReceiverSocket struct {
	socket             *ipv4.PacketConn
	stopListener       chan struct{}
	multicastInterface *net.Interface // the interface that is used for joining multicast groups
	//OnChangeCallback gets called if the data on one universe has changed. Gets called in own goroutine
	onChangeCallback func(old DataPacket, new DataPacket)
	//TimeoutCallback gets called, if a timout on a universe occurs. Gets called in own goroutine
	timeoutCallback func(universe uint16)
	lastDatas       map[uint16]lastData
	timeoutCalled   map[uint16]bool //true, if the timeout was called. To prevent send a timeoutcallback twice
}

type lastData struct {
	lastTime   time.Time
	lastPacket DataPacket
}

//JoinUniverse joins the used udp socket to the multicast-group that is used for the universe.
//After the multicast-group was joined, any source that transmitt on this universe via multicast
//should reach this socket.
//Please read the notice above about multicast use.
func (r *ReceiverSocket) JoinUniverse(universe uint16) {
	r.socket.JoinGroup(r.multicastInterface, calcMulticastUDPAddr(universe))
}

//LeaveUniverse will leave the mutlicast-group of the given universe.
//If the the socket was not joined to the multicast-group nothing will happen.
//Please note, that if you leave a group, a timeout may occurr, because no more data has arrived.
func (r *ReceiverSocket) LeaveUniverse(universe uint16) {
	r.socket.LeaveGroup(r.multicastInterface, calcMulticastUDPAddr(universe))
}

//Close will close the open udp socket.
//If you want to receive again, create a new ReceiverSocket object. Do not call close twice!
func (r *ReceiverSocket) Close() {
	close(r.stopListener) // stop the running listener on the socket, because we will close the socket
}

//SetOnChangeCallback sets the given function as callback for the receiver. If no old DataPacket can
//be provided, it is a packet with universe 0.
func (r *ReceiverSocket) SetOnChangeCallback(callback func(old DataPacket, new DataPacket)) {
	r.onChangeCallback = callback
}

//SetTimeoutCallback sets the callback for timeouts. The callback gets called everytime a timeout is
//recognized.
func (r *ReceiverSocket) SetTimeoutCallback(callback func(universe uint16)) {
	r.timeoutCallback = callback
}
