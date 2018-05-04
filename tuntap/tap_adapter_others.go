// +build !windows,!darwin

package tuntap

import "errors"

// NewTAPAdapter instantiates a new TAP adapter.
func NewTAPAdapter(config *TAPAdapterConfig) (TAPAdapter, error) {
	return nil, errors.New("not implemented on this platform")
}
