package tuntap

import (
	"bytes"
	"encoding/binary"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// DHCPProxyAdapter implements a fake DHCP server as part of the adapter.
//
// All DHCP requests sent on the interface will be transparently handled.
type DHCPProxyAdapter struct {
	Adapter
	RootLayer          gopacket.LayerType
	ServerHardwareAddr net.HardwareAddr
	Entries            DHCPEntries
}

// DHCPEntry represents a DHCP entry.
type DHCPEntry struct {
	HardwareAddr net.HardwareAddr
	IPv4         *net.IPNet
	LeaseTime    time.Duration
}

// DHCPEntries represents a list of DHCP entries.
type DHCPEntries []DHCPEntry

func (a *DHCPProxyAdapter) Read(b []byte) (n int, err error) {
	for {
		n, err = a.Adapter.Read(b)

		if err != nil {
			return
		}

		if a.handlePacket(b[:n]) {
			return
		}
	}
}

func (a *DHCPProxyAdapter) handlePacket(b []byte) bool {
	// No IPv4 address is configured: we can't reply to anything.
	if a.Config().IPv4 == nil {
		return true
	}

	packet := gopacket.NewPacket(
		b,
		a.RootLayer,
		gopacket.DecodeOptions{Lazy: true, NoCopy: true},
	)

	var ok bool
	var ethernet *layers.Ethernet

	if a.RootLayer == layers.LayerTypeEthernet {
		ethernet, ok = packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)

		if !ok || ethernet == nil {
			return true
		}
	}

	ipv4, ok := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)

	if !ok || ipv4 == nil {
		return true
	}

	udp, ok := packet.Layer(layers.LayerTypeUDP).(*layers.UDP)

	if !ok || udp == nil {
		return true
	}

	dhcp, ok := packet.Layer(layers.LayerTypeDHCPv4).(*layers.DHCPv4)

	if !ok || dhcp == nil {
		return true
	}

	dhcpEntry, ok := a.Entries.Find(dhcp.ClientHWAddr)

	if !ok {
		// The requester hardward address is not known: ignoring the message.
		return true
	}

	if dhcp.Operation == layers.DHCPOpReply {
		// We ignore DHCP replies.
		return true
	}

	optMessageType := getDHCPOption(dhcp.Options, layers.DHCPOptMessageType)

	if optMessageType == nil || optMessageType.Length != 1 {
		// Invalid DHCP message does not contain a message type: ignoring.
		return false
	}

	messageType := layers.DHCPMsgType(optMessageType.Data[0])
	var respOptions layers.DHCPOptions

	// We try to honor the requested lease-time.
	optLeaseTime := getDHCPOption(dhcp.Options, layers.DHCPOptLeaseTime)

	var leaseTimeBuf [4]byte

	if optLeaseTime != nil && optLeaseTime.Length == 4 {
		copy(leaseTimeBuf[:], optLeaseTime.Data)
	} else {
		binary.BigEndian.PutUint32(leaseTimeBuf[:], uint32(dhcpEntry.LeaseTime.Seconds()))
	}

	respIPv4Address := a.Config().IPv4.IP

	switch messageType {
	case layers.DHCPMsgTypeDiscover:
		respOptions = append(
			respOptions,
			layers.NewDHCPOption(
				layers.DHCPOptMessageType,
				[]byte{byte(layers.DHCPMsgTypeOffer)},
			),
			layers.NewDHCPOption(
				layers.DHCPOptLeaseTime,
				leaseTimeBuf[:],
			),
		)
	case layers.DHCPMsgTypeRequest:
		optRequestIP := getDHCPOption(dhcp.Options, layers.DHCPOptRequestIP)

		if optRequestIP == nil {
			respOptions = append(respOptions, layers.NewDHCPOption(
				layers.DHCPOptMessageType,
				[]byte{byte(layers.DHCPMsgTypeNak)},
			))
		} else {
			if dhcpEntry.IPv4.IP.Equal(net.IP(optRequestIP.Data)) {
				respOptions = append(
					respOptions,
					layers.NewDHCPOption(
						layers.DHCPOptMessageType,
						[]byte{byte(layers.DHCPMsgTypeAck)},
					),
					layers.NewDHCPOption(
						layers.DHCPOptLeaseTime,
						leaseTimeBuf[:],
					),
				)
			} else {
				respOptions = append(respOptions, layers.NewDHCPOption(
					layers.DHCPOptMessageType,
					[]byte{byte(layers.DHCPMsgTypeNak)},
				))
			}
		}
	case layers.DHCPMsgTypeInform:
		// When we inform, we must not give an address back.
		respIPv4Address = net.IPv4zero

		respOptions = append(respOptions, layers.NewDHCPOption(
			layers.DHCPOptMessageType,
			[]byte{byte(layers.DHCPMsgTypeAck)},
		))
	default:
		// Unhandled message types.
		return false
	}

	serverIPv4 := a.Config().IPv4.IP.Mask(a.Config().IPv4.Mask)

	respOptions = append(respOptions, layers.NewDHCPOption(
		layers.DHCPOptServerID,
		[]byte(serverIPv4),
	))

	optParamsRequest := getDHCPOption(dhcp.Options, layers.DHCPOptParamsRequest)

	if optParamsRequest != nil {
		for _, param := range optParamsRequest.Data {
			param := layers.DHCPOpt(param)

			switch param {
			case layers.DHCPOptSubnetMask:
				respOptions = append(respOptions, layers.NewDHCPOption(
					param,
					dhcpEntry.IPv4.Mask,
				))
			}
		}
	}

	respOptions = append(respOptions, layers.NewDHCPOption(
		layers.DHCPOptEnd,
		nil,
	))

	respLayers := []gopacket.SerializableLayer{}

	if a.RootLayer == layers.LayerTypeEthernet {
		respEthernet := &layers.Ethernet{
			SrcMAC:       a.ServerHardwareAddr,
			DstMAC:       ethernet.SrcMAC,
			EthernetType: layers.EthernetTypeIPv4,
		}
		respLayers = append(respLayers, respEthernet)
	}

	respIPv4 := &layers.IPv4{
		Version:    ipv4.Version,
		IHL:        ipv4.IHL,
		TOS:        ipv4.TOS,
		Id:         ipv4.Id,
		Flags:      ipv4.Flags,
		FragOffset: ipv4.FragOffset,
		TTL:        ipv4.TTL,
		Protocol:   ipv4.Protocol,
		SrcIP:      serverIPv4,
		DstIP:      ipv4.SrcIP,
		Options:    ipv4.Options,
	}
	respUDP := &layers.UDP{
		SrcPort: udp.DstPort,
		DstPort: udp.SrcPort,
	}
	respUDP.SetNetworkLayerForChecksum(respIPv4)
	respDHCP := &layers.DHCPv4{
		Operation:    layers.DHCPOpReply,
		HardwareType: dhcp.HardwareType,
		HardwareLen:  dhcp.HardwareLen,
		HardwareOpts: dhcp.HardwareOpts,
		Xid:          dhcp.Xid,
		Secs:         dhcp.Secs,
		Flags:        dhcp.Flags,
		ClientIP:     net.IPv4zero,
		YourClientIP: respIPv4Address,
		NextServerIP: serverIPv4,
		RelayAgentIP: net.IPv4zero,
		ClientHWAddr: dhcp.ClientHWAddr,
		Options:      respOptions,
	}
	respLayers = append(respLayers, respIPv4, respUDP, respDHCP)

	sbuf := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	if err := gopacket.SerializeLayers(sbuf, options, respLayers...); err != nil {
		panic(err)
	}

	// If we failed to write the message, we do so silently. Packet loss happen...
	a.Write(sbuf.Bytes())

	return false
}

// Find a DHCP entry from its hardward address.
func (e DHCPEntries) Find(addr net.HardwareAddr) (DHCPEntry, bool) {
	for _, entry := range e {
		if bytes.Compare(entry.HardwareAddr, addr) == 0 {
			return entry, true
		}
	}

	return DHCPEntry{}, false
}

func getDHCPOption(options layers.DHCPOptions, opt layers.DHCPOpt) *layers.DHCPOption {
	for _, option := range options {
		if option.Type == opt {
			return &option
		}
	}

	return nil
}
