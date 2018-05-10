package tuntap

import (
	"fmt"
	"net"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	userModeDeviceDir   = "\\\\.\\Global\\"
	tapWinSuffix        = ".tap"
	adaptersRegistryKey = `SYSTEM\\CurrentControlSet\\Control\\Class\\{4D36E972-E325-11CE-BFC1-08002BE10318}`
	tapComponentID      = "tap0901"
)

type tapAdapter struct {
	*overlappedFile
	inf *net.Interface
}

// NewTapAdapter instantiates a new tap adapter.
func NewTapAdapter(config *TapAdapterConfig) (TapAdapter, error) {
	if config == nil {
		config = NewTapAdapterConfig()
	}

	aas, err := getTapAdaptersAddresses()

	if err != nil {
		return nil, fmt.Errorf("failed to get tap adapters addresses: %s", err)
	}

	var h windows.Handle
	var aa adapterAddresses

	for _, aa = range aas {
		if config.Name == "" || config.Name == aa.Name {
			if h, err = openTapAdapter(aa.Name); err == nil {
				break
			}

			if config.Name != "" {
				return nil, fmt.Errorf("failed to open TAP adapter `%s`: %s", config.Name, err)
			}
		}
	}

	if h == 0 {
		return nil, fmt.Errorf("no available TAP adapter were found")
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

func getTapAdaptersNames() ([]string, error) {
	root, err := registry.OpenKey(registry.LOCAL_MACHINE, adaptersRegistryKey, registry.READ)

	if err != nil {
		return nil, fmt.Errorf("opening root key at `%s`: %s", adaptersRegistryKey, err)
	}

	defer root.Close()

	names, err := root.ReadSubKeyNames(0)

	if err != nil {
		return nil, fmt.Errorf("enumerating sub-keys: %s", err)
	}

	result := make([]string, 0, len(names))

	for _, name := range names {
		k, err := registry.OpenKey(root, name, registry.READ)

		if err != nil {
			continue
		}

		defer k.Close()

		componentID, _, err := k.GetStringValue("ComponentId")

		if err == nil && componentID == tapComponentID {
			ifName, _, err := k.GetStringValue("NetCfgInstanceId")

			if err != nil {
				return nil, fmt.Errorf("reading NetCfgInstanceId from `%s`: %s", name, err)
			}

			result = append(result, ifName)
		}
	}

	return result, nil
}

type adapterAddresses struct {
	Name         string
	Description  string
	FriendlyName string
	Index        int
}

func getTapAdaptersAddresses() (result []adapterAddresses, err error) {
	var names []string

	if names, err = getTapAdaptersNames(); err != nil {
		return nil, fmt.Errorf("listing TAP adapters names: %s", err)
	}

	if result, err = getAdaptersAddresses(); err != nil {
		return nil, fmt.Errorf("listing TAP adapters addreses: %s", err)
	}

	filteredResult := make([]adapterAddresses, 0, len(result))

	for _, aa := range result {
		for _, name := range names {
			if aa.Name == name {
				filteredResult = append(filteredResult, aa)
			}
		}
	}

	return filteredResult, nil
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

func openTapAdapter(name string) (windows.Handle, error) {
	path := fmt.Sprintf("%s%s%s", userModeDeviceDir, name, tapWinSuffix)
	pathp, err := syscall.UTF16PtrFromString(path)

	if err != nil {
		return 0, fmt.Errorf("failed to convert path to UTF16: %s", err)
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
		return 0, fmt.Errorf("failed to open tap adapter `%s`: %s", name, err)
	}

	return h, nil
}