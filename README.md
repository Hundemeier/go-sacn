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

`Receiver` takes two arguments: a channel for `packets.DataPacket`s and the universe on 
which the function should listen on. The channel returns every packet that is received on the given 
universe. If a timeout occurred (2,5s no message) or a packet with the StreamTermination-bit was set, 
the channel will close.

Please note: This implementation is subjected to change!

Example:
``` go
package main

import (
	"fmt"

	"github.com/Hundemeier/go-sacn/packets"
	"github.com/Hundemeier/go-sacn/receiver"
)

func main() {
	ch := make(chan packets.DataPacket)
	go receiver.Receive(ch, 1)

	for i := range ch {
		fmt.Println(i.Data())
	}
}
```

[e1.31]: http://tsp.esta.org/tsp/documents/docs/E1-31-2016.pdf