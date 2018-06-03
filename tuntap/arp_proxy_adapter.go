package tuntap

import (
	"bytes"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// ARPProxyAdapter implements fake ARP support.
//
// All ARPv4 requests sent on the interface will be transparently handled.
//
// Gratuitous ARP requests are never replied to. All other ARP requests on the
// configured ARP network are replied to with the interface ethernet address.
type ARPProxyAdapter struct {
	Adapter
	ARPTable ARPTable
}

// ARPTable represents an ARP table.
type ARPTable interface {
	Register(*net.IPNet, net.HardwareAddr)
	Unregister(*net.IPNet)
	Resolve(net.IP) net.HardwareAddr
}

type arpTable struct {
	table map[*net.IPNet]net.HardwareAddr
}

// NewARPTable instanciates a new ARP table.
func NewARPTable() ARPTable {
	return &arpTable{
		table: make(map[*net.IPNet]net.HardwareAddr),
	}
}

func (t *arpTable) Register(network *net.IPNet, hwaddr net.HardwareAddr) {
	t.table[network] = hwaddr
}

func (t *arpTable) Unregister(network *net.IPNet) {
	delete(t.table, network)
}

func (t *arpTable) Resolve(ip net.IP) net.HardwareAddr {
	for ipnet, hwaddr := range t.table {
		if ipnet.Contains(ip) {
			return hwaddr
		}
	}

	return nil
}

func (a *ARPProxyAdapter) Read(b []byte) (n int, err error) {
	for {
		n, err = a.Adapter.Read(b)

		if err != nil {
			return
		}

		ipv4 := a.Config().IPv4

		// No IPv4 address is configured: we can't reply to anything.
		if ipv4 == nil {
			return
		}

		packet := gopacket.NewPacket(
			b[:n],
			layers.LayerTypeEthernet,
			gopacket.DecodeOptions{Lazy: true, NoCopy: true},
		)

		arp, ok := packet.Layer(layers.LayerTypeARP).(*layers.ARP)

		if !ok || arp == nil {
			return
		}

		// We only care about ARP requests.
		if arp.Operation != layers.ARPRequest {
			continue
		}

		// Bogus request: source protocol address is supposed to be the interface's
		// address.
		if bytes.Compare(arp.SourceProtAddress, ipv4.IP.To4()) != 0 {
			continue
		}

		// We don't reply to gratuitous ARP requests.
		if bytes.Compare(arp.DstProtAddress, ipv4.IP.To4()) == 0 {
			continue
		}

		reqIPv4 := net.IP(arp.DstProtAddress).To4()

		// If the request is outside the interface's network, we don't reply.
		hwaddr := a.ARPTable.Resolve(reqIPv4)

		if hwaddr == nil {
			continue
		}

		ethernetResp := &layers.Ethernet{
			SrcMAC:       hwaddr,
			DstMAC:       arp.SourceHwAddress,
			EthernetType: layers.EthernetTypeARP,
		}
		arpResp := &layers.ARP{
			AddrType:          arp.AddrType,
			Protocol:          arp.Protocol,
			HwAddressSize:     arp.HwAddressSize,
			ProtAddressSize:   arp.ProtAddressSize,
			Operation:         layers.ARPReply,
			SourceHwAddress:   hwaddr,
			SourceProtAddress: arp.DstProtAddress,
			DstHwAddress:      arp.SourceHwAddress,
			DstProtAddress:    arp.SourceProtAddress,
		}

		sbuf := gopacket.NewSerializeBuffer()
		options := gopacket.SerializeOptions{
			ComputeChecksums: true,
			FixLengths:       true,
		}

		if err = gopacket.SerializeLayers(sbuf, options, ethernetResp, arpResp); err != nil {
			panic(err)
		}

		// If we failed to write the message, we do so silently. Packet loss happen...
		a.Write(sbuf.Bytes())
	}
}
