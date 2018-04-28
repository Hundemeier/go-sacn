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
		fmt.Println(i)
	}

	// d := []byte{1, 2, 3, 4, 5, 6}
	// fmt.Println(d[0:0])

	// a := []byte{1, 2, 3, 4, 5, 6}
	// b := []byte{7, 8, 9}
	// i := 1
	// a = append(a[:i], append(b, a[len(b)+i:]...)...)
	// fmt.Println(a)
}
