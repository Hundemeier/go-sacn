package receiver

import (
	"net"
	"time"

	"github.com/Hundemeier/go-sacn/packets"
)

func Receive(c chan<- packets.DataPacket, universe uint16) error {
	ServerAddr, err := net.ResolveUDPAddr("udp", ":5568")
	if err != nil {
		return err
	}

	ServerConn, err := net.ListenUDP("udp", ServerAddr)
	if err != nil {
		return err
	}
	defer ServerConn.Close()

	buf := make([]byte, 638)
	lastTime := time.Now()
	for {
		//Set the timout according to the E1.31 protocol
		ServerConn.SetDeadline(time.Now().Add(time.Millisecond * 2500))
		n, addr, _ := ServerConn.ReadFromUDP(buf) //n, addr, err
		if addr == nil {                          //Check if we had a timeout
			close(c) //Close the channel in case of a timeout
		}
		p, err := packets.NewDataPacketRaw(buf[0:n])
		if err != nil {
			break
		}
		if p.Universe() == universe {
			lastTime = time.Now()
			c <- p
		} else if time.Since(lastTime) >= time.Millisecond*2500 ||
			p.StreamTerminated() {
			close(c) //If we are here we had a atimeout on the universe
			//or a strem termination
		}
	}
	return nil
}
