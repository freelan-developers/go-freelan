package fscp

import (
	"io"
	"net"
	"sync"
)

// Client represents a FSCP connection.
type Client struct {
	transportConn net.PacketConn
	backlog       chan *Conn
	connsByAddr   map[string]*Conn
	lock          sync.Mutex
	once          sync.Once
}

// NewClient creates a new client.
func NewClient(conn net.PacketConn) (*Client, error) {
	client := &Client{
		transportConn: conn,
		backlog:       make(chan *Conn, 20),
		connsByAddr:   map[string]*Conn{},
	}

	go client.acceptLoop()

	return client, nil
}

// Addr returns the listener address.
func (c *Client) Addr() net.Addr {
	return &Addr{TransportAddr: c.transportConn.LocalAddr()}
}

// Accept a new connection.
func (c *Client) Accept() (net.Conn, error) {
	if conn, ok := <-c.backlog; ok {
		return conn, nil
	}

	return nil, io.EOF
}

// Close the listener.
func (c *Client) Close() (err error) {
	c.once.Do(func() {
		c.lock.Lock()

		for _, conn := range c.connsByAddr {
			conn.Close()
		}
		c.connsByAddr = nil

		c.lock.Unlock()

		close(c.backlog)
		err = c.transportConn.Close()
	})

	return
}

// Connect connects to the specified host.
func (c *Client) Connect(addr *Addr) (*Conn, error) {
	conn := c.getConn(addr)

	if conn == nil {
		return nil, io.EOF
	}

	return conn, conn.waitSync()
}

// getConn returns the connection associated with the specified remote address.
//
// If no such connection exists, a new one is started.
func (c *Client) getConn(remoteAddr *Addr) *Conn {
	key := remoteAddr.String()

	c.lock.Lock()
	defer c.lock.Unlock()

	if c.connsByAddr == nil {
		return nil
	}

	conn, ok := c.connsByAddr[key]

	if !ok {
		conn = newConn(c, remoteAddr)
		c.connsByAddr[key] = conn
	}

	return conn
}

func (c *Client) acceptLoop() {
	b := make([]byte, 1500)

	for {
		n, addr, err := c.transportConn.ReadFrom(b)

		if err != nil {
			c.Close()
			return
		}

		b := b[:n]
		conn := c.getConn(&Addr{TransportAddr: addr})

		if conn == nil {
			// The client has been closed.
			return
		}

		select {
		case conn.incoming <- b:
		default:
			// If the connection's incoming queue is full, we simply discard the frame.
		}
	}
}

func (c *Client) writeTo(b []byte, addr *Addr) (err error) {
	_, err = c.transportConn.WriteTo(b, addr.TransportAddr)

	return
}
