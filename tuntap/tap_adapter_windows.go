package tuntap

import (
	"fmt"
	"net"
	"runtime"
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
	*overlappedFile
	inf *net.Interface
}

// NewTAPAdapter instantiates a new TAP adapter.
func NewTAPAdapter(config *TAPAdapterConfig) (TAPAdapter, error) {
	if config == nil {
		config = NewTAPAdapterConfig()
	}

	aas, err := getAdaptersAddresses()

	if err != nil {
		return nil, fmt.Errorf("failed to get adapters addresses: %s", err)
	}

	var aa adapterAddresses

	for _, aa = range aas {
		// TODO: Use the registry and do this better.
		if strings.HasPrefix(aa.Description, "TAP") {
			break
		}
	}

	path := fmt.Sprintf("%s%s%s", userModeDeviceDir, aa.Name, tapWinSuffix)
	pathp, err := syscall.UTF16PtrFromString(path)

	if err != nil {
		return nil, fmt.Errorf("failed to convert path to UTF16: %s", err)
	}

	h, err := windows.CreateFile(
		pathp,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		0,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_SYSTEM|syscall.FILE_FLAG_OVERLAPPED,
		0,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %s", path, err)
	}

	inf, err := net.InterfaceByIndex(aa.Index)

	if err != nil {
		return nil, fmt.Errorf("failed to get interface details for `%s`: %v", aa.FriendlyName, err)
	}

	ta := &tapAdapter{
		&overlappedFile{
			fd:   h,
			name: aa.Name,
		},
		inf,
	}

	runtime.SetFinalizer(ta.overlappedFile, (*ta.overlappedFile).Close())

	return ta, nil
}

func (a *tapAdapter) Interface() *net.Interface {
	return a.inf
}

type adapterAddresses struct {
	Name         string
	Description  string
	FriendlyName string
	Index        int
}

func getAdaptersAddresses() (result []adapterAddresses, err error) {
	// MSDN recommends starting with a 15KB buffer to store the results.
	buf := make([]byte, 15*1024)
	size := uint32(len(buf))

	for {
		if err = windows.GetAdaptersAddresses(
			windows.AF_UNSPEC,
			0,
			0,
			(*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])),
			&size,
		); err == nil {
			break
		}

		if err != windows.ERROR_BUFFER_OVERFLOW {
			return
		}

		buf = make([]byte, int(size))
	}

	for aa := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])); aa != nil; aa = aa.Next {
		value := adapterAddresses{
			Name:         bytePtrToString(aa.AdapterName),
			Description:  uint16PtrToString(aa.Description),
			FriendlyName: uint16PtrToString(aa.FriendlyName),
			Index:        int(aa.IfIndex),
		}

		result = append(result, value)
	}

	return
}

// bytePtrToString will convert a pointer to a null-terminated string to a Go
// string.
func bytePtrToString(b *byte) string {
	buf := make([]byte, 0, 256)

	for c := unsafe.Pointer(b); *((*byte)(c)) != 0; c = unsafe.Pointer(uintptr(c) + 1) {
		buf = append(buf, *((*byte)(c)))
	}

	return string(buf)
}

func uint16PtrToString(b *uint16) string {
	buf := make([]uint16, 0, 256)

	for c := unsafe.Pointer(b); *((*uint16)(c)) != 0; c = unsafe.Pointer(uintptr(c) + unsafe.Sizeof(uint16(0))) {
		buf = append(buf, *((*uint16)(c)))
	}

	// UTF16ToString expects a zero-terminated buffer.
	buf = append(buf, 0)

	return syscall.UTF16ToString(buf)
}
