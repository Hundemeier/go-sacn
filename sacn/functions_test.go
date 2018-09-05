package sacn

import (
	"bytes"
	"testing"
)

func TestCalculateFal(t *testing.T) {
	out := calculateFal(0x123)
	if out[0] != 0x71 || out[1] != 0x23 {
		t.Error("Wrong output of calculateFal!")
	}
}

func TestGetAsBytes16(t *testing.T) {
	out := getAsBytes16(0x1234)
	shouldBe := [...]byte{0x12, 0x34}
	if !bytes.Equal(out, shouldBe[:]) {
		t.Errorf("Wrong output! Was: %v; Should've been: %v", out, shouldBe)
	}
}

func TestGetAsBytes32(t *testing.T) {
	out := getAsBytes32(0x12345678)
	shouldBe := [...]byte{0x12, 0x34, 0x56, 0x78}
	if !bytes.Equal(out, shouldBe[:]) {
		t.Errorf("Wrong output! Was: %v; Should've been: %v", out, shouldBe)
	}
}

func TestGetAsUint32(t *testing.T) {
	out := getAsUint32([]byte{0x12, 0x34, 0x56, 0x78})
	shouldBe := uint32(0x12345678)
	if out != shouldBe {
		t.Errorf("Wrong output! Was: %v; Should've been: %v", out, shouldBe)
	}
}

func TestCalcMulticastAddr(t *testing.T) {
	out := calcMulticastAddr(257)
	shouldBe := "239.255.1.1"
	if out != shouldBe {
		t.Errorf("Wrong output! Was: %v; Should've been: %v", out, shouldBe)
	}
}

func TestCalcMulticastUdpAddr(t *testing.T) {
	out := calcMulticastUDPAddr(100)
	if out.Port != 5568 ||
		!out.IP.IsMulticast() ||
		out.IP.To4().String() != "239.255.0.100" {
		t.Errorf("IP should have been 239.255.0.100, was %v", out.IP.To4())
	}
}

func TestCheckSequ(t *testing.T) {
	if !checkSequ(12, 13) {
		t.Error("Sequence was one higher, should be good!")
	}
	if !checkSequ(100, 80) {
		t.Error("New sequence was 20 behind old one. Should be allowed!")
	}
	if checkSequ(100, 81) {
		t.Error("New sequence number of 81 with old 100 shouldn't be allowed!")
	}
	if checkSequ(255, 250) {
		t.Error("should not be allowed!")
	}
}
