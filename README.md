# go-sacn
This is a sACN implementation for golang. It is based on the E1.31 protocol by the ESTA. 
A copy can be found [here][e1.31].

This is by no means a full implementation yet, but may be in the future.
If you want to see a full DMX package, see the 
[OLA](http://opendmx.net/index.php/Open_Lighting_Architecture) project.

## Receiving
**BETA!**

This is currently the only implemented feature. The simplest way to receive sACN packets is 
to use `receiver.Receiver`.

`Receiver` takes the universe on which the function should listen on and an interface name to bind to.
If you want to listen on all interfaces use an empty string "". 
It returns a data channel and an error channel.
The data channel returns every packet that is received on the given universe. If a timeout occurred 
(2,5s no message) or a packet with the StreamTermination-bit was set, the channel will close.

The receiver checks for out-of-order packets (inspecting the sequence number) and sorts for priority.
The channel only gets used for changed DMX data, so it behaves like a change listener.
Note: this behaiviour is bypassed if there are two or more sources transmitting on the same universe 
with the same highest priority. Then ALL packets are getting send through the channel (although 
packets with too low priority will be skipped). Then your program is responsible for sorting or 
alerting the user.

Synchronization must be implemented in your program, but currently there is no way to receive
the sACN sync-packets. This feature may come in a future version.

Please note: This implementation is subjected to change!

Example:
``` go
package main

import (
	"fmt"

	"github.com/Hundemeier/go-sacn/sacn"
)

func main() {
	//listen on universe 1 and bind to all interfaces
	ch, _ := sacn.Receive(1, "") //returns a data channel and an error channel
	//in this example we dont wnat to mess with errors
	for i := range ch {
		//print as long as we receive data
		fmt.Println(i.Data())
	}
	//now the channel was closed and the program exits
	//the channel was closed, because the universe timeouted and no more data was received
	//if we want we cloud start listening again. Depends on the use case
}
```

[e1.31]: http://tsp.esta.org/tsp/documents/docs/E1-31-2016.pdf