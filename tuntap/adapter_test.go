package tuntap

import (
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func closeAndCheck(t *testing.T, c io.Closer) {
	t.Helper()

	if err := c.Close(); err != nil {
		t.Fatalf("failed to close: %s", err)
	}
}
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

	defer closeAndCheck(t, tap)

	fmt.Println(tap.Interface().Addrs())
	buf := make([]byte, tap.Interface().MTU)
	if n, err := tap.Read(buf); err == nil {
		fmt.Println(hex.EncodeToString(buf[:n]))
		packet := gopacket.NewPacket(buf[:n], layers.LayerTypeEthernet, gopacket.DecodeOptions{Lazy: true, NoCopy: true})

		for i, layer := range packet.Layers() {
			fmt.Println(i, layer.LayerType())
		}
	}
	time.Sleep(time.Millisecond * 1000)
}

func TestTunAdapter(t *testing.T) {
	config := &TunAdapterConfig{
		IPv4: &net.IPNet{
			IP:   net.ParseIP("192.168.11.10"),
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

	defer closeAndCheck(t, tun)

	fmt.Println(tun.Interface().Addrs())
	buf := make([]byte, tun.Interface().MTU)
	if n, err := tun.Read(buf); err == nil {
		fmt.Println(hex.EncodeToString(buf[:n]))
		packet := gopacket.NewPacket(buf[:n], layers.LayerTypeIPv4, gopacket.DecodeOptions{Lazy: true, NoCopy: true})

		for i, layer := range packet.Layers() {
			fmt.Println(i, layer.LayerType(), hex.EncodeToString(layer.LayerContents()))
		}
	}
	time.Sleep(time.Millisecond * 1000)
}
