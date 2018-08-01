package fscp

import (
	"net"
	"time"
)

// Conn is a FSCP connection.
type Conn struct {
	transportConn net.PacketConn
	remoteAddr    *Addr
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
	return &Addr{TransportAddr: c.transportConn.LocalAddr()}
}

// RemoteAddr returns the remote address of the connection.
func (c *Conn) RemoteAddr() net.Addr { return c.remoteAddr }

// SetDeadline sets the deadline on the connection.
func (c *Conn) SetDeadline(t time.Time) error {
	// TODO: Implement.
	return c.transportConn.SetDeadline(t)
}

// SetReadDeadline sets the deadline on the connection.
func (c *Conn) SetReadDeadline(t time.Time) error {
	// TODO: Implement.
	return c.transportConn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline on the connection.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	// TODO: Implement.
	return c.transportConn.SetWriteDeadline(t)
}
