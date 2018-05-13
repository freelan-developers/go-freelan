package tuntap

import "testing"

func TestTapCtlCode(t *testing.T) {
	var expected uint32 = 0x00220018
	value := tapCtlCode(6)

	if value != expected {
		t.Errorf("expected: %08x\ngot     : %08x", expected, value)
	}
}
