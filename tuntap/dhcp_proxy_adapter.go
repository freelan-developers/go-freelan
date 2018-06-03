package tuntap

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// DHCPProxyAdapter implements a fake DHCP server as part of the adapter.
//
// All DHCP requests sent on the interface will be transparently handled.
type DHCPProxyAdapter struct {
	Adapter
	RootLayer gopacket.LayerType
}

func (a *DHCPProxyAdapter) Read(b []byte) (n int, err error) {
	n, err = a.Adapter.Read(b)

	if err != nil {
		return
	}

	packet := gopacket.NewPacket(
		b[:n],
		a.RootLayer,
		gopacket.DecodeOptions{Lazy: true, NoCopy: true},
	)

	packetLayers := packet.Layers()

	for i, layer := range packetLayers {
		if layer.LayerType() == layers.LayerTypeIPv4 {
			packetLayers = packetLayers[i:]
			break
		}
	}

	return
}
