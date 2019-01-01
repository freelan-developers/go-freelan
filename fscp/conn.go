package fscp

import (
	"bytes"
	"crypto/elliptic"
	crand "crypto/rand"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
	"time"
)

type messageFrame struct {
	messageType MessageType
	message     interface{}
}

// Conn is a FSCP connection.
type Conn struct {
	writer                       io.Writer
	localAddr                    *Addr
	remoteAddr                   *Addr
	localHostIdentifier          HostIdentifier
	remoteHostIdentifier         *HostIdentifier
	security                     ClientSecurity
	currentOutgoingSessionNumber SessionNumber
	currentOutgoingCipherSuite   CipherSuite
	currentOutgoingEllipticCurve EllipticCurve
	currentIncomingSessionNumber SessionNumber
	currentIncomingCipherSuite   CipherSuite
	currentIncomingEllipticCurve EllipticCurve

	incoming   chan messageFrame
	connected  chan struct{}
	closed     chan struct{}
	closeError error
	once       sync.Once
}

func newConn(localAddr *Addr, remoteAddr *Addr, w io.Writer, hostIdentifier HostIdentifier, security ClientSecurity) *Conn {
	conn := &Conn{
		writer:              w,
		localAddr:           localAddr,
		remoteAddr:          remoteAddr,
		localHostIdentifier: hostIdentifier,
		security:            security,

		incoming:  make(chan messageFrame, 10),
		connected: make(chan struct{}),
		closed:    make(chan struct{}),
	}

	go conn.dispatchLoop()

	return conn
}

func (c *Conn) Read(b []byte) (n int, err error) {
	// TODO: Implement.
	return 0, nil
}

func (c *Conn) Write(b []byte) (n int, err error) {
	// TODO: Implement.
	return 0, nil
}

// Close closes the connection.
func (c *Conn) Close() error {
	return c.closeWithError(io.EOF)
}

func (c *Conn) debugPrintf(msg string, args ...interface{}) {
	debugPrintf("(%s <- %s) %s", c.LocalAddr(), c.RemoteAddr(), fmt.Sprintf(msg, args...))
}

// closeWithError closes the connection with the specified error.
func (c *Conn) closeWithError(err error) error {
	c.once.Do(func() {
		c.debugPrintf("closing connection: %s\n", err)

		c.closeError = err
		close(c.closed)
	})

	return c.closeError
}

func (c *Conn) warning(err error) {
	c.debugPrintf("Warning: %s\n", err.Error())
}

// LocalAddr returns the local address of the connection.
func (c *Conn) LocalAddr() net.Addr { return c.localAddr }

// RemoteAddr returns the remote address of the connection.
func (c *Conn) RemoteAddr() net.Addr { return c.remoteAddr }

// SetDeadline sets the deadline on the connection.
func (c *Conn) SetDeadline(t time.Time) error {
	// TODO: Implement.
	return nil
}

// SetReadDeadline sets the deadline on the connection.
func (c *Conn) SetReadDeadline(t time.Time) error {
	// TODO: Implement.
	return nil
}

// SetWriteDeadline sets the deadline on the connection.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	// TODO: Implement.
	return nil
}

func (c *Conn) writeMessage(messageType MessageType, message serializable) (err error) {
	// FIXME: If we know for sure that no two writeMessage() calls ever happen
	// concurrently, we can reuse the same buffer over and over (don't forget
	// to Reset() it).

	buf := &bytes.Buffer{}

	if err = writeMessage(buf, messageType, message); err != nil {
		return err
	}

	_, err = buf.WriteTo(c.writer)

	return
}

func (c *Conn) sendHelloRequest(uniqueNumber UniqueNumber) (err error) {
	msg := &messageHello{
		UniqueNumber: uniqueNumber,
	}

	c.debugPrintf("Sending %s.\n", msg)

	if err = c.writeMessage(MessageTypeHelloRequest, msg); err != nil {
		return err
	}

	return nil
}

func (c *Conn) sendHelloResponse(uniqueNumber UniqueNumber) error {
	msg := &messageHello{
		UniqueNumber: uniqueNumber,
	}

	c.debugPrintf("Sending %s.\n", msg)

	return c.writeMessage(MessageTypeHelloResponse, msg)
}

func (c *Conn) sendPresentation() error {
	msg := &messagePresentation{
		Certificate: c.security.Certificate,
	}

	c.debugPrintf("Sending %s.\n", msg)

	return c.writeMessage(MessageTypePresentation, msg)
}

func (c *Conn) sendSessionRequest(sessionNumber SessionNumber) error {
	msg := &messageSessionRequest{
		CipherSuites:   c.security.supportedCipherSuites(),
		EllipticCurves: c.security.supportedEllipticCurves(),
		HostIdentifier: c.localHostIdentifier,
		SessionNumber:  sessionNumber,
	}

	if err := msg.computeSignature(c.security); err != nil {
		return fmt.Errorf("failed to forge session request message: %s", err)
	}

	c.debugPrintf("Sending %s request.\n", msg)

	return c.writeMessage(MessageTypeSessionRequest, msg)
}

