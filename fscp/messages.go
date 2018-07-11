package fscp

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// MessageVersion represents a message version.
type MessageVersion uint8

// MessageType represents a message type.
type MessageType uint8

const (
	// MessageVersion3 is the mandatory version 3 in messages.
	MessageVersion3 MessageVersion = 3

	// MessageTypeHelloRequest is a HELLO request message.
	MessageTypeHelloRequest MessageType = 0x00
	// MessageTypeHelloResponse is a HELLO response message.
	MessageTypeHelloResponse MessageType = 0x01
)

// Write a message header to the specified writer.
func writeHeader(b *bytes.Buffer, t MessageType, payloadSize int) {
	b.Grow(4 + payloadSize)
	binary.Write(b, binary.BigEndian, uint8(3))
	binary.Write(b, binary.BigEndian, t)
	binary.Write(b, binary.BigEndian, uint32(payloadSize))
}

func writeHelloMessage(b *bytes.Buffer, t MessageType, uniqueNumber uint32) {
	writeHeader(b, t, 4)
	binary.Write(b, binary.BigEndian, uniqueNumber)
}

func writeHelloRequest(b *bytes.Buffer, uniqueNumber uint32) {
	writeHelloMessage(b, MessageTypeHelloRequest, uniqueNumber)
}

func writeHelloResponse(b *bytes.Buffer, uniqueNumber uint32) {
	writeHelloMessage(b, MessageTypeHelloResponse, uniqueNumber)
}

func readHeader(b *bytes.Reader) (t MessageType, payloadSize int, err error) {
	if b.Len() < 4 {
		err = fmt.Errorf("unable to parse header: only %d byte(s) when %d or more were expected", b.Len(), 4)
		return
	}

	var version MessageVersion

	binary.Read(b, binary.BigEndian, &version)

	if version != MessageVersion3 {
		err = fmt.Errorf("error when parsing header: unexpected version %d when %d was expected", version, MessageVersion3)
		return
	}

	binary.Read(b, binary.BigEndian, &t)
	binary.Read(b, binary.BigEndian, &payloadSize)

	return
}

func readMessage(b *bytes.Reader) (t MessageType, msg interface{}, err error) {
	var payloadSize int

	if t, payloadSize, err = readHeader(b); err != nil {
		return
	} else if b.Len() < payloadSize {
		err = fmt.Errorf("error when parsing body: buffer is supposed to be at least %d byte(s) long but is only %d", payloadSize, b.Len())
		return
	}

	switch t {
	case MessageTypeHelloRequest, MessageTypeHelloResponse:
		msg = MessageHello{}
	default:
		err = fmt.Errorf("error when parsing body: unknown message type '%02x'", t)
		return
	}

	err = binary.Read(b, binary.BigEndian, &msg)

	return
}

// An UniqueNumber is a randomly generated number used during the HELLO exchange.
type UniqueNumber uint32

// MessageHello is a HELLO message.
type MessageHello struct {
	UniqueNumber UniqueNumber
}

// UnmarshallBinary unmarshalles the message as a binary stream.
func (m *MessageHello) UnmarshallBinary(b []byte) (err error) {
	if len(b) != 4 {
		return fmt.Errorf("buffer should be %d bytes long but is only %d", 4, len(b))
	}

	return binary.Read(bytes.NewReader(b), binary.BigEndian, &m.UniqueNumber)
}
