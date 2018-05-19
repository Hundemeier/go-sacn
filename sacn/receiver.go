package sacn

import (
	"errors"
	"net"
	"time"
)

//Set the timout according to the E1.31 protocol
const timeoutMs = 2500

var socketRecv *net.UDPConn

//Receiver is for holding the channels for the data and the errors
type Receiver struct {
	DataChan chan DataPacket
	ErrChan  chan error
	stopChan chan struct{}
}

//NewReceiver returns a new Receiver object that can be used to listen with it
func newReceiver() Receiver {
	return Receiver{
		DataChan: make(chan DataPacket),
		ErrChan:  make(chan error),
		stopChan: make(chan struct{}),
	}
}

//Stop sends a stop signal to the listener and ends the transmission of data or errors in the channels
func (r *Receiver) Stop() {
	close(r.stopChan)
}

//Receive returns two chnnels: one for data and one for errors.
//the data channel only returns data from the universe that was given.
//parameters: universe: universe to listen on; bind: the interface on which the listener should bind to.
//This Receiver checks for out-of-order packets and sorts out packets with too low priority.
//Note: if there are two sources with the same highest priority, there will be send a
//"sources exceeded" error in the error channel.
//Furthermore: through the channel only changed data will be send. So the sequence numbers may not be in order.
func Receive(universe uint16, bind string) (Receiver, error) {
	r := newReceiver()
	var ServerConn *net.UDPConn
	if socketRecv == nil {
		ServerAddr, err := net.ResolveUDPAddr("udp", bind+":5568")
		errToCh(err, r.ErrChan)

		ServerConn, err = net.ListenUDP("udp", ServerAddr)
		errToCh(err, r.ErrChan)
		//do not start goroutine when the socket could not be created
		if err != nil {
			close(r.stopChan) // close the receiver when returning, so that it has no function
			close(r.DataChan)
			close(r.ErrChan)
			return r, err
		}
		socketRecv = ServerConn
	} else {
		ServerConn = socketRecv
	}

	//Receive the unprocessed data and sort out the ones with the corrct universe
	go func() {
		defer ServerConn.Close()
		listenOn(ServerConn, universe, r.breakCondition, r.ErrChan, r.DataChan)
	}()
	return r, nil
}

//ReceiveMulticast is the same as normal Receive, but uses multicast instead.
//Depending on your OS you have to provide an Interface to bind to.
//This Receiver checks for out-of-order packets and sorts out packets with too low priority.
//Note: if there are two sources with the same highest priority, there will be send a
//"sources exceeded" error in the error channel.
//Furthermore: through the channel only changed data will be send. So the sequence numbers may not be in order.
//Note: sometimes the packetloss with multicast can be very high and so expect some unintentional
//timeouts and therefore closing channels
func ReceiveMulticast(universe uint16, ifi *net.Interface) (Receiver, error) {
	r := newReceiver()
	var ServerConn *net.UDPConn
	if socketRecv == nil {
		ServerAddr, err := net.ResolveUDPAddr("udp", calcMulticastAddr(universe)+":5568")
		errToCh(err, r.ErrChan)

		ServerConn, err = net.ListenMulticastUDP("udp", ifi, ServerAddr)
		errToCh(err, r.ErrChan)
		//do not start goroutine when the socket could not be created
		if err != nil {
			close(r.stopChan) // close the receiver when returning, so that it has no function
			close(r.DataChan)
			close(r.ErrChan)
			return r, err
		}
		socketRecv = ServerConn
	} else {
		ServerConn = socketRecv
	}

	//Receive the unprocessed data and sort out the ones with the corrct universe
	go func() {
		defer ServerConn.Close()
		//some testing revealed that sometimes in multicast-use packets were lost
		//this should help out the problem
		ServerConn.SetReadBuffer(3 * 638)
		listenOn(ServerConn, universe, r.breakCondition, r.ErrChan, r.DataChan)
	}()
	return r, nil
}

func (r *Receiver) breakCondition() bool {
	select {
	case <-r.stopChan:
		return true //break if we had a stop signal from the stopChannel
	default:
		return false
	}
}

//breakCondition: if this function return true, the listening will be breaked
func listenOn(conn *net.UDPConn, universe uint16, breakCondition func() bool, errChan chan<- error, dataChan chan<- DataPacket) {
	buf := make([]byte, 638)
	//store the lasttime a packet on the universe was received
	lastTime := time.Now()
	//store the sequence number of the last valid packet
	lastSequ := byte(0)
	//store the last DMX data to check if it was changed
	var lastData []byte
	//sources map that stores all sources that have used this universe
	m := make(map[[16]byte]source)
F:
	for {
		if breakCondition() {
			break F
		}

		conn.SetDeadline(time.Now().Add(time.Millisecond * timeoutMs))
		n, addr, _ := conn.ReadFromUDP(buf) //n, addr, err
		if addr == nil {                    //Check if we had a timeout
			//that means we did not receive a packet in 2,5s at all
			errChan <- errors.New("timeout")
			continue
		}
		p, err := NewDataPacketRaw(buf[0:n])
		errToCh(err, errChan)
		if p.Universe() == universe {
			updateSourcesMap(m, p)
			tmp := getAllowedSources(m)

			//if the length of allowed sources is greater than 1, we have the situation of
			//multiple sources transmitting on the same priority, so we send sources exceeded to the errchan
			if len(tmp) > 1 {
				errChan <- errors.New("sources exceeded")
				continue //skip all steps down
			}
			//if the source of this packet is in the allowed sources list, let this packet pass
			if _, ok := tmp[p.CID()]; ok {
				//check the sequence
				if !checkSequ(lastSequ, p.Sequence()) {
					continue //if the sequence is not good, discard this packet
				}
				lastSequ = p.Sequence()
				//check if the data was changed
				if !equalData(lastData, p.Data()) {
					dataChan <- p
					//make a copy as lastData, otherwise it will be a reference
					lastData = append([]byte(nil), p.Data()...)
				}
			}

			if p.StreamTerminated() {
				//if the stream termination bit was set, we escape the loop to close the channel
				//r.ErrChan <- errors.New("stream terminated")
				continue
			}
		} else if time.Since(lastTime) > timeoutMs*time.Millisecond {
			//timeout of our universe, this if is needed, because we may receive packets from other
			//universes but we have to listen only for ours
			//r.ErrChan <- errors.New("timeout")
			continue
		}
	}
	close(errChan)
	close(dataChan)
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

func checkSequ(old, new byte) bool {
	//calculate in int
	tmp := int(new) - int(old)
	if tmp <= 0 && tmp > -20 {
		return false
	}
	return true
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
