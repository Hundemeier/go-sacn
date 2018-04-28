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
		if err != nil {
			errch <- err
		}

		ServerConn, err := net.ListenUDP("udp", ServerAddr)
		if err != nil {
			errch <- err
		}
		defer ServerConn.Close()

		buf := make([]byte, 638)
		//store the lasttime a packet on the universe was received
		lastTime := time.Now()
		for {
			//Set the timout according to the E1.31 protocol
			ServerConn.SetDeadline(time.Now().Add(time.Millisecond * 2500))
			n, addr, _ := ServerConn.ReadFromUDP(buf) //n, addr, err
			if addr == nil {                          //Check if we had a timeout
				break //escape the for loop
			}
			p, err := NewDataPacketRaw(buf[0:n])
			if err != nil {
				errch <- err
			}
			if p.Universe() == universe {
				//received a packet on the universe to listen on
				lastTime = time.Now()
				data <- p

				if p.StreamTerminated() {
					//if the stream termination bit was set, we escape the loop to close the channel
					break
				}
			} else if time.Since(lastTime) > 2500*time.Millisecond {
				break
			}
		}
		close(errch)
		close(data)
	}()

	return data, errch
}
