package fscp

import (
	"bytes"
	"crypto/rand"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io"
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

type lenReader interface {
	io.Reader
	Len() int
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

func readHeader(b lenReader) (t MessageType, payloadSize int, err error) {
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

	if err = binary.Read(b, binary.BigEndian, &t); err != nil {
		err = fmt.Errorf("reading message type: %s", err)
		return
	}

	if err = binary.Read(b, binary.BigEndian, &size); err != nil {
		err = fmt.Errorf("reading payload size: %s", err)
		return
	}

	payloadSize = int(size)

	return
}

func readMessage(b lenReader) (t MessageType, msg deserializable, err error) {
	var payloadSize int

	if t, payloadSize, err = readHeader(b); err != nil {
		err = fmt.Errorf("parsing header: %s", err)
		return
	} else if b.Len() < payloadSize {
		err = fmt.Errorf("parsing body: buffer is supposed to be at least %d byte(s) long but is only %d", payloadSize, b.Len())
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
			err = fmt.Errorf("parsing body: unknown message type '%02x'", t)
			return
		}
	}

	if err = msg.deserialize(b); err != nil {
		err = fmt.Errorf("failed to deserialize %s message: %s", t, err)
	}

	return
}

type serializable interface {
	serializationSize() int
	serialize(io.Writer) error
}

type deserializable interface {
	deserialize(lenReader) error
}

// An UniqueNumber is a randomly generated number used during the HELLO exchange.
type UniqueNumber uint32

// messageHello is a HELLO message.
type messageHello struct {
	UniqueNumber UniqueNumber
}

func (m *messageHello) serialize(b io.Writer) error {
	return binary.Write(b, binary.BigEndian, &m.UniqueNumber)
}

func (m *messageHello) serializationSize() int { return 4 }

func (m *messageHello) deserialize(b lenReader) (err error) {
	if b.Len() != 4 {
		return fmt.Errorf("buffer should be %d bytes long but is %d", 4, b.Len())
	}

	return binary.Read(b, binary.BigEndian, &m.UniqueNumber)
}

func (m *messageHello) String() string {
	return fmt.Sprintf("HELLO [unique_number:%08x]", m.UniqueNumber)
}

// messagePresentation is a HELLO message.
type messagePresentation struct {
	Certificate *x509.Certificate
}

func (m *messagePresentation) serialize(b io.Writer) error {
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

func (m *messagePresentation) deserialize(b lenReader) (err error) {
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

		if err = binary.Read(b, binary.BigEndian, der); err != nil {
			return
		}

		m.Certificate, err = x509.ParseCertificate(der)
	}

	return
}

func (m *messagePresentation) String() string {
	if m.Certificate != nil {
		return fmt.Sprintf("PRESENTATION [cert:%s]", m.Certificate.Subject)
	}

	return fmt.Sprintf("PRESENTATION [cert:]")
}

// SessionNumber represents a session number.
type SessionNumber uint32

// HostIdentifier represents a host identifier.
type HostIdentifier [32]byte

// GenerateHostIdentifier generates a new random host identifier.
func GenerateHostIdentifier() (result HostIdentifier, err error) {
	var n int

	if n, err = rand.Read(result[:]); n != len(result) && err != nil {
		err = fmt.Errorf("generating a random host identifier: %s", err)
	}

	return
}

type messageSessionRequest struct {
	SessionNumber  SessionNumber
	HostIdentifier HostIdentifier
	CipherSuites   CipherSuiteSlice
	EllipticCurves EllipticCurveSlice
	Signature      []byte
}

func (m *messageSessionRequest) computeSignature(signer Signer) (err error) {
	buf := &bytes.Buffer{}

	if err = m.serializeUnsigned(buf); err != nil {
		return fmt.Errorf("serializing unsigned payload: %s", err)
	}

	if m.Signature, err = signer.Sign(buf.Bytes()); err != nil {
		return fmt.Errorf("generating signature: %s", err)
	}

	return nil
}

func (m *messageSessionRequest) verifySignature(verifier Verifier) (err error) {
	buf := &bytes.Buffer{}

	if err = m.serializeUnsigned(buf); err != nil {
		return fmt.Errorf("serializing unsigned payload: %s", err)
	}

	if err = verifier.Verify(buf.Bytes(), m.Signature); err != nil {
		return fmt.Errorf("verifying signature: %s", err)
	}

	return nil
}

func (m *messageSessionRequest) serializeUnsigned(b io.Writer) (err error) {
	if err = binary.Write(b, binary.BigEndian, m.SessionNumber); err != nil {
		return fmt.Errorf("writing session number: %s", err)
	}

	if err = binary.Write(b, binary.BigEndian, m.HostIdentifier); err != nil {
		return fmt.Errorf("writing host identifier: %s", err)
	}

	if err = binary.Write(b, binary.BigEndian, uint16(len(m.CipherSuites))); err != nil {
		return fmt.Errorf("writing cipher suites size: %s", err)
	}

	for i, cipherSuite := range m.CipherSuites {
		if err = binary.Write(b, binary.BigEndian, cipherSuite); err != nil {
			return fmt.Errorf("writing cipher suite %d (out of %d): %s", i, len(m.CipherSuites), err)
		}
	}

	if err = binary.Write(b, binary.BigEndian, uint16(len(m.EllipticCurves))); err != nil {
		return fmt.Errorf("writing elliptic curves size: %s", err)
	}

	for i, ellipticCurve := range m.EllipticCurves {
		if err = binary.Write(b, binary.BigEndian, ellipticCurve); err != nil {
			return fmt.Errorf("writing elliptic curve %d (out of %d): %s", i, len(m.EllipticCurves), err)
		}
	}

	return nil
}

func (m *messageSessionRequest) serialize(b io.Writer) (err error) {
	if err = m.serializeUnsigned(b); err != nil {
		return err
	}

	if err = binary.Write(b, binary.BigEndian, uint16(len(m.Signature))); err != nil {
		return fmt.Errorf("writing signature size: %s", err)
	}

	if err = binary.Write(b, binary.BigEndian, m.Signature); err != nil {
		return fmt.Errorf("writing signature: %s", err)
	}

	return nil
}

func (m *messageSessionRequest) serializationSize() int {
	return 4 + 4 + 2 + len(m.CipherSuites) + 2 + len(m.EllipticCurves) + 2 + len(m.Signature)
}

func (m *messageSessionRequest) deserialize(b lenReader) (err error) {
	if b.Len() < 10 {
		return fmt.Errorf("buffer should be at least %d bytes long but is %d", 10, b.Len())
	}

	if err = binary.Read(b, binary.BigEndian, &m.SessionNumber); err != nil {
		return fmt.Errorf("reading session number: %s", err)
	}

	if err = binary.Read(b, binary.BigEndian, &m.HostIdentifier); err != nil {
		return fmt.Errorf("reading host identifier: %s", err)
	}

	var size uint16

	if err = binary.Read(b, binary.BigEndian, &size); err != nil {
		return fmt.Errorf("reading cipher suite size: %s", err)
	}

	m.CipherSuites = make(CipherSuiteSlice, size)

	for i := range m.CipherSuites {
		if err = binary.Read(b, binary.BigEndian, &m.CipherSuites[i]); err != nil {
			return fmt.Errorf("reading cipher suite %d (of %d): %s", i, len(m.CipherSuites), err)
		}
	}

	if err = binary.Read(b, binary.BigEndian, &size); err != nil {
		return fmt.Errorf("reading elliptic curves size: %s", err)
	}

	m.EllipticCurves = make(EllipticCurveSlice, size)

	for i := range m.EllipticCurves {
		if err = binary.Read(b, binary.BigEndian, &m.EllipticCurves[i]); err != nil {
			return fmt.Errorf("reading elliptic curve %d (of %d): %s", i, len(m.EllipticCurves), err)
		}
	}

	if err = binary.Read(b, binary.BigEndian, &size); err != nil {
		return fmt.Errorf("reading signature size: %s", err)
	}

	if size == 0 {
		m.Signature = nil
	} else {
		m.Signature = make([]byte, size)
		if err = binary.Read(b, binary.BigEndian, m.Signature); err != nil {
			return fmt.Errorf("reading signature: %s", err)
		}
	}

	return
}

func (m *messageSessionRequest) String() string {
	return fmt.Sprintf("SESSION_REQUEST [sid:%08x,hid:%08x,ciphers:%s,curves:%s]", m.SessionNumber, m.HostIdentifier, m.CipherSuites, m.EllipticCurves)
}

type messageSession struct {
	SessionNumber  SessionNumber
	HostIdentifier HostIdentifier
	CipherSuite    CipherSuite
	EllipticCurve  EllipticCurve
	PublicKey      []byte
	Signature      []byte
}

func (m *messageSession) computeSignature(signer Signer) (err error) {
	buf := &bytes.Buffer{}

	if err = m.serializeUnsigned(buf); err != nil {
		return fmt.Errorf("serializing unsigned payload: %s", err)
	}

	if m.Signature, err = signer.Sign(buf.Bytes()); err != nil {
		return fmt.Errorf("generating signature: %s", err)
	}

	return nil
}

func (m *messageSession) verifySignature(verifier Verifier) (err error) {
	buf := &bytes.Buffer{}

	if err = m.serializeUnsigned(buf); err != nil {
		return fmt.Errorf("serializing unsigned payload: %s", err)
	}

	if err = verifier.Verify(buf.Bytes(), m.Signature); err != nil {
		return fmt.Errorf("verifying signature: %s", err)
	}

	return nil
}

func (m *messageSession) serializeUnsigned(b io.Writer) error {
	if err := binary.Write(b, binary.BigEndian, m.SessionNumber); err != nil {
		return fmt.Errorf("writing session number: %s", err)
	}

	if err := binary.Write(b, binary.BigEndian, m.HostIdentifier); err != nil {
		return fmt.Errorf("writing host identifier: %s", err)
	}

	if err := binary.Write(b, binary.BigEndian, m.CipherSuite); err != nil {
		return fmt.Errorf("writing cipher suite: %s", err)
	}

	if err := binary.Write(b, binary.BigEndian, m.EllipticCurve); err != nil {
		return fmt.Errorf("writing elliptic curve: %s", err)
	}

	// These two bytes are always zero.
	if err := binary.Write(b, binary.BigEndian, []byte{0x00, 0x00}); err != nil {
		return fmt.Errorf("writing null bytes: %s", err)
	}

	if err := binary.Write(b, binary.BigEndian, uint16(len(m.PublicKey))); err != nil {
		return fmt.Errorf("writing public key length: %s", err)
	}

	if err := binary.Write(b, binary.BigEndian, m.PublicKey); err != nil {
		return fmt.Errorf("writing public key: %s", err)
	}

	return nil
}

func (m *messageSession) serialize(b io.Writer) error {
	if err := m.serializeUnsigned(b); err != nil {
		return err
	}

	if err := binary.Write(b, binary.BigEndian, uint16(len(m.Signature))); err != nil {
		return fmt.Errorf("writing signature length: %s", err)
	}

	if err := binary.Write(b, binary.BigEndian, m.Signature); err != nil {
		return fmt.Errorf("writing signature: %s", err)
	}

	return nil
}

func (m *messageSession) serializationSize() int {
	return 4 + 4 + 2 + 2 + 2 + len(m.PublicKey) + 2 + len(m.Signature)
}

func (m *messageSession) deserialize(b lenReader) (err error) {
	if b.Len() < 16 {
		return fmt.Errorf("buffer should be at least %d bytes long but is %d", 16, b.Len())
	}

	if err = binary.Read(b, binary.BigEndian, &m.SessionNumber); err != nil {
		return fmt.Errorf("reading session number: %s", err)
	}

	if err = binary.Read(b, binary.BigEndian, &m.HostIdentifier); err != nil {
		return fmt.Errorf("reading host identifier: %s", err)
	}

	if err = binary.Read(b, binary.BigEndian, &m.CipherSuite); err != nil {
		return fmt.Errorf("reading cipher suite: %s", err)
	}

	if err = binary.Read(b, binary.BigEndian, &m.EllipticCurve); err != nil {
		return fmt.Errorf("reading elliptic curve: %s", err)
	}

	// Discard two bytes.
	var n int

	if n, err = b.Read([]byte{0x00, 0x00}); n != 2 {
		if err != nil {
			return fmt.Errorf("failing to read unused bytes: %s", err)
		}

		return fmt.Errorf("failing to read unused bytes: read only %d but %d were expected", n, 2)
	}

	var size uint16

	if err = binary.Read(b, binary.BigEndian, &size); err != nil {
		return fmt.Errorf("reading public key size: %s", err)
	}

	m.PublicKey = make([]byte, size)

	if err = binary.Read(b, binary.BigEndian, m.PublicKey); err != nil {
		return fmt.Errorf("reading public key: %s", err)
	}

	if err = binary.Read(b, binary.BigEndian, &size); err != nil {
		return fmt.Errorf("reading signature size: %s", err)
	}

	m.Signature = make([]byte, size)

	if err = binary.Read(b, binary.BigEndian, m.Signature); err != nil {
		return fmt.Errorf("reading signature: %s", err)
	}

	return
}

func (m *messageSession) String() string {
	return fmt.Sprintf("SESSION [sid:%08x,hid:%08x,cipher:%s,curve:%s]", m.SessionNumber, m.HostIdentifier, m.CipherSuite, m.EllipticCurve)
}

// A SequenceNumber is a 4 bytes sequence number.
type SequenceNumber uint32

type messageData struct {
	Channel        uint8
	SequenceNumber SequenceNumber
	GCMTag         [16]byte
	Ciphertext     []byte
}

func (m *messageData) serialize(b io.Writer) (err error) {
	if err = binary.Write(b, binary.BigEndian, m.SequenceNumber); err != nil {
		return fmt.Errorf("writing sequence number: %s", err)
	}

	if err = binary.Write(b, binary.BigEndian, m.GCMTag); err != nil {
		return fmt.Errorf("writing GCM tag: %s", err)
	}

	if err = binary.Write(b, binary.BigEndian, uint16(len(m.Ciphertext))); err != nil {
		return fmt.Errorf("writing ciphertext size: %s", err)
	}

	if err = binary.Write(b, binary.BigEndian, m.Ciphertext); err != nil {
		return fmt.Errorf("writing ciphertext: %s", err)
	}

	return nil
}

func (m *messageData) serializationSize() int {
	return 4 + 16 + 2 + len(m.Ciphertext)
}

func (m *messageData) deserialize(b lenReader) (err error) {
	if b.Len() < 22 {
		return fmt.Errorf("buffer should be at least %d bytes long but is %d", 22, b.Len())
	}

	if err = binary.Read(b, binary.BigEndian, &m.SequenceNumber); err != nil {
		return fmt.Errorf("reading sequence number: %s", err)
	}

	if err = binary.Read(b, binary.BigEndian, &m.GCMTag); err != nil {
		return fmt.Errorf("reading GCM tag: %s", err)
	}

	var size uint16

	if err = binary.Read(b, binary.BigEndian, &size); err != nil {
		return fmt.Errorf("reading ciphertext size: %s", err)
	}

	m.Ciphertext = make([]byte, size)

	if err = binary.Read(b, binary.BigEndian, m.Ciphertext); err != nil {
		return fmt.Errorf("reading ciphertext: %s", err)
	}

	return
}

func (m *messageData) String() string {
	return fmt.Sprintf("DATA [ch:%1x,seq:%08x,clen:%d]", m.Channel, m.SequenceNumber, len(m.Ciphertext))
}
