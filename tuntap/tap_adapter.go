package tuntap

import (
	"io"
	"net"
)

// TAPAdapter represents a TAP adapter.
type TAPAdapter interface {
	io.ReadWriteCloser
	Interface() *net.Interface
}

// TAPAdapterConfig represents a TAP adapter config.
type TAPAdapterConfig struct {
	// Name is the name of the TAP adapter to open.
	//
	// The exact value of this field is operating-system-dependant.
	//
	// On most systems, specifying an empty name will trigger auto-assignation
	// or device creation.
	Name string
}

// NewTAPAdapterConfig instantiate a new default configuration.
func NewTAPAdapterConfig() *TAPAdapterConfig {
	return &TAPAdapterConfig{}
}
