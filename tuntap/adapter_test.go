package tuntap

import (
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	ping "github.com/sparrc/go-ping"
)

func closeAndCheck(t *testing.T, c io.Closer) {
	t.Helper()

	if err := c.Close(); err != nil {
		t.Fatalf("failed to close: %s", err)
	}
}

func skipIfNotRoot(t *testing.T) {
	t.Helper()

	if os.Geteuid() > 0 {
		t.Skip("must be root")
	}
}

func testPing(t *testing.T, addr string) <-chan bool {
	t.Helper()

	pinger, err := ping.NewPinger(addr)

	if err != nil {
		t.Fatalf("failed to instanciate a pinger: %s", err)
	}

	ch := make(chan bool, 1)

	pinger.Count = 1
	pinger.Timeout = time.Second * 3
	pinger.SetPrivileged(true)
	pinger.OnFinish = func(s *ping.Statistics) {
		ch <- (s.PacketsRecv > 0)
	}

	go func() {
		time.Sleep(pinger.Timeout)
		ch <- false
	}()

	go pinger.Run()

	return ch
}

func TestTapAdapter(t *testing.T) {
	skipIfNotRoot(t)

	config := &AdapterConfig{
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

	ch := testPing(t, "192.168.10.1")
	buf := make([]byte, tap.Interface().MTU)

	for {
		n, err := tap.Read(buf)

		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		packet := gopacket.NewPacket(
			buf[:n],
			layers.LayerTypeEthernet,
			gopacket.DecodeOptions{Lazy: true, NoCopy: true},
		)

		ethernet, _ := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
		ipv4, _ := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
		icmp, _ := packet.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4)

		if icmp != nil {
			ethernetResp := &layers.Ethernet{
				SrcMAC:       ethernet.DstMAC,
				DstMAC:       ethernet.SrcMAC,
				EthernetType: ethernet.EthernetType,
			}
			ipv4Resp := &layers.IPv4{
				Version:    ipv4.Version,
				IHL:        ipv4.IHL,
				TOS:        ipv4.TOS,
				Id:         ipv4.Id,
				Flags:      ipv4.Flags,
				FragOffset: ipv4.FragOffset,
				TTL:        ipv4.TTL,
				Protocol:   ipv4.Protocol,
				SrcIP:      ipv4.DstIP,
				DstIP:      ipv4.SrcIP,
				Options:    ipv4.Options,
				Padding:    ipv4.Padding,
			}
			icmpResp := &layers.ICMPv4{
				TypeCode: layers.ICMPv4TypeEchoReply,
				Id:       icmp.Id,
				Seq:      icmp.Seq,
			}
			payload := gopacket.Payload(icmp.LayerPayload())

			sbuf := gopacket.NewSerializeBuffer()
			options := gopacket.SerializeOptions{
				ComputeChecksums: true,
				FixLengths:       true,
			}

			if err = gopacket.SerializeLayers(sbuf, options, ethernetResp, ipv4Resp, icmpResp, payload); err != nil {
				t.Fatalf("expected no error but got: %s", err)
			}

			if n, err = tap.Write(sbuf.Bytes()); err != nil {
				t.Fatalf("expected no error but got: %s", err)
			}

			break
		}
	}

	if ok := <-ch; !ok {
		t.Errorf("received no response")
	}
}

func TestTunAdapter(t *testing.T) {
	skipIfNotRoot(t)

	config := &AdapterConfig{
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

	ch := testPing(t, "192.168.11.1")
	buf := make([]byte, tun.Interface().MTU)

	for {
		n, err := tun.Read(buf)

		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		layerType := layers.LayerTypeIPv4

		if (buf[0]&0xf0)>>4 == 6 {
			layerType = layers.LayerTypeIPv6
		}

		packet := gopacket.NewPacket(
			buf[:n],
			layerType,
			gopacket.DecodeOptions{Lazy: true, NoCopy: true},
		)

		ipv4, _ := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
		icmp, _ := packet.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4)

		if icmp != nil {
			ipv4Resp := &layers.IPv4{
				Version:    ipv4.Version,
				IHL:        ipv4.IHL,
				TOS:        ipv4.TOS,
				Id:         ipv4.Id,
				Flags:      ipv4.Flags,
				FragOffset: ipv4.FragOffset,
				TTL:        ipv4.TTL,
				Protocol:   ipv4.Protocol,
				SrcIP:      ipv4.DstIP,
				DstIP:      ipv4.SrcIP,
				Options:    ipv4.Options,
				Padding:    ipv4.Padding,
			}
			icmpResp := &layers.ICMPv4{
				TypeCode: layers.ICMPv4TypeEchoReply,
				Id:       icmp.Id,
				Seq:      icmp.Seq,
			}
			payload := gopacket.Payload(icmp.LayerPayload())

			sbuf := gopacket.NewSerializeBuffer()
			options := gopacket.SerializeOptions{
				ComputeChecksums: true,
				FixLengths:       true,
			}

			if err = gopacket.SerializeLayers(sbuf, options, ipv4Resp, icmpResp, payload); err != nil {
				t.Fatalf("expected no error but got: %s", err)
			}

			if n, err = tun.Write(sbuf.Bytes()); err != nil {
				t.Fatalf("expected no error but got: %s", err)
			}

			break
		}
	}

	if ok := <-ch; !ok {
		t.Errorf("received no response")
	}
}
