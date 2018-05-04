package tuntap

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	userModeDeviceDir = "\\\\.\\Global\\"
	tapWinSuffix      = ".tap"
)

type tapAdapter struct {
}

// NewTAPAdapter instantiates a new TAP adapter.
func NewTAPAdapter(config *TAPAdapterConfig) (TAPAdapter, error) {
	if config == nil {
		config = NewTAPAdapterConfig()
	}

	var bytesSize uint32

	if err := windows.GetAdaptersInfo(nil, &bytesSize); err != syscall.ERROR_BUFFER_OVERFLOW {
		return nil, err
	}

	size := uintptr(bytesSize) / unsafe.Sizeof(windows.IpAdapterInfo{})
	ai := make([]windows.IpAdapterInfo, size)

	if err := windows.GetAdaptersInfo(&ai[0], &bytesSize); err != nil {
		return nil, err
	}

	for i, a := range ai {
		name := strings.TrimRight(string(a.AdapterName[:]), "\x00")
		description := strings.TrimRight(string(a.Description[:]), "\x00")
		fmt.Printf("%d: %#v\n", i, name)
		fmt.Printf("%d: %#v\n", i, description)

		// TODO: Use the registry and do this better.
		if strings.HasPrefix(description, "TAP") {
			path := fmt.Sprintf("%s%s%s", userModeDeviceDir, name, tapWinSuffix)
			// We may need to copy Freelan and open the file in overlapped mode.
			f, err := os.OpenFile(path, os.O_RDWR, 0)

			if err != nil {
				return nil, err
			}

			fmt.Println(f.Name())
			break
		}
	}

	return nil, errors.New("not implemented")
}
