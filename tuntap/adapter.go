package tuntap

import (
	"io"
	"net"
)

// Adapter is the base interface for tun & tap adapters.
type Adapter interface {
	io.ReadWriteCloser
	Interface() *net.Interface
	Config() AdapterConfig
}

// AdapterConfig represents a tap adapter config.
type AdapterConfig struct {
	// Name is the name of the tap adapter to open.
	//
	// The exact value of this field is operating-system-dependant.
	//
	// On most systems, specifying an empty name will trigger auto-assignation
	// or device creation.
	Name string

	// IPv4 is an IPv4 address to set on the interface after its goes up.
	IPv4 *net.IPNet

	// IPv6 is an IPv6 address to set on the interface after its goes up.
	IPv6 *net.IPNet

	// DisableARP disables the ARP proxy.
	//
	// This is ignored on TUN adapters.
	DisableARP bool

	// DisableDHCP disables the fake DHCP server.
	DisableDHCP bool
}

// NewAdapterConfig instantiate a new default configuration.
func NewAdapterConfig() *AdapterConfig {
	return &AdapterConfig{}
}
