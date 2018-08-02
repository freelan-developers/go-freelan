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

// Conn is a FSCP connection.
type Conn struct {
	client     *Client
	remoteAddr *Addr
	incoming   chan []byte
	synced     chan struct{}
	closed     chan struct{}
	closeError error
	once       sync.Once
}

func newConn(client *Client, remoteAddr *Addr) *Conn {
	conn := &Conn{
		client:     client,
		remoteAddr: remoteAddr,
		incoming:   make(chan []byte, 10),
		synced:     make(chan struct{}),
		closed:     make(chan struct{}),
	}

	go conn.handshake()

	return conn
}

func (c *Conn) Read(b []byte) (n int, err error) {
	// TODO: Implement.
	return 0, nil
}

func (c *Conn) Write(b []byte) (n int, err error) {
	// TODO: Implement.
	return len(b), c.client.writeTo(b, c.remoteAddr)
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
func (c *Conn) LocalAddr() net.Addr {
	return &Addr{TransportAddr: c.client.Addr()}
}

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

func (c *Conn) writeMessage(buf *bytes.Buffer, messageType MessageType, message serializable) (err error) {
	writeMessage(buf, messageType, message)
	err = c.client.writeTo(buf.Bytes(), c.remoteAddr)
	buf.Reset()

	return
}

func (c *Conn) handshake() {
	uniqueNumber := UniqueNumber(rand.Uint32())
	msgHello := &messageHello{
		UniqueNumber: uniqueNumber,
	}

	buf := &bytes.Buffer{}

	if err := c.writeMessage(buf, MessageTypeHelloRequest, msgHello); err != nil {
		c.closeWithError(err)
		return
	}

	// TODO: Wait for the reply.
}

func (c *Conn) incomingLoop() {
	for b := range c.incoming {
		// TODO: Do something.
		fmt.Println(b)
	}
}

func (c *Conn) waitSync() error {
	select {
	case <-c.synced:
		return nil
	case <-c.closed:
		return c.closeError
	}
}
