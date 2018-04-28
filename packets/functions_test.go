package packets

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
