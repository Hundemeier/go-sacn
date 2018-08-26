/*Package sacn is a simple sACN implementation. The standard can be obtained here: http://tsp.esta.org/tsp/documents/docs/E1-31-2016.pdf

This is by no means a full implementation yet, but may be in the future.
If you want to see a full DMX package, see the
http://opendmx.net/index.php/Open_Lighting_Architecture project.

Receiving

The simplest way to receive sACN packets is to use `sacn.NewReceiverSocket`.

The receiver checks for out-of-order packets (inspecting the sequence number) and sorts for priority.
Synchronization must be implemented in the callers program, but currently there is no way to receive
the sACN sync-packets. This feature may come in a future version.

This `sacn.ReceiverSocket` can use multicast groups to receive its data. Unicast packets that are received
are also processed like the normal unicast receiver. Depending on your operating system, you might can
provide `nil` as an interface, sometimes you have to use a dedicated interface, to get multicast working.
Windows needs an interface and Linux generally not.

Note that the network infrastructure has to be multicast ready and that on some networks the delay of
packets will increase. Also the packet loss can be higher if multicast is chosen
(This is often a problem when WLAN is used). This can cause unintentional timeouts, if the sources
are only transmitting every 2 seconds (like grandMA2 consoles).

Transmitting

To transmitt DMX data, you have to initialize a `Transmitter` object. This handles all the protocol
specific actions (currently not all). You can activate universes, if you wish to send out data.
Then you can use a channel for 512-byte arrays to transmitt them over the network.

There are two different types of addressing the receiver: unicast and multicast.
When using multicast, note that you have to provide a bind address on some operating systems
(eg Windows). You can use both at the same time and any number of unicast addresses.
To set wether multicast should be used, call `transmitter.SetMulticast(<universe>, <bool>)`.
You can set multiple unicast destinations as slice via
`transmitter.SetDestinations(<universe>, <[]string>)`.
Note that any existing destinations will be overwritten. If you want to append a destination, you
can use `transmitter.Destination(<universe>)` which returns a deep copy of the used net.UDPAddr
objects.

Example

	package main

	import (
		"log"
		"math/rand"
		"time"

		"github.com/Hundemeier/go-sacn/sacn"
	)

	func main() {
		//instead of "" you could provide an ip-address that the socket should bind to
		trans, err := sacn.NewTransmitter("", [16]byte{1, 2, 3}, "test")
		if err != nil {
			log.Fatal(err)
		}

		//activates the first universe
		ch, err := trans.Activate(1)
		if err != nil {
			log.Fatal(err)
		}
		//deactivate the channel on exit
		defer close(ch)

		//set a unicast destination, and/or use multicast
		trans.SetMulticast(1, true)//this specific setup will not multicast on windows,
		//because no bind address was provided

		//set some example ip-addresses
		trans.SetDestinations(1, []string{"192.168.1.13", "192.168.1.1"})

		//send some random data for 10 seconds
		for i := 0; i < 20; i++ {
			ch <- [512]byte{byte(rand.Int()), byte(i & 0xFF)}
			time.Sleep(500 * time.Millisecond)
		}
	}*/
package sacn
