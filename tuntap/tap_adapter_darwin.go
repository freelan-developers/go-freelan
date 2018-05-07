package tuntap

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

/*
#include <stdlib.h>

#include <tap_adapter_unix.h>
*/
import "C"

type tapAdapter struct {
	*os.File
	inf *net.Interface
}

// NewTAPAdapter instantiates a new TAP adapter.
func NewTAPAdapter(config *TAPAdapterConfig) (TAPAdapter, error) {
	if config == nil {
		config = NewTAPAdapterConfig()
	}

	var f *os.File
	var err error
	var name = config.Name

	if name == "" {
		for i := 0; i < 16; i++ {
			name = fmt.Sprintf("/dev/tap%d", i)
			f, err = os.OpenFile(name, os.O_RDWR, 0)

			if err == nil {
				break
			}

			if err, ok := err.(*os.PathError); ok && err.Err == syscall.ENOENT {
				break
			}
		}

		if f == nil {
			return nil, errors.New("no usable TAP adapter was found")
		}
	} else {
		if f, err = os.OpenFile(name, os.O_RDWR, 0); err != nil {
			return nil, err
		}
	}

	if _, _, err := syscall.RawSyscall(syscall.SYS_FCNTL, f.Fd(), syscall.F_SETFD, syscall.FD_CLOEXEC); err != 0 {
		f.Close()

		return nil, fmt.Errorf("failed to set CLOEXEC: %v", err)
	}

	interfaceName := filepath.Base(name)

	inf, err := net.InterfaceByName(interfaceName)

	if err != nil {
		return nil, fmt.Errorf("failed to get interface details for `%s`: %v", interfaceName, err)
	}

	if config.IPv4 != nil {
		if err = setIPv4Address(name, *config.IPv4); err != nil {
			return nil, err
		}
	}

	return &tapAdapter{
		File: f,
		inf:  inf,
	}, nil
}

func (a *tapAdapter) Interface() *net.Interface {
	return a.inf
}

type ifreq struct {
	Name  [unix.IFNAMSIZ]byte
	Flags uint16
	pad   [40 - unix.IFNAMSIZ - 2]byte
}

func setIPv4Address(name string, ip net.IPNet) error {
	cName := unsafe.Pointer(C.CString(name))
	defer C.free(cName)

	ipBytes := C.CBytes(ip.IP)
	defer C.free(ipBytes)

	ones, _ := ip.Mask.Size()

	errno := C.set_ipv4_address((*C.char)(cName), (*C.char)(ipBytes), C.int(ones))

	if errno != 0 {
		return fmt.Errorf("setting IPv4 address to `%s` on `%s`: %s", ip, name, syscall.Errno(errno))
	}

	return nil
}
