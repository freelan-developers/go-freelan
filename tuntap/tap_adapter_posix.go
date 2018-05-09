// +build darwin linux

package tuntap

import (
	"fmt"
	"net"
	"runtime"
	"syscall"
)

/*
#include "tap_adapter_posix.h"
*/
import "C"

type tapAdapter struct {
	*tapAdapterFD
	inf *net.Interface
}

type tapAdapterFD struct {
	ptr *C.struct_tap_adapter
}

func (t *tapAdapterFD) Close() error {
	_, err := C.close_tap_adapter(t.ptr)

	runtime.SetFinalizer(t, nil)

	return err
}

func (t *tapAdapterFD) Name() string {
	return C.GoString(&t.ptr.name[0])
}

func (t *tapAdapterFD) Read(p []byte) (int, error) {
	return syscall.Read((int)(t.ptr.fd), p)
}

func (t *tapAdapterFD) Write(p []byte) (int, error) {
	return syscall.Write((int)(t.ptr.fd), p)
}

func (t *tapAdapterFD) SetIPv4(addr net.IPNet) error {
	ipBytes := C.CBytes(addr.IP.To4())
	defer C.free(ipBytes)

	ones, _ := addr.Mask.Size()

	_, err := C.set_tap_adapter_ipv4(t.ptr, *(*C.struct_in_addr)(ipBytes), C.int(ones))

	return err
}

func (t *tapAdapterFD) SetRemoteIPv4(addr net.IP) error {
	ipBytes := C.CBytes(addr.To4())
	defer C.free(ipBytes)

	_, err := C.set_tap_adapter_remote_ipv4(t.ptr, *(*C.struct_in_addr)(ipBytes))

	return err
}

func (t *tapAdapterFD) SetIPv6(addr net.IPNet) error {
	ipBytes := C.CBytes(addr.IP.To16())
	defer C.free(ipBytes)

	ones, _ := addr.Mask.Size()

	_, err := C.set_tap_adapter_ipv6(t.ptr, *(*C.struct_in6_addr)(ipBytes), C.int(ones))

	return err
}

// NewTAPAdapter instantiates a new TAP adapter.
func NewTAPAdapter(config *TAPAdapterConfig) (TAPAdapter, error) {
	if config == nil {
		config = NewTAPAdapterConfig()
	}

	cname := (*C.char)(C.NULL)

	if config.Name != "" {
		//TODO: set cname to the cstring.
	}

	ptr, err := C.open_tap_adapter(C.TA_ETHERNET, cname)

	if err != nil {
		return nil, fmt.Errorf("failed to open tap adapter name: %s", err)
	}

	taFD := &tapAdapterFD{ptr}
	runtime.SetFinalizer(taFD, (*tapAdapterFD).Close)

	name := taFD.Name()

	inf, err := net.InterfaceByName(name)

	if err != nil {
		return nil, fmt.Errorf("failed to get interface details for `%s`: %v", name, err)
	}

	if config.IPv4 != nil {
		if err = taFD.SetIPv4(*config.IPv4); err != nil {
			return nil, fmt.Errorf("setting IPv4 address to %s: %s", *config.IPv4, err)
		}
	}

	return &tapAdapter{
		tapAdapterFD: taFD,
		inf:          inf,
	}, nil
}

func (a *tapAdapter) Interface() *net.Interface {
	return a.inf
}

//func setIPv4Address(name string, ip net.IPNet) error {
//	cName := unsafe.Pointer(C.CString(name))
//	defer C.free(cName)
//
//	ipBytes := C.CBytes(ip.IP)
//	defer C.free(ipBytes)
//
//	ones, _ := ip.Mask.Size()
//
//	errno := C.set_ipv4_address((*C.char)(cName), (*C.char)(ipBytes), C.int(ones))
//
//	if errno != 0 {
//		return fmt.Errorf("setting IPv4 address to `%s` on `%s`: %s", ip, name, syscall.Errno(errno))
//	}
//
//	return nil
//}
