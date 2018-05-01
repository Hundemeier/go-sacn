package sacn

import (
	"net"
	"time"
)

//Set the timout according to the E1.31 protocol
const timeoutMs = 2500

//Receive returns two chnnels: one for data and one for errors.
//the data channel only returns data from the universe that was given.
//parameters: universe: universe to listen on; bind: the interface on which the listener should bind to.
//This Receiver checks for out-of-order packets and sorts out packets with too low priority.
//Note: if there are two sources with the same highest priority, all their data will get through the channel.
//Furthermore: through the channel only changed data will be send. So the sequence numbers may not be in order.
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
//This Receiver checks for out-of-order packets and sorts out packets with too low priority.
//Note: if there are two sources with the same highest priority, all their data will get through the channel.
//Furthermore: through the channel only changed data will be send. So the sequence numbers may not be in order.
//Note: sometimes the packetloss with multicast can be very high and so expect some unintentional
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
	buf := make([]byte, 638)
	//store the lasttime a packet on the universe was received
	lastTime := time.Now()
	//sources map that stores all sources that have used this universe
	m := make(map[[16]byte]source)
	for {
		conn.SetDeadline(time.Now().Add(time.Millisecond * timeoutMs))
		n, addr, _ := conn.ReadFromUDP(buf) //n, addr, err
		if addr == nil {                    //Check if we had a timeout
			break //escape the for loop
		}
		p, err := NewDataPacketRaw(buf[0:n])
		errToCh(err, errch)
		if p.Universe() == universe {
			updateSourcesMap(m, p)
			tmp := getAllowedSources(m)

			//if the length of allowed sources is greater than 1, we have the situation of
			//multiple sources transmitting on the same priority, so we send all apckets to the channel
			if len(tmp) > 1 {
				data <- p
				continue //skip all steps down
			}
			//if the source of this packet is in the allowed sources list, let this packet pass
			if _, ok := tmp[p.CID()]; ok {
				//TODO: check for the update of the dmx data-------------------------------------------------------
				//TODO: check the sequence number (new function?)
				data <- p
			}

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

//Helping functions and structs for storing source information
type source struct {
	//store the last time this source occurs
	lastTime time.Time
	//store the last time this priority occurs
	lastTimeHighPrio time.Time
	//store the highest priority from this source that is currently sended out
	highestPrio byte
}

//updates the map according to current time and the given packet
func updateSourcesMap(m map[[16]byte]source, p DataPacket) {
	//go through the map and update the entries
	for key, value := range m {
		if key == p.CID() {
			//We have the entry that is the same source as the one from the packet
			//update time
			value.lastTime = time.Now()
			//Check if the priority has been changed
			if value.highestPrio < p.Priority() {
				//priority is increased
				value.highestPrio = p.Priority()
				value.lastTimeHighPrio = time.Now()
			} else if value.highestPrio == p.Priority() {
				//priority stays the same, so update the time
				value.lastTimeHighPrio = time.Now()
			} else if value.highestPrio > p.Priority() {
				//the stored priority is lower than the packet's one
				//check for timeout of the highest priority
				if time.Since(value.lastTimeHighPrio) > timeoutMs*time.Millisecond {
					//if the highest priority is timeouted decrease the current priority
					value.highestPrio = p.Priority()
					value.lastTimeHighPrio = time.Now()
				}
			}
		} else {
			//If the source timeouted, delete it
			if time.Since(value.lastTime) > timeoutMs*time.Millisecond {
				delete(m, key)
			}
		}
	}
	//check if the source is new
	_, ok := m[p.CID()]
	if !ok { //if the source is new create a new entry
		m[p.CID()] = source{
			lastTime:         time.Now(),
			lastTimeHighPrio: time.Now(),
			highestPrio:      p.Priority(),
		}
	}
}

func getAllowedSources(m map[[16]byte]source) map[[16]byte]struct{} {
	//filter for the highest priority
	highestPrio := byte(0)
	for _, value := range m {
		if value.highestPrio > highestPrio {
			highestPrio = value.highestPrio
		}
	}
	//now get all sources with the highest priority
	new := make(map[[16]byte]struct{})
	//and store them in this set
	for key, value := range m {
		if value.highestPrio == highestPrio {
			new[key] = struct{}{}
		}
	}
	return new
}
