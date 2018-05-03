# go-sacn
This is a sACN implementation for golang. It is based on the E1.31 protocol by the ESTA. 
A copy can be found [here][e1.31].

This is by no means a full implementation yet, but may be in the future.
If you want to see a full DMX package, see the 
[OLA](http://opendmx.net/index.php/Open_Lighting_Architecture) project.

## Receiving
**BETA!**

This is currently the only implemented feature. The simplest way to receive sACN packets is 
to use `sacn.Receiver`.

The receiver checks for out-of-order packets (inspecting the sequence number) and sorts for priority.
The channel only gets used for changed DMX data, so it behaves like a change listener.
Note: if two or more sources are transmitting on the same universe with the same priority, 
there will be errors send through the error channel with "sources exceeded" as text. 
No data will be transmitted through the data channel.

Synchronization must be implemented in your program, but currently there is no way to receive
the sACN sync-packets. This feature may come in a future version.

Please note: This implementation is subjected to change!


### Unicast

Example for simple unicast listener:
``` go
package main

import (
	"fmt"

	"github.com/Hundemeier/go-sacn/sacn"
)

func main() {
	recv := sacn.NewReceiver()
	recv.Receive(1, "") //receive on the universe 1 and bind to all interfaces
	go func() {         //print every error that occurs
		for i := range recv.ErrChan {
			fmt.Println(i)
		}
	}()
	for j := range recv.DataChan {
		fmt.Println(j.Sequence())
	}
	//recv.Stop() //use this to stop the receiving of messages and close the channels
	//Note: This does not stop immediately the channels, worst case: it takes 2,5 seconds
}
```

### Multicast

This `Receiver` also uses multicast groups to receive its data. Unicast packets that are received
are also processed like the normal unicast receiver. Depending on your operating system, you might can 
provide `nil` as an interface, sometimes you have to use a dedicated interface, to get multicast working.
Windows needs an interface and linux generally not.

Example for multicast use:
``` go
package main

import (
	"fmt"
	"net"

	"github.com/Hundemeier/go-sacn/sacn"
)

func main() {
	recv := sacn.NewReceiver()
	//get the interface we use to listen via multicast
	//see the net package for more information
	ifi, err := net.InterfaceByName("WLAN")
	if err != nil {
		panic(err)
	}
	//the use of a dedicated interface is dependend on your OS
	//if you use Windows you have to provide an interface, on other OS you might not
	recv.ReceiveMulticast(1, ifi) //receive on the universe 1 and bind to the interface
	go func() {                   //print every error that occurs
		for i := range recv.ErrChan {
			fmt.Println(i)
		}
	}()
	for j := range recv.DataChan {
		fmt.Println(j.Data())
	}
}

```

### Stoping

You can stop the receiving of packets on a Receiver via `receiver.Stop()`. 
Please note that it can take up to 2,5s to stop the receiving and close all channels.
If you have stoped a receiver once, you can not start listening again. You have to create a 
new `Receiver` object via `sacn.NewReceiver()`.

[e1.31]: http://tsp.esta.org/tsp/documents/docs/E1-31-2016.pdf