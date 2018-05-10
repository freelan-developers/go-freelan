// +build darwin linux

package tuntap

import (
	"fmt"
	"net"
	"runtime"
	"syscall"
)

/*
#include "adapter_posix.h"
*/
import "C"

type adapterImpl struct {
	*adapterDescriptor
	inf *net.Interface
}

type adapterDescriptor struct {
	ptr *C.struct_adapter
}

func (t *adapterDescriptor) Close() error {
	_, err := C.close_adapter(t.ptr)

	runtime.SetFinalizer(t, nil)

	return err
}

func (t *adapterDescriptor) Name() string {
	return C.GoString(&t.ptr.name[0])
}

func (t *adapterDescriptor) Read(p []byte) (int, error) {
	return syscall.Read((int)(t.ptr.fd), p)
}

func (t *adapterDescriptor) Write(p []byte) (int, error) {
	return syscall.Write((int)(t.ptr.fd), p)
}

func (t *adapterDescriptor) SetIPv4(addr net.IPNet) error {
	ipBytes := C.CBytes(addr.IP.To4())
	defer C.free(ipBytes)

	ones, _ := addr.Mask.Size()

	_, err := C.set_adapter_ipv4(t.ptr, *(*C.struct_in_addr)(ipBytes), C.int(ones))

	return err
}

func (t *adapterDescriptor) SetRemoteIPv4(remoteAddr net.IP) error {
	ipBytes := C.CBytes(remoteAddr.To4())
	defer C.free(ipBytes)

	_, err := C.set_adapter_remote_ipv4(t.ptr, *(*C.struct_in_addr)(ipBytes))

	return err
}

func (t *adapterDescriptor) SetIPv6(addr net.IPNet) error {
	ipBytes := C.CBytes(addr.IP.To16())
	defer C.free(ipBytes)

	ones, _ := addr.Mask.Size()

	_, err := C.set_adapter_ipv6(t.ptr, *(*C.struct_in6_addr)(ipBytes), C.int(ones))

	return err
}

func newAdapter(name string, _type C.adapter_layer) (*adapterDescriptor, error) {
	cname := (*C.char)(C.NULL)

	if name != "" {
		//TODO: set cname to the cstring.
	}

	ptr, err := C.open_adapter(_type, cname)

	if err != nil {
		return nil, fmt.Errorf("failed to open tap adapter `%s`: %s", name, err)
	}

	desc := &adapterDescriptor{ptr}
	runtime.SetFinalizer(desc, (*adapterDescriptor).Close)

	return desc, nil
}

// NewTapAdapter instantiates a new tap adapter.
func NewTapAdapter(config *TapAdapterConfig) (TapAdapter, error) {
	if config == nil {
		config = NewTapAdapterConfig()
	}

	desc, err := newAdapter(config.Name, C.AL_ETHERNET)

	if err != nil {
		return nil, err
	}

	inf, err := net.InterfaceByName(desc.Name())

	if err != nil {
		return nil, fmt.Errorf("failed to get interface details for `%s`: %v", desc.Name(), err)
	}

	if config.IPv4 != nil {
		if err = desc.SetIPv4(*config.IPv4); err != nil {
			return nil, fmt.Errorf("setting IPv4 address to %s: %s", *config.IPv4, err)
		}
	}

	if config.IPv6 != nil {
		if err = desc.SetIPv6(*config.IPv6); err != nil {
			return nil, fmt.Errorf("setting IPv6 address to %s: %s", *config.IPv6, err)
		}
	}

	return &adapterImpl{
		adapterDescriptor: desc,
		inf:               inf,
	}, nil
}

// NewTunAdapter instantiates a new tun adapter.
func NewTunAdapter(config *TunAdapterConfig) (TunAdapter, error) {
	if config == nil {
		config = NewTunAdapterConfig()
	}

	desc, err := newAdapter(config.Name, C.AL_IP)

	if err != nil {
		return nil, err
	}

	inf, err := net.InterfaceByName(desc.Name())

	if err != nil {
		return nil, fmt.Errorf("failed to get interface details for `%s`: %v", desc.Name(), err)
	}

	if config.IPv4 != nil {
		if err = desc.SetIPv4(*config.IPv4); err != nil {
			return nil, fmt.Errorf("setting IPv4 address to %s: %s", *config.IPv4, err)
		}
	}

	if config.RemoteIPv4 != nil {
		if err = desc.SetRemoteIPv4(*config.RemoteIPv4); err != nil {
			return nil, fmt.Errorf("setting remote IPv4 address to %s: %s", *config.RemoteIPv4, err)
		}
	}

	if config.IPv6 != nil {
		if err = desc.SetIPv6(*config.IPv6); err != nil {
			return nil, fmt.Errorf("setting IPv6 address to %s: %s", *config.IPv6, err)
		}
	}

	return &adapterImpl{
		adapterDescriptor: desc,
		inf:               inf,
	}, nil
}

func (a *adapterImpl) Interface() *net.Interface {
	return a.inf
}
