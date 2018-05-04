package tuntap

import (
	"fmt"
	"testing"
)

func TestTAPAdapter(t *testing.T) {
	tap, err := NewTAPAdapter(nil)

	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	if tap == nil {
		t.Fatal("expected not nil")
	}

	defer tap.Close()

	fmt.Println(tap.Interface().Name)
	fmt.Println(tap.Interface().Addrs())
	fmt.Println(tap.Interface().HardwareAddr)
}
