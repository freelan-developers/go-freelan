package tuntap

import (
	"fmt"
	"net"
	"testing"
	"time"
)

func TestTAPAdapter(t *testing.T) {
	config := &TAPAdapterConfig{
		IPv4: &net.IPNet{
			IP:   net.ParseIP("10.1.0.1"),
			Mask: net.CIDRMask(24, 32),
		},
	}
	tap, err := NewTAPAdapter(config)

	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	if tap == nil {
		t.Fatal("expected not nil")
	}

	defer tap.Close()

	for i := 0; i < 10; i++ {
		fmt.Println(tap.Interface().Addrs())
		buf := make([]byte, tap.Interface().MTU)
		n, err := tap.Read(buf)
		fmt.Println(n, err)
		time.Sleep(time.Millisecond * 1000)
	}
}
