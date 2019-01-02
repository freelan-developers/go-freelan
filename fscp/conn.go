package fscp

import (
	"bytes"
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
	writer               io.Writer
	localAddr            *Addr
	remoteAddr           *Addr
	localHostIdentifier  HostIdentifier
	remoteHostIdentifier *HostIdentifier
	security             ClientSecurity
	session              *Session
	nextSession          *Session

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

	if buf.Len() != message.serializationSize()+4 {
		panic(fmt.Errorf("expected buffer of size %d but was %d byte(s) long", message.serializationSize()+4, buf.Len()))
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

func (c *Conn) sendSession(session *Session) error {
	msg := &messageSession{
		CipherSuite:    session.CipherSuite,
		EllipticCurve:  session.EllipticCurve,
		HostIdentifier: c.localHostIdentifier,
		SessionNumber:  session.SessionNumber,
		PublicKey:      session.PublicKey,
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

					if c.security.RemoteClientSecurity == nil {
						remoteClientSecurity := &RemoteClientSecurity{}

						if imsg.Certificate != nil {
							// If we receive a presentation message, store its
							// certificate only if we don't have one already.
							remoteClientSecurity.Certificate = imsg.Certificate
							c.debugPrintf("Stored certificate (%s) for remote host.\n", imsg.Certificate.Subject)
						} else {
							c.debugPrintf("Using pre-shared key for remote host.\n")
						}

						c.security.RemoteClientSecurity = remoteClientSecurity
					} else {
						c.debugPrintf("Ignoring repeated presentation for remote host.\n")

						continue
					}

					var sessionNumber SessionNumber = 1

					// If we have an existing next session, use the next session number.
					if c.nextSession != nil {
						sessionNumber = c.nextSession.SessionNumber
					}

					if err := c.sendSessionRequest(sessionNumber); err != nil {
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

				if c.remoteHostIdentifier == nil {
					c.remoteHostIdentifier = &imsg.HostIdentifier
					c.debugPrintf("Setting remote host identifier: %s\n", imsg.HostIdentifier)
				} else if imsg.HostIdentifier != *c.remoteHostIdentifier {
					c.warning(fmt.Errorf("ignoring session request because host identifier does not match: expected %s but got %s", *c.remoteHostIdentifier, imsg.HostIdentifier))
					continue
				}

				if c.session != nil && c.session.SessionNumber >= imsg.SessionNumber {
					c.debugPrintf("Session request is for an oudated session (%d): resending current session (%d).\n", imsg.SessionNumber, c.session.SessionNumber)

					// The session request is oudated: we resend the current session.
					if err := c.sendSession(c.session); err != nil {
						c.closeWithError(err)
						return
					}

					continue
				}

				cipherSuite := c.security.supportedCipherSuites().FindCommon(imsg.CipherSuites)
				ellipticCurve := c.security.supportedEllipticCurves().FindCommon(imsg.EllipticCurves)
				session, err := NewSession(imsg.SessionNumber, cipherSuite, ellipticCurve)

				if err != nil {
					c.warning(fmt.Errorf("failed to initialize new session: %s", err))

					if err := c.sendSession(session); err != nil {
						c.closeWithError(err)
						return
					}

					continue
				}

				c.debugPrintf("Session number: %d.\n", session.SessionNumber)
				c.debugPrintf("Selected cipher suite: %s.\n", session.CipherSuite)
				c.debugPrintf("Selected elliptic curve: %s.\n", session.EllipticCurve)

				c.nextSession = session

				if err := c.sendSession(session); err != nil {
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
}
