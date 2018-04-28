package main

import (
	"fmt"

	"github.com/Hundemeier/go-sacn/sacn"
)

func main() {
	ch := make(chan sacn.DataPacket)
	go sacn.Receive(ch, 1)

	for i := range ch {
		fmt.Println(i.Data())
	}

	// d := []byte{1, 2, 3, 4, 5, 6}
	// fmt.Println(d[0:0])

	// a := []byte{1, 2, 3, 4, 5, 6}
	// b := []byte{7, 8, 9}
	// i := 1
	// a = append(a[:i], append(b, a[len(b)+i:]...)...)
	// fmt.Println(a)
}
