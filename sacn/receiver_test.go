package sacn

import (
	"reflect"
	"testing"
)

func TestGetAllowedSources(t *testing.T) {
	m := make(map[[16]byte]source)
	m[[16]byte{1}] = source{
		highestPrio: 100,
	}
	m[[16]byte{2}] = source{
		highestPrio: 100,
	}
	m[[16]byte{4}] = source{
		highestPrio: 50,
	}
	m[[16]byte{3}] = source{
		highestPrio: 70,
	}
	out := getAllowedSources(m)
	shouldBe := make(map[[16]byte]struct{})
	shouldBe[[16]byte{1}] = struct{}{}
	shouldBe[[16]byte{2}] = struct{}{}
	if !reflect.DeepEqual(shouldBe, out) {
		t.Errorf("Output: %v \nShould have been: %v", out, shouldBe)
	}
}

func TestUpdateSourcesMap(t *testing.T) {
	firstSource := [16]byte{1}
	m := make(map[[16]byte]source)
	p := NewDataPacket()
	p.SetPriority(120)
	p.SetCID(firstSource)
	shouldBe := make(map[[16]byte]source)
	shouldBe[firstSource] = source{
		highestPrio: 120,
	}
	updateSourcesMap(m, p)
	if m[firstSource].highestPrio != shouldBe[firstSource].highestPrio ||
		m[firstSource].lastTime != m[firstSource].lastTimeHighPrio {
		t.Errorf("The priorities are not the same or the last times are not the same!\nOut: %v\nShouldbe:%v", m, shouldBe)
	}
}
