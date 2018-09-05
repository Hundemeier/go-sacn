package sacn

import (
	"bytes"
	"time"
)

//the listener is responsible for listening on the UDP socket and parsing the incoming data.
//It dispatches the received packets to the corresponding handlers.
func (r *ReceiverSocket) startListener() {
	go func() {
		buf := make([]byte, 638)
	Loop:
		for {
			select {
			case <-r.stopListener:
				break Loop //break if we had a stop signal from the stopChannel
			default:
			}

			r.socket.SetDeadline(time.Now().Add(time.Millisecond * timeoutMs))
			n, _, addr, _ := r.socket.ReadFrom(buf) //n, ControlMessage, addr, err
			if addr == nil {                        //Check if we had a timeout
				//that means we did not receive a packet in 2,5s at all
				r.checkForTimeouts()
			}
			p, err := NewDataPacketRaw(buf[0:n])
			if err != nil {
				continue //if the packet could not be parsed, just skip it
			}
			//send the packet to the responding handler and the other are getting nil
			r.handle(p)
		}
		r.socket.Close()     //close the channel, if the listener is finished
		r.stopListener = nil //set the channel to nil, so it can be used as indicator if the routine is running
	}()
}

//the handler is responsible for checking all necessary things to decide if callbacks should be invoked
func (r *ReceiverSocket) handle(p DataPacket) {
	r.checkForTimeouts()
	//check if we had a change in priority to the last data we received on the universe
	last, ok := r.lastDatas[p.Universe()]
	if ok {
		//check if the last packet is too long ago, then we do not have to check all other things
		if time.Since(last.lastTime) > time.Millisecond*timeoutMs {
			//invoke callback and store the new packet and time
			if !bytes.Equal(last.lastPacket.Data(), p.Data()) {
				r.invokeCallback(p)
			}
			r.storeLastPacket(p)
			return // we are finished with this packet
		}
		//we have last data for this universe, so check the priority
		if last.lastPacket.Priority() == p.Priority() {
			//we have the same priority
			//check sequence:
			if checkSequ(last.lastPacket.Sequence(), p.Sequence()) {
				//sequence is good:; check if the data has changed. If so, then invoke callback
				if !bytes.Equal(last.lastPacket.Data(), p.Data()) {
					r.invokeCallback(p)
				}
				r.storeLastPacket(p)
			}
		} else if last.lastPacket.Priority() < p.Priority() {
			//priority is higher: invoke callback on data change
			if !bytes.Equal(last.lastPacket.Data(), p.Data()) {
				r.invokeCallback(p)
			}
			//store the new packet regardless
			r.storeLastPacket(p)
		}
	} else {
		//store new packet and invoke callback, because we never had data on this one
		r.invokeCallback(p)
		r.storeLastPacket(p)
	}
}

//invokeCallback calls the callback if it is present.
func (r *ReceiverSocket) invokeCallback(new DataPacket) {
	oldData, ok := r.lastDatas[new.Universe()]
	var old DataPacket
	if ok {
		old = oldData.lastPacket
	} else {
		old = NewDataPacket()
	}
	if r.onChangeCallback != nil {
		go r.onChangeCallback(old, new)
	}
}

//storeLastPacket stores the packet in the lastDatas store
func (r *ReceiverSocket) storeLastPacket(p DataPacket) {
	r.lastDatas[p.Universe()] = lastData{
		lastPacket: p.copy(),
		lastTime:   time.Now(),
	}
	r.timeoutCalled[p.Universe()] = false
}

//checkForTimeouts checks all last data if a universe had a timeout. Calls the timeoutCallback.
func (r *ReceiverSocket) checkForTimeouts() {
	for univ, last := range r.lastDatas {
		if time.Since(last.lastTime) > time.Millisecond*timeoutMs {
			//timeout
			if r.timeoutCallback != nil && !r.timeoutCalled[univ] {
				go r.timeoutCallback(univ)
				r.timeoutCalled[univ] = true
			}
		}
	}
}
