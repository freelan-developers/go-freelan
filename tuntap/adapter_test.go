package tuntap

import (
	"fmt"
	"net"
	"testing"
	"time"
)

func TestTapAdapter(t *testing.T) {
	config := &TapAdapterConfig{
		IPv4: &net.IPNet{
			IP:   net.ParseIP("192.168.10.10"),
			Mask: net.CIDRMask(24, 32),
		},
	}

	tap, err := NewTapAdapter(config)

	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	if tap == nil {
		t.Fatal("expected not nil")
	}

	defer tap.Close()

	fmt.Println(tap.Interface().Addrs())
	buf := make([]byte, tap.Interface().MTU)
	n, err := tap.Read(buf)
	fmt.Println(n, err)
	fmt.Printf("%0x\n", buf[0:n])
	time.Sleep(time.Millisecond * 1000)
}

func TestTunAdapter(t *testing.T) {
	config := &TunAdapterConfig{
		IPv4: &net.IPNet{
			IP:   net.ParseIP("192.168.10.10"),
			Mask: net.CIDRMask(24, 32),
		},
	}

	tun, err := NewTunAdapter(config)

	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	if tun == nil {
		t.Fatal("expected not nil")
	}

	defer tun.Close()

	fmt.Println(tun.Interface().Addrs())
	buf := make([]byte, tun.Interface().MTU)
	n, err := tun.Read(buf)
	fmt.Println(n, err)
	fmt.Printf("%0x\n", buf[0:n])
	time.Sleep(time.Millisecond * 1000)
}
