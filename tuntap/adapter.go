package tuntap

import (
	"io"
	"net"
)

// Adapter is the base interface for tun & tap adapters.
type Adapter interface {
	io.ReadWriteCloser
	Interface() *net.Interface
}

// TapAdapter represents a tap adapter.
type TapAdapter interface {
	Adapter
}

// TunAdapter represents a tun adapter.
type TunAdapter interface {
	Adapter
}

// TapAdapterConfig represents a tap adapter config.
type TapAdapterConfig struct {
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
}

// TunAdapterConfig represents a tun adapter config.
type TunAdapterConfig struct {
	// Name is the name of the tun adapter to open.
	//
	// The exact value of this field is operating-system-dependant.
	//
	// On most systems, specifying an empty name will trigger auto-assignation
	// or device creation.
	Name string

	// IPv4 is an IPv4 address to set on the interface after its goes up.
	IPv4 *net.IPNet

	// RemoteIPv4 is a remote IPv4 address to set on the interface after its goes up.
	RemoteIPv4 *net.IP

	// IPv6 is an IPv6 address to set on the interface after its goes up.
	IPv6 *net.IPNet
}

// NewTapAdapterConfig instantiate a new default configuration.
func NewTapAdapterConfig() *TapAdapterConfig {
	return &TapAdapterConfig{}
}

// NewTunAdapterConfig instantiate a new default configuration.
func NewTunAdapterConfig() *TunAdapterConfig {
	return &TunAdapterConfig{}
}
