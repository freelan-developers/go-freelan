package fscp

import (
	"bytes"
	"crypto/x509"
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
	// MessageTypePresentation is a PRESENTATION message.
	MessageTypePresentation MessageType = 0x02
	// MessageTypeSessionRequest is a SESSION REQUEST message.
	MessageTypeSessionRequest MessageType = 0x03
)

func (m MessageType) String() string {
	switch m {
	case MessageTypeHelloRequest:
		return "HELLO (request)"
	case MessageTypeHelloResponse:
		return "HELLO (response)"
	case MessageTypePresentation:
		return "PRESENTATION"
	case MessageTypeSessionRequest:
		return "SESSION (request)"
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

func writeMessage(b *bytes.Buffer, t MessageType, msg serializable) {
	writeHeader(b, t, msg.serializationSize())
	msg.serialize(b)
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

func readMessage(b *bytes.Reader) (t MessageType, msg deserializable, err error) {
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
	case MessageTypePresentation:
		msg = &messagePresentation{}
	case MessageTypeSessionRequest:
		msg = &messageSessionRequest{}
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

// messagePresentation is a HELLO message.
type messagePresentation struct {
	Certificate *x509.Certificate
}

func (m *messagePresentation) serialize(b *bytes.Buffer) error {
	if m.Certificate == nil {
		return binary.Write(b, binary.BigEndian, uint16(0))
	}

	binary.Write(b, binary.BigEndian, uint16(len(m.Certificate.Raw)))
	b.Write(m.Certificate.Raw)

	return nil
}

func (m *messagePresentation) serializationSize() int {
	if m.Certificate == nil {
		return 2
	}

	return 2 + len(m.Certificate.Raw)
}

func (m *messagePresentation) deserialize(b *bytes.Reader) (err error) {
	if b.Len() < 2 {
		return fmt.Errorf("buffer should be at least %d bytes long but is %d", 2, b.Len())
	}

	var size uint16

	binary.Read(b, binary.BigEndian, &size)

	if size == 0 {
		m.Certificate = nil
	} else {
		if b.Len() < int(size) {
			return fmt.Errorf("buffer should be at least %d bytes long but is %d", 2+size, 2+b.Len())
		}

		der := make([]byte, int(size))

		if _, err = b.Read(der); err != nil {
			return
		}

		m.Certificate, err = x509.ParseCertificate(der)
	}

	return
}

// SessionNumber represents a session number.
type SessionNumber uint32

// HostIdentifier represents a host identifier.
type HostIdentifier uint32

// CipherSuite represents a cipher suite.
type CipherSuite uint8

const (
	// CipherSuiteECDHERSAAES128GCMSHA256 is the ECDHE-RSA-AES-128-GCM-SHA256 cipher suite.
	CipherSuiteECDHERSAAES128GCMSHA256 = 0x01
	// CipherSuiteECDHERSAAES256GCMSHA384 is the ECDHE-RSA-AES-256-GCM-SHA384 cipher suite.
	CipherSuiteECDHERSAAES256GCMSHA384 = 0x02
)

// EllipticCurve represents an elliptic curve.
type EllipticCurve uint8

const (
	// EllipticCurveSECT571K1 is the SECT571K1 elliptic curve.
	EllipticCurveSECT571K1 = 0x01
	// EllipticCurveSECP384R1 is the SECP384R1 elliptic curve.
	EllipticCurveSECP384R1 = 0x02
	// EllipticCurveSECP521R1 is the SECP521R1 elliptic curve.
	EllipticCurveSECP521R1 = 0x03
)

type messageSessionRequest struct {
	SessionNumber  SessionNumber
	HostIdentifier HostIdentifier
	CipherSuites   []CipherSuite
	EllipticCurves []EllipticCurve
	Signature      []byte
}

func (m *messageSessionRequest) serialize(b *bytes.Buffer) error {
	binary.Write(b, binary.BigEndian, m.SessionNumber)
	binary.Write(b, binary.BigEndian, m.HostIdentifier)
	binary.Write(b, binary.BigEndian, uint16(len(m.CipherSuites)))

	for _, cipherSuite := range m.CipherSuites {
		binary.Write(b, binary.BigEndian, cipherSuite)
	}

	binary.Write(b, binary.BigEndian, uint16(len(m.EllipticCurves)))

	for _, ellipticCurve := range m.EllipticCurves {
		binary.Write(b, binary.BigEndian, ellipticCurve)
	}

	binary.Write(b, binary.BigEndian, uint16(len(m.Signature)))
	b.Write(m.Signature)

	return nil
}

func (m *messageSessionRequest) serializationSize() int {
	return 4 + 4 + 2 + len(m.CipherSuites) + 2 + len(m.EllipticCurves) + 2 + len(m.Signature)
}

func (m *messageSessionRequest) deserialize(b *bytes.Reader) (err error) {
	if b.Len() < 10 {
		return fmt.Errorf("buffer should be at least %d bytes long but is %d", 10, b.Len())
	}

	binary.Read(b, binary.BigEndian, &m.SessionNumber)
	binary.Read(b, binary.BigEndian, &m.HostIdentifier)

	var size uint16

	binary.Read(b, binary.BigEndian, &size)

	m.CipherSuites = make([]CipherSuite, size)

	for i := range m.CipherSuites {
		binary.Read(b, binary.BigEndian, &m.CipherSuites[i])
	}

	binary.Read(b, binary.BigEndian, &size)

	m.EllipticCurves = make([]EllipticCurve, size)

	for i := range m.EllipticCurves {
		binary.Read(b, binary.BigEndian, &m.EllipticCurves[i])
	}

	binary.Read(b, binary.BigEndian, &size)

	m.Signature = make([]byte, size)
	_, err = b.Read(m.Signature)

	return
}
