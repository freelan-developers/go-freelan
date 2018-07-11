package fscp

import (
	"bytes"
	"testing"
)

func TestSerialization(t *testing.T) {
	buf := &bytes.Buffer{}
	msg := messageHello{}

	writeHelloRequest(buf, msg)
}
