package fscp

import (
	"fmt"
	"net"
	"time"
)

// Conn is a FSCP connection.
type Conn struct {
	client     *Client
	remoteAddr *Addr
	incoming   chan []byte
}

func newConn(client *Client, remoteAddr *Addr) *Conn {
	conn := &Conn{
		client:     client,
		remoteAddr: remoteAddr,
		incoming:   make(chan []byte, 10),
	}

	go conn.incomingLoop()

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

// Close the connection.
func (c *Conn) Close() error {
	// TODO: Implement.
	return nil
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

func (c *Conn) incomingLoop() {
	for b := range c.incoming {
		// TODO: Do something.
		fmt.Println(b)
	}
}
