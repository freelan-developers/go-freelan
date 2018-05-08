// +build darwin linux

package tuntap

import (
	"fmt"
	"io"
	"net"
	"runtime"
	"strings"
	"unsafe"
)

/*
#include "tap_adapter_posix.h"
*/
import "C"

type tapAdapter struct {
	io.ReadWriter
	*tapAdapterFD
	inf *net.Interface
}

type tapAdapterFD struct {
	fd C.int
}

func (t *tapAdapterFD) Close() error {
	_, err := C.close_tap_adapter(t.fd)

	runtime.SetFinalizer(t, nil)

	return err
}

// NewTAPAdapter instantiates a new TAP adapter.
func NewTAPAdapter(config *TAPAdapterConfig) (TAPAdapter, error) {
	if config == nil {
		config = NewTAPAdapterConfig()
	}

	cname := (*C.char)(C.NULL)

	if config.Name != "" {
		//TODO: set name to the cstring.
	}

	fd, err := C.open_tap_adapter(C.TA_ETHERNET, cname)

	if err != nil {
		return nil, fmt.Errorf("failed to open tap adapter name: %s", err)
	}

	taFD := &tapAdapterFD{fd}
	runtime.SetFinalizer(taFD, (*tapAdapterFD).Close)

	buf := make([]byte, C.IFNAMSIZ)

	if _, err := C.get_tap_adapter_name((*C.char)(unsafe.Pointer(&buf[0])), (C.ulong)(len(buf)), fd); err != nil {
		return nil, fmt.Errorf("failed to get tap adapter name: %s", err)
	}

	name := strings.TrimRight(string(buf), "\\0")

	inf, err := net.InterfaceByName(name)

	if err != nil {
		return nil, fmt.Errorf("failed to get interface details for `%s`: %v", name, err)
	}

	//if config.IPv4 != nil {
	//	if err = setIPv4Address(name, *config.IPv4); err != nil {
	//		return nil, err
	//	}
	//}

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
