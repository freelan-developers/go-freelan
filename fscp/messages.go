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

func (m MessageType) String() string {
	switch m {
	case MessageTypeHelloRequest:
		return "HELLO-request"
	case MessageTypeHelloResponse:
		return "HELLO-response"
	default:
		return "unknown message type"
	}
}

// Write a message header to the specified writer.
func writeHeader(b *bytes.Buffer, t MessageType, payloadSize int) {
	b.Grow(4 + payloadSize)
	binary.Write(b, binary.BigEndian, uint8(3))
	binary.Write(b, binary.BigEndian, t)
	binary.Write(b, binary.BigEndian, uint16(payloadSize))
}

func writeHelloMessage(b *bytes.Buffer, t MessageType, msg *messageHello) {
	writeHeader(b, t, msg.serializationSize())
	msg.serialize(b)
}

func writeHelloRequest(b *bytes.Buffer, msg *messageHello) {
	writeHelloMessage(b, MessageTypeHelloRequest, msg)
}

func writeHelloResponse(b *bytes.Buffer, msg *messageHello) {
	writeHelloMessage(b, MessageTypeHelloResponse, msg)
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

	var size uint16

	binary.Read(b, binary.BigEndian, &t)
	binary.Read(b, binary.BigEndian, &size)

	payloadSize = int(size)

	return
}

func readMessage(b *bytes.Reader) (t MessageType, msg genericMessage, err error) {
	var payloadSize int

	if t, payloadSize, err = readHeader(b); err != nil {
		return
	} else if b.Len() < payloadSize {
		err = fmt.Errorf("error when parsing body: buffer is supposed to be at least %d byte(s) long but is only %d", payloadSize, b.Len())
		return
	}

	switch t {
	case MessageTypeHelloRequest, MessageTypeHelloResponse:
		msg = &messageHello{}
	default:
		err = fmt.Errorf("error when parsing body: unknown message type '%02x'", t)
		return
	}

	err = msg.deserialize(b)

	return
}

type serializable interface {
	serializationSize() int
	serialize(*bytes.Buffer) error
}

type deserializable interface {
	deserialize(*bytes.Reader) error
}

type genericMessage interface {
	serializable
	deserializable
}

// An UniqueNumber is a randomly generated number used during the HELLO exchange.
type UniqueNumber uint32

// messageHello is a HELLO message.
type messageHello struct {
	UniqueNumber UniqueNumber
}

func (m *messageHello) serialize(b *bytes.Buffer) error {
	return binary.Write(b, binary.BigEndian, &m.UniqueNumber)
}

func (m *messageHello) serializationSize() int { return 4 }

func (m *messageHello) deserialize(b *bytes.Reader) (err error) {
	if b.Len() != 4 {
		return fmt.Errorf("buffer should be %d bytes long but is %d", 4, b.Len())
	}

	return binary.Read(b, binary.BigEndian, &m.UniqueNumber)
}
