package sacn

import (
	"net"
	"time"
)

//Receive returns two chnnels: one for data and one for errors.
//the data channel only returns data from the universe that was given.
//parameters: universe: universe to listen on; bind: the interface on which the listener should bind to
func Receive(universe uint16, bind string) (<-chan DataPacket, <-chan error) {
	data := make(chan DataPacket)
	errch := make(chan error)

	//Receive the unprocessed data and sort out the ones with the corrct universe
	go func() {
		ServerAddr, err := net.ResolveUDPAddr("udp", bind+":5568")
		errToCh(err, errch)

		ServerConn, err := net.ListenUDP("udp", ServerAddr)
		errToCh(err, errch)
		defer ServerConn.Close()

		listenOn(ServerConn, data, errch, universe)
	}()

	return data, errch
}

//ReceiveMulticast is the same as normal Receive, but uses multicast instead.
//Depending on your OS you have to provide an Interface to bind to.
//Note: sometimes the packetloss with multicast is very high and so expect some unintentional
//timeouts and therefore closing channels
func ReceiveMulticast(universe uint16, ifi *net.Interface) (<-chan DataPacket, <-chan error) {
	data := make(chan DataPacket)
	errch := make(chan error)

	//Receive the unprocessed data and sort out the ones with the corrct universe
	go func() {
		ServerAddr, err := net.ResolveUDPAddr("udp", calcMulticastAddr(universe)+":5568")
		errToCh(err, errch)

		ServerConn, err := net.ListenMulticastUDP("udp", ifi, ServerAddr)
		errToCh(err, errch)
		defer ServerConn.Close()
		//some testing revealed that sometimes in multicast-use packets were lost
		//this should help out the problem
		ServerConn.SetReadBuffer(3 * 638)
		listenOn(ServerConn, data, errch, universe)
	}()

	return data, errch
}

func listenOn(conn *net.UDPConn, data chan<- DataPacket, errch chan<- error, universe uint16) {
	const timeoutMs = 2500
	buf := make([]byte, 638)
	//store the lasttime a packet on the universe was received
	lastTime := time.Now()
	for {
		//Set the timout according to the E1.31 protocol (plus 200ms)
		conn.SetDeadline(time.Now().Add(time.Millisecond * timeoutMs))
		n, addr, _ := conn.ReadFromUDP(buf) //n, addr, err
		if addr == nil {                    //Check if we had a timeout
			break //escape the for loop
		}
		p, err := NewDataPacketRaw(buf[0:n])
		errToCh(err, errch)
		if p.Universe() == universe {
			//received a packet on the universe to listen on
			lastTime = time.Now()
			data <- p

			if p.StreamTerminated() {
				//if the stream termination bit was set, we escape the loop to close the channel
				break
			}
		} else if time.Since(lastTime) > timeoutMs*time.Millisecond {
			break
		}
	}
	close(errch)
	close(data)
}

func errToCh(err error, ch chan<- error) {
	if err != nil {
		ch <- err
	}
}
