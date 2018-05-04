package tuntap

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
)

type tapAdapter struct {
	*os.File
	inf *net.Interface
}

// NewTAPAdapter instantiates a new TAP adapter.
func NewTAPAdapter(config *TAPAdapterConfig) (TAPAdapter, error) {
	if config == nil {
		config = &TAPAdapterConfig{}
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

	return &tapAdapter{
		File: f,
		inf:  inf,
	}, nil
}

func (a *tapAdapter) Interface() *net.Interface {
	return a.inf
}
