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
	localAddr            *Addr
	remoteAddr           *Addr
	writer               io.Writer
	hostIdentifier       HostIdentifier
	remoteHostIdentifier *HostIdentifier
	security             ClientSecurity
	currentSessionNumber SessionNumber

	incoming   chan messageFrame
	connected  chan struct{}
	closed     chan struct{}
	closeError error
	once       sync.Once
}

func newConn(localAddr *Addr, remoteAddr *Addr, w io.Writer, security ClientSecurity) *Conn {
	conn := &Conn{
		localAddr:      localAddr,
		remoteAddr:     remoteAddr,
		writer:         w,
		hostIdentifier: HostIdentifier(rand.Uint32()),
		security:       security,

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

// closeWithError closes the connection with the specified error.
func (c *Conn) closeWithError(err error) error {
	c.once.Do(func() {
		c.closeError = err
		close(c.closed)
	})

	return c.closeError
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

	if err = c.writeMessage(MessageTypeHelloRequest, msg); err != nil {
		return err
	}

	return nil
}

func (c *Conn) sendHelloResponse(uniqueNumber UniqueNumber) error {
	msg := &messageHello{
		UniqueNumber: uniqueNumber,
	}

	return c.writeMessage(MessageTypeHelloResponse, msg)
}

func (c *Conn) sendPresentation() error {
	msg := &messagePresentation{
		Certificate: c.security.Certificate,
	}

	return c.writeMessage(MessageTypePresentation, msg)
}

func (c *Conn) sendSessionRequest(sessionNumber SessionNumber) error {
	msg := &messageSessionRequest{
		CipherSuites:   c.security.supportedCipherSuites(),
		EllipticCurves: c.security.supportedEllipticCurves(),
		HostIdentifier: c.hostIdentifier,
		SessionNumber:  sessionNumber,
	}

	if err := msg.computeSignature(); err != nil {
		return fmt.Errorf("failed to forge session request message: %s", err)
	}

	return c.writeMessage(MessageTypeSessionRequest, msg)
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
					debugPrint("(%s <- %s) Received %s request.\n", c.LocalAddr(), c.RemoteAddr(), imsg)

					if err := c.sendHelloResponse(imsg.UniqueNumber); err != nil {
						c.closeWithError(err)
						return
					}

				case MessageTypeHelloResponse:
					debugPrint("(%s <- %s) Received %s response.\n", c.LocalAddr(), c.RemoteAddr(), imsg)

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
					debugPrint("(%s <- %s) Received %s.\n", c.LocalAddr(), c.RemoteAddr(), imsg)

					//TODO: Check if the certificate is acceptable.

					// If we receive a presentation message, store its
					// certificate only if we don't have one already.
					c.security.RemoteCertificate = imsg.Certificate

					if err := c.sendSessionRequest(c.currentSessionNumber); err != nil {
						c.closeWithError(err)
						return
					}
				}
			case *messageSessionRequest:
				debugPrint("(%s <- %s) Received %s.\n", c.LocalAddr(), c.RemoteAddr(), imsg)
			case *messageSession:
				debugPrint("(%s <- %s) Received %s.\n", c.LocalAddr(), c.RemoteAddr(), imsg)
			default:
				debugPrint("(%s <- %s) Received %s.\n", c.LocalAddr(), c.RemoteAddr(), frame.message)
			}
		case <-c.closed:
			return
		}
	}

	//close(c.connected)

	// TODO: Wait for the reply.
}
