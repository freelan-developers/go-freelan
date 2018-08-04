package fscp

import (
	"bytes"
	"crypto/x509"
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
	localAddr  *Addr
	remoteAddr *Addr
	writer     io.Writer
	security   ClientSecurity

	incoming   chan messageFrame
	connected  chan struct{}
	closed     chan struct{}
	closeError error
	once       sync.Once
}

func newConn(localAddr *Addr, remoteAddr *Addr, w io.Writer, security ClientSecurity) *Conn {
	conn := &Conn{
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
		writer:     w,
		security:   security,

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

func (c *Conn) sendHelloRequest() (msg *messageHello, err error) {
	msg = &messageHello{
		UniqueNumber: UniqueNumber(rand.Uint32()),
	}

	if err = c.writeMessage(MessageTypeHelloRequest, msg); err != nil {
		return nil, err
	}

	return msg, nil
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

func (c *Conn) dispatchLoop() {
	helloRequestTimeout := time.Second * 3
	omsgHelloRequest, err := c.sendHelloRequest()

	if err != nil {
		c.closeWithError(err)
		return
	}

	helloRequestTimer := time.NewTimer(helloRequestTimeout)
	defer helloRequestTimer.Stop()

	presentationDone := false
	var remoteCertificate *x509.Certificate

	for {
		select {
		case frame := <-c.incoming:
			switch imsg := frame.message.(type) {
			case *messageHello:
				switch frame.messageType {
				case MessageTypeHelloRequest:
					if err := c.sendHelloResponse(imsg.UniqueNumber); err != nil {
						c.closeWithError(err)
						return
					}

				case MessageTypeHelloResponse:
					if omsgHelloRequest == nil {
						// We have no outstanding hello request. Ignoring.
						continue
					}

					if imsg.UniqueNumber != omsgHelloRequest.UniqueNumber {
						// The received response does not match the outstanding
						// hello request. Ignoring.
						continue
					}

					if !helloRequestTimer.Stop() {
						// The timer already fired: ignoring the reply.
						continue
					}

					omsgHelloRequest = nil

					if err = c.sendPresentation(); err != nil {
						c.closeWithError(err)
						return
					}
				}
			case *messagePresentation:
				switch frame.messageType {
				case MessageTypePresentation:
					if presentationDone {
						if imsg.Certificate.Equal(remoteCertificate) {
							// Previous certificate, matches.
						}
					} else {
						remoteCertificate = imsg.Certificate
						presentationDone = true
						fmt.Println("presentation done", remoteCertificate)
						close(c.connected)
					}
				}
			}
		case <-helloRequestTimer.C:
			if omsgHelloRequest, err = c.sendHelloRequest(); err != nil {
				c.closeWithError(err)
				return
			}

			helloRequestTimer.Reset(helloRequestTimeout)
		case <-c.closed:
			return
		}
	}

	//close(c.connected)

	// TODO: Wait for the reply.
}
