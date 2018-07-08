// +build darwin linux

package tuntap

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/freelan-developers/go-freelan/routing"
	"github.com/google/gopacket/layers"
)

/*
#include "adapter_posix.h"
*/
import "C"

type adapterImpl struct {
	*adapterDescriptor
	inf    *net.Interface
	config *AdapterConfig
}

type adapterDescriptor struct {
	ptr *C.struct_adapter
}

func (t *adapterDescriptor) Close() error {
	t.SetConnectedState(false)
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

	if _, err := C.set_adapter_ipv4(t.ptr, *(*C.struct_in_addr)(ipBytes), C.int(ones)); err != nil {
		return err
	}

	if t.ptr.layer == C.AL_ETHERNET {
		return nil
	}

	// On OSX, the ioctl apparently doesn't have the desired effect, so we set the remote IPv4 address through other means add
	// an explicit route instead.
	if runtime.GOOS == "darwin" {
		network := &net.IPNet{
			IP:   addr.IP.Mask(addr.Mask),
			Mask: addr.Mask,
		}
		args := []string{
			t.Name(),
			addr.IP.String(),
			net.IPv4bcast.Mask(addr.Mask).String(),
			network.IP.String(),
		}
		cmd := exec.Command("ifconfig", args...)
		b, err := cmd.CombinedOutput()

		if err != nil {
			return fmt.Errorf("failed to call `ifconfig %s`: %s (output follows)\n%s", strings.Join(args, " "), err, string(b))
		}

		router := routing.NewRouter()
		_, err = router.AddRoute(network, addr.IP)

		return err
	}

	return t.SetRemoteIPv4(addr.IP.Mask(addr.Mask))
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

func (t *adapterDescriptor) SetConnectedState(state bool) error {
	flag := C.int(0)

	if state {
		flag = C.int(1)
	}

	_, err := C.set_adapter_connected_state(t.ptr, flag)

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
func NewTapAdapter(config *AdapterConfig) (Adapter, error) {
	if config == nil {
		config = NewAdapterConfig()
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
		config:            config,
	}

	var result Adapter = adapter

	if config.IPv4 != nil {
		if err = adapter.SetIPv4(config.IPv4); err != nil {
			adapter.Close()
			return nil, fmt.Errorf("setting IPv4 address to %s: %s", *config.IPv4, err)
		}

		if !config.DisableARP {
			arpTable := NewARPTable()
			arpTable.Register(config.IPv4, net.HardwareAddr{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFE})

			result = &ARPProxyAdapter{
				Adapter:  result,
				ARPTable: arpTable,
			}
		}

		if !config.DisableDHCP {
			result = &DHCPProxyAdapter{
				Adapter:            result,
				RootLayer:          layers.LayerTypeEthernet,
				ServerHardwareAddr: net.HardwareAddr{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFE},
				Entries: DHCPEntries{
					DHCPEntry{
						HardwareAddr: inf.HardwareAddr,
						IPv4:         config.IPv4,
						LeaseTime:    time.Hour,
					},
				},
			}
		}
	}

	if config.IPv6 != nil {
		if err = adapter.SetIPv6(config.IPv6); err != nil {
			adapter.Close()
			return nil, fmt.Errorf("setting IPv6 address to %s: %s", *config.IPv6, err)
		}
	}

	if err = adapter.SetConnectedState(true); err != nil {
		adapter.Close()
		return nil, fmt.Errorf("failed to bring adapter up: %s", err)
	}

	return result, nil
}

// NewTunAdapter instantiates a new tun adapter.
func NewTunAdapter(config *AdapterConfig) (Adapter, error) {
	if config == nil {
		config = NewAdapterConfig()
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
		config:            config,
	}

	if config.IPv4 != nil {
		if err = adapter.SetIPv4(config.IPv4); err != nil {
			return nil, fmt.Errorf("setting IPv4 address to %s: %s", *config.IPv4, err)

		}

		if !config.DisableDHCP {
			return &DHCPProxyAdapter{
				Adapter:            adapter,
				RootLayer:          layers.LayerTypeIPv4,
				ServerHardwareAddr: net.HardwareAddr{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFE},
				Entries: DHCPEntries{
					DHCPEntry{
						HardwareAddr: inf.HardwareAddr,
						IPv4:         config.IPv4,
						LeaseTime:    time.Hour,
					},
				},
			}, nil
		}
	}

	if config.IPv6 != nil {
		if err = adapter.SetIPv6(config.IPv6); err != nil {
			return nil, fmt.Errorf("setting IPv6 address to %s: %s", *config.IPv6, err)
		}
	}

	if err = adapter.SetConnectedState(true); err != nil {
		adapter.Close()
		return nil, fmt.Errorf("failed to bring adapter up: %s", err)
	}

	return adapter, nil
}

func (a *adapterImpl) Interface() *net.Interface {
	return a.inf
}

func (a *adapterImpl) Config() AdapterConfig {
	return *a.config
}
