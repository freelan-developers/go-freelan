// +build darwin linux

package tuntap

import (
	"fmt"
	"net"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/freelan-developers/go-freelan/routing"
)

/*
#include "adapter_posix.h"
*/
import "C"

type adapterImpl struct {
	*adapterDescriptor
	inf                   *net.Interface
	remoteIPv4            net.IP
	remoteIPv4RemovalFunc func()
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

func (t *adapterDescriptor) SetIPv4(addr *net.IPNet) error {
	ipBytes := C.CBytes(addr.IP.To4())
	defer C.free(ipBytes)

	ones, _ := addr.Mask.Size()

	_, err := C.set_adapter_ipv4(t.ptr, *(*C.struct_in_addr)(ipBytes), C.int(ones))

	return err
}

func (t *adapterDescriptor) RemoteIPv4() (net.IP, error) {
	result := net.IP(make([]byte, 4))
	ipBytes := unsafe.Pointer(&result)

	_, err := C.get_adapter_remote_ipv4(t.ptr, (*C.struct_in_addr)(ipBytes))

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (t *adapterDescriptor) SetRemoteIPv4(remoteAddr net.IP) error {
	ipBytes := C.CBytes(remoteAddr.To4())
	defer C.free(ipBytes)

	_, err := C.set_adapter_remote_ipv4(t.ptr, *(*C.struct_in_addr)(ipBytes))

	return err
}

func (t *adapterDescriptor) SetIPv6(addr *net.IPNet) error {
	ipBytes := C.CBytes(addr.IP.To16())
	defer C.free(ipBytes)

	ones, _ := addr.Mask.Size()

	_, err := C.set_adapter_ipv6(t.ptr, *(*C.struct_in6_addr)(ipBytes), C.int(ones))

	return err
}

func (a *adapterImpl) IPv4() (*net.IPNet, error) {
	addrs, err := a.Interface().Addrs()

	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		if ip, ipnet, err := net.ParseCIDR(addr.String()); err == nil {
			if ipv4 := ip.To4(); ipv4 != nil {
				return &net.IPNet{
					IP:   ipv4,
					Mask: ipnet.Mask,
				}, nil
			}
		}
	}

	return nil, nil
}

func (a *adapterImpl) IPv6() (*net.IPNet, error) {
	addrs, err := a.Interface().Addrs()

	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		if ip, ipnet, err := net.ParseCIDR(addr.String()); err == nil {
			if ipv6 := ip.To16(); ipv6 != nil {
				return &net.IPNet{
					IP:   ipv6,
					Mask: ipnet.Mask,
				}, nil
			}
		}
	}

	return nil, nil
}

func (a *adapterImpl) RemoteIPv4() (net.IP, error) {
	if runtime.GOOS == "darwin" {
		return a.remoteIPv4, nil
	}

	return a.adapterDescriptor.RemoteIPv4()
}

func (a *adapterImpl) SetRemoteIPv4(remoteAddr net.IP) error {
	// tuntaposx does not support set a remote IPv4 on the tun interface.
	//
	// Let's add a route instead.
	if runtime.GOOS == "darwin" {
		// Let's get rid of the previous routing entry, if there is one.
		if a.remoteIPv4RemovalFunc != nil {
			a.remoteIPv4RemovalFunc()
			a.remoteIPv4RemovalFunc = nil
		}

		ipv4, err := a.IPv4()

		if err != nil {
			return fmt.Errorf("setting remote IPv4: could not determine local IPv4 address: %s", err)
		}

		if ipv4 == nil {
			return fmt.Errorf("setting remote IPv4: interface has no local IPv4 address")
		}

		// The remote address is contained in the network: no need to add a route for it.
		if ipv4.Contains(remoteAddr) {
			return nil
		}

		router := routing.NewRouter()
		ok, err := router.AddRoute(&net.IPNet{IP: remoteAddr, Mask: net.CIDRMask(32, 32)}, ipv4.IP)

		if err != nil {
			return err
		}

		a.remoteIPv4 = remoteAddr

		if ok {
			a.remoteIPv4RemovalFunc = func() {
				router.DeleteRoute(&net.IPNet{IP: remoteAddr, Mask: net.CIDRMask(32, 32)}, ipv4.IP)
			}
		}

		return nil
	}

	return a.adapterDescriptor.SetRemoteIPv4(remoteAddr)
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

	adapter := &adapterImpl{
		adapterDescriptor: desc,
		inf:               inf,
	}

	if config.IPv4 != nil {
		if err = adapter.SetIPv4(config.IPv4); err != nil {
			return nil, fmt.Errorf("setting IPv4 address to %s: %s", *config.IPv4, err)
		}
	}

	if config.IPv6 != nil {
		if err = adapter.SetIPv6(config.IPv6); err != nil {
			return nil, fmt.Errorf("setting IPv6 address to %s: %s", *config.IPv6, err)
		}
	}

	return adapter, nil
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

	adapter := &adapterImpl{
		adapterDescriptor: desc,
		inf:               inf,
	}

	if config.IPv4 != nil {
		if err = adapter.SetIPv4(config.IPv4); err != nil {
			return nil, fmt.Errorf("setting IPv4 address to %s: %s", *config.IPv4, err)
		}
	}

	if config.RemoteIPv4 != nil {
		if err = adapter.SetRemoteIPv4(*config.RemoteIPv4); err != nil {
			return nil, fmt.Errorf("setting remote IPv4 address to %s: %s", *config.RemoteIPv4, err)
		}
	}

	if config.IPv6 != nil {
		if err = adapter.SetIPv6(config.IPv6); err != nil {
			return nil, fmt.Errorf("setting IPv6 address to %s: %s", *config.IPv6, err)
		}
	}

	return adapter, nil
}

func (a *adapterImpl) Interface() *net.Interface {
	return a.inf
}
