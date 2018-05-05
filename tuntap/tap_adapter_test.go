package tuntap

import (
	"fmt"
	"testing"
)

func TestTAPAdapter(t *testing.T) {
	tap, err := NewTAPAdapter(nil)

	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}

	if tap == nil {
		t.Fatal("expected not nil")
	}

	defer tap.Close()

	buf := make([]byte, tap.Interface().MTU)
	n, err := tap.Read(buf)

	fmt.Println(n, err)
}
