package tuntap

import (
	"io"
	"runtime"

	"golang.org/x/sys/windows"
)

type overlappedFile struct {
	fd   windows.Handle
	name string
}

func (f *overlappedFile) Read(b []byte) (int, error) {
	return 0, io.EOF
}

func (f *overlappedFile) Write(b []byte) (int, error) {
	return 0, io.EOF
}

func (f *overlappedFile) Close() error {
	runtime.SetFinalizer(f, nil)

	return windows.Close(f.fd)
}
