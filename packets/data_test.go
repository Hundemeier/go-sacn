package packets

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestReplace(t *testing.T) {
	p := NewDataPacket()
	r := []byte{1, 2, 3, 4, 5, 6}
	p.replace(0, r)
	if !bytes.Equal(p.data[0:6], r) {
		t.Errorf("Wrong output! Was: %v; Should've been: %v", p.data[0:6], r)
	}
}

func TestSetCID(t *testing.T) {
	p := NewDataPacket()
	r := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	p.SetCID(r)
	if !bytes.Equal(p.data[22:38], r[:]) {
		t.Errorf("Wrong output! Was: %v; Should've been: %v", p.data[22:38], r)
	}
}

func TestCID(t *testing.T) {
	p := NewDataPacket()
	r := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	p.SetCID(r)
	o := p.CID()
	if !bytes.Equal(o[:], r[:]) {
		t.Errorf("Wrong output! Was: %v; Should've been: %v", o, r)
	}
}

func TestSetSourceName(t *testing.T) {
	p := NewDataPacket()
	s := "this is a test!"
	p.SetSourceName(s)
	o := p.data[44:108]
	r := [64]byte{}
	copy(r[:], []byte(s))
	if !bytes.Equal(o, r[:]) {
		t.Errorf("Wrong output! Was: %v; Should've been: %v", o, r)
	}
	p = NewDataPacket()
	s = "this is a test!"
	p.SetSourceName(s)
	s = "this should be different!"
	o = p.data[44:108]
	r = [64]byte{}
	copy(r[:], []byte(s))
	if bytes.Equal(o, r[:]) {
		t.Errorf("Wrong output! Was: %v; Should've been different!: %v", o, r)
	}
}

func TestSourceName(t *testing.T) {
	p := NewDataPacket()
	s := "Test source name!"
	p.SetSourceName(s)
	o := p.SourceName()
	if s != o {
		t.Errorf("Wrong output! Was: %v; Should've been different!: %v", o, s)
	}
}

func TestSetPriority(t *testing.T) {
	p := NewDataPacket()
	prio := byte(150)
	p.SetPriority(prio)
	o := p.data[108]
	if o != prio {
		t.Errorf("Wrong output! Was: %v; Should've been different!: %v", o, prio)
	}
	err := p.SetPriority(210)
	if err == nil {
		t.Error("Err was nil! Should have been an error!")
	}
}

func TestPriority(t *testing.T) {
	p := NewDataPacket()
	prio := byte(150)
	p.SetPriority(prio)
	o := p.Priority()
	if o != prio {
		t.Errorf("Wrong output! Was: %v; Should've been different!: %v", o, prio)
	}
}

func TestSetSyncAddress(t *testing.T) {
	p := NewDataPacket()
	sync := uint16(0x1234)
	p.SetSyncAddress(sync)
	o := p.data[109:111]
	if !bytes.Equal([]byte{0x12, 0x34}, o) {
		t.Errorf("Wrong output! Was: %v; Should've been different!: %v", o, sync)
	}
}

func TestSyncAddress(t *testing.T) {
	p := NewDataPacket()
	sync := uint16(0x1234)
	p.SetSyncAddress(sync)
	o := p.SyncAddress()
	if o != sync {
		t.Errorf("Wrong output! Was: %v; Should've been different!: %v", o, sync)
	}
}

func TestSetPreviewData(t *testing.T) {
	p := NewDataPacket()
	p.SetPreviewData(true)
	if !p.PreviewData() {
		t.Error("Preview data should have been true")
	}
	p.SetPreviewData(false)
	if p.PreviewData() {
		t.Error("Preview data should have been false")
	}
	p.SetPreviewData(false)
	if p.PreviewData() {
		t.Error("Preview data should have been false")
	}
}

func TestSetData(t *testing.T) {
	p := NewDataPacket()
	i := []byte{1, 2, 3, 4}
	p.SetData(i)
	if !bytes.Equal(i, p.Data()) {
		t.Error("DMX data was not set or getted properly!")
	}
	i = make([]byte, 600)
	for j := range i {
		i[j] = byte(rand.Uint32())
	}
	p.SetData(i)
	if !bytes.Equal(i[0:512], p.Data()) {
		t.Errorf("DMX data was not set or getted properly! Was: %v \nShouldbe: %v", p.Data(), i)
	}
}