func (c *Conn) sendSession(sessionNumber SessionNumber) error {
	// TODO: Make this parameterizable.
	curve := elliptic.P521()
	d, x, y, err := elliptic.GenerateKey(curve, crand.Reader)

	if err != nil {
		return fmt.Errorf("failed to generate ECDHE key: %s", err)
	}

	privateKey := d
	// TODO: This is how you would compute the shared key: with x & y being the REMOTE keys.
	//publicKey, _ := curve.ScalarMult(x, y, d)
	publicKey := elliptic.Marshal(curve, x, y)

	// TODO: Store this.
	if privateKey == nil {
		panic(true)
	}

	msg := &messageSession{
		CipherSuite:    c.currentOutgoingCipherSuite,
		EllipticCurve:  c.currentOutgoingEllipticCurve,
		HostIdentifier: c.localHostIdentifier,
		SessionNumber:  sessionNumber,
		PublicKey:      publicKey,
	}

	if err := msg.computeSignature(c.security); err != nil {
		return fmt.Errorf("failed to forge session message: %s", err)
	}

	c.debugPrintf("Sending %s.\n", msg)

	return c.writeMessage(MessageTypeSession, msg)
}

func (c *Conn) dispatchLoop() {
	uniqueNumber := UniqueNumber(rand.Uint32())

	helloRequestRetrier := &Retrier{
		Operation: func() error {
			return c.sendHelloRequest(uniqueNumber)
		},
		OnFailure: func(err error) {
			c.closeWithError(err)
		},
		Period: time.Second * 3,
	}

	helloRequestRetrier.Start()
	defer helloRequestRetrier.Stop()

	for {
		select {
		case frame := <-c.incoming:
			switch imsg := frame.message.(type) {
			case *messageHello:
				switch frame.messageType {
				case MessageTypeHelloRequest:
					c.debugPrintf("Received %s request.\n", imsg)

					if err := c.sendHelloResponse(imsg.UniqueNumber); err != nil {
						c.closeWithError(err)
						return
					}

				case MessageTypeHelloResponse:
					c.debugPrintf("Received %s response.\n", imsg)

					if imsg.UniqueNumber != uniqueNumber {
						// The received response does not match the outstanding
						// hello request. Ignoring.
						continue
					}

					if !helloRequestRetrier.Stop() {
						// The retrier was stopped already, so we do nothing.
						continue
					}

					if err := c.sendPresentation(); err != nil {
						c.closeWithError(err)
						return
					}
				}
			case *messagePresentation:
				switch frame.messageType {
				case MessageTypePresentation:
					c.debugPrintf("Received %s.\n", imsg)

					//TODO: Check if the certificate is acceptable.

					if imsg.Certificate != nil {
						// If we receive a presentation message, store its
						// certificate only if we don't have one already.
						c.security.RemoteCertificate = imsg.Certificate
						c.debugPrintf("Stored certificate (%s) for remote host.\n", imsg.Certificate.Subject)
					}

					if err := c.sendSessionRequest(c.currentIncomingSessionNumber); err != nil {
						c.closeWithError(err)
						return
					}
				}
			case *messageSessionRequest:
				c.debugPrintf("Received %s.\n", imsg)

				if err := imsg.verifySignature(c.security); err != nil {
					c.warning(fmt.Errorf("session request signature verification failed: %s", err))
					continue
				}

				//TODO: Filter out some hosts based on a callback or other client logic.

				cipherSuite, err := c.security.supportedCipherSuites().FindCommon(imsg.CipherSuites)

				if err != nil {
					c.warning(fmt.Errorf("ignoring session request: %s", err))
					continue
				}

				ellipticCurve, err := c.security.supportedEllipticCurves().FindCommon(imsg.EllipticCurves)

				if err != nil {
					c.warning(fmt.Errorf("ignoring session request: %s", err))
					continue
				}

				if c.currentOutgoingCipherSuite != NullCipherSuite && cipherSuite != c.currentOutgoingCipherSuite {
					c.warning(fmt.Errorf("ignoring session request: refusing to change cipher suite from %s to %s", c.currentOutgoingCipherSuite, cipherSuite))
					continue
				}

				if c.currentOutgoingEllipticCurve != NullEllipticCurve && ellipticCurve != c.currentOutgoingEllipticCurve {
					c.warning(fmt.Errorf("ignoring session request: refusing to change elliptic curve from %s to %s", c.currentOutgoingEllipticCurve, ellipticCurve))
					continue
				}

				c.currentOutgoingCipherSuite = cipherSuite
				c.currentOutgoingEllipticCurve = ellipticCurve

				c.debugPrintf("Selected cipher suite: %s.\n", cipherSuite)
				c.debugPrintf("Selected elliptic curve: %s.\n", ellipticCurve)

				if err := c.sendSession(c.currentOutgoingSessionNumber); err != nil {
					c.closeWithError(err)
					return
				}
			case *messageSession:
				c.debugPrintf("Received %s.\n", imsg)
			default:
				c.debugPrintf("Received %s.\n", frame.message)
			}
		case <-c.closed:
			return
		}
	}

	//close(c.connected)

	// TODO: Wait for the reply.
}
