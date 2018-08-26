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
	// MessageTypeSession is a SESSION message.
	MessageTypeSession MessageType = 0x04
	// MessageTypeData is a DATA message.
	MessageTypeData = 0x70
	// MessageTypeContactRequest is a CONTACT REQUEST message.
	MessageTypeContactRequest = 0xfd
	// MessageTypeContact is a CONTACT message.
	MessageTypeContact = 0xfe
	// MessageTypeKeepAlive is a KEEP-ALIVE message.
	MessageTypeKeepAlive = 0xff
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
	case MessageTypeSession:
		return "SESSION"
	case MessageTypeData:
		return "DATA"
	case MessageTypeContactRequest:
		return "CONTACT (request)"
	case MessageTypeContact:
		return "CONTACT"
	case MessageTypeKeepAlive:
		return "KEEP-ALIVE"
	default:
		return "unknown message type"
	}
}

// Write a message header to the specified writer.
func writeHeader(b *bytes.Buffer, t MessageType, payloadSize int) (err error) {
	b.Grow(4 + payloadSize)

	if err = binary.Write(b, binary.BigEndian, uint8(3)); err != nil {
		return err
	}

	if err = binary.Write(b, binary.BigEndian, t); err != nil {
		return err
	}

	if err = binary.Write(b, binary.BigEndian, uint16(payloadSize)); err != nil {
		return err
	}

	return nil
}

func writeMessage(b *bytes.Buffer, t MessageType, msg serializable) (err error) {
	if err = writeHeader(b, t, msg.serializationSize()); err != nil {
		return err
	}

	return msg.serialize(b)
}

func writeDataMessage(b *bytes.Buffer, msg *messageData) error {
	return writeMessage(b, MessageTypeData+MessageType(msg.Channel), msg)
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

	if t&MessageTypeData == MessageTypeData {
		msg = &messageData{
			Channel: uint8(t - MessageTypeData),
		}
	} else {
		switch t {
		case MessageTypeHelloRequest, MessageTypeHelloResponse:
			msg = &messageHello{}
		case MessageTypePresentation:
			msg = &messagePresentation{}
		case MessageTypeSessionRequest:
			msg = &messageSessionRequest{}
		case MessageTypeSession:
			msg = &messageSession{}
		case MessageTypeContactRequest, MessageTypeContact, MessageTypeKeepAlive:
			msg = &messageData{
				Channel: 0,
			}
		default:
			err = fmt.Errorf("error when parsing body: unknown message type '%02x'", t)
			return
		}
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

type messageSessionRequest struct {
	SessionNumber  SessionNumber
	HostIdentifier HostIdentifier
	CipherSuites   []CipherSuite
	EllipticCurves []EllipticCurve
	Signature      []byte
}

func (m *messageSessionRequest) computeSignature() error {
	return nil
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

type messageSession struct {
	SessionNumber  SessionNumber
	HostIdentifier HostIdentifier
	CipherSuite    CipherSuite
	EllipticCurve  EllipticCurve
	PublicKey      []byte
	Signature      []byte
}

func (m *messageSession) serialize(b *bytes.Buffer) error {
	binary.Write(b, binary.BigEndian, m.SessionNumber)
	binary.Write(b, binary.BigEndian, m.HostIdentifier)
	binary.Write(b, binary.BigEndian, m.CipherSuite)
	binary.Write(b, binary.BigEndian, m.EllipticCurve)

	// These two bytes are always zero.
	b.Write([]byte{0x00, 0x00})

	binary.Write(b, binary.BigEndian, uint16(len(m.PublicKey)))
	b.Write(m.PublicKey)

	binary.Write(b, binary.BigEndian, uint16(len(m.Signature)))
	b.Write(m.Signature)

	return nil
}

func (m *messageSession) serializationSize() int {
	return 4 + 4 + 2 + 2 + 2 + len(m.PublicKey) + 2 + len(m.Signature)
}

func (m *messageSession) deserialize(b *bytes.Reader) (err error) {
	if b.Len() < 16 {
		return fmt.Errorf("buffer should be at least %d bytes long but is %d", 16, b.Len())
	}

	binary.Read(b, binary.BigEndian, &m.SessionNumber)
	binary.Read(b, binary.BigEndian, &m.HostIdentifier)
	binary.Read(b, binary.BigEndian, &m.CipherSuite)
	binary.Read(b, binary.BigEndian, &m.EllipticCurve)

	// Discard two bytes.
	b.Read([]byte{0x00, 0x00})

	var size uint16

	binary.Read(b, binary.BigEndian, &size)

	m.PublicKey = make([]byte, size)
	_, err = b.Read(m.PublicKey)

	binary.Read(b, binary.BigEndian, &size)

	m.Signature = make([]byte, size)
	_, err = b.Read(m.Signature)

	return
}

// A SequenceNumber is a 4 bytes sequence number.
type SequenceNumber uint32

type messageData struct {
	Channel        uint8
	SequenceNumber SequenceNumber
	GCMTag         [16]byte
	Ciphertext     []byte
}

func (m *messageData) serialize(b *bytes.Buffer) error {
	binary.Write(b, binary.BigEndian, m.SequenceNumber)
	b.Write(m.GCMTag[:])

	binary.Write(b, binary.BigEndian, uint16(len(m.Ciphertext)))
	b.Write(m.Ciphertext)

	return nil
}

func (m *messageData) serializationSize() int {
	return 4 + 16 + 2 + len(m.Ciphertext)
}

func (m *messageData) deserialize(b *bytes.Reader) (err error) {
	if b.Len() < 22 {
		return fmt.Errorf("buffer should be at least %d bytes long but is %d", 22, b.Len())
	}

	binary.Read(b, binary.BigEndian, &m.SequenceNumber)
	_, err = b.Read(m.GCMTag[:])

	var size uint16

	binary.Read(b, binary.BigEndian, &size)

	m.Ciphertext = make([]byte, size)
	_, err = b.Read(m.Ciphertext)

	return
}
