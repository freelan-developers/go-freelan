// +build !windows,!darwin,!linux

package tuntap

import "errors"

// NewTapAdapter instantiates a new tap adapter.
func NewTapAdapter(config *TapAdapterConfig) (TapAdapter, error) {
	return nil, errors.New("not implemented on this platform")
}

// NewTunAPAdapter instantiates a new tun adapter.
func NewTunAdapter(config *TunAdapterConfig) (TunAdapter, error) {
	return nil, errors.New("not implemented on this platform")
}
