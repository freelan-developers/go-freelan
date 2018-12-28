package fscp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"sync"
)

// Client represents a FSCP connection.
type Client struct {
	transportConn  net.PacketConn
	hostIdentifier HostIdentifier
	security       ClientSecurity
	backlog        chan *Conn
	closed         bool

	lock        sync.Mutex
	connsByAddr map[string]*Conn
}

// NewClient creates a new client.
func NewClient(conn net.PacketConn, security *ClientSecurity) (client *Client, err error) {
	if security == nil {
		security = &ClientSecurity{}
	}

	if err = security.Validate(); err != nil {
		return nil, fmt.Errorf("failed to instanciate a new client: %s", err)
	}

	client = &Client{
		transportConn: conn,
		security:      *security,
		backlog:       make(chan *Conn, 20),
		closed:        false,
		connsByAddr:   map[string]*Conn{},
	}

	if client.hostIdentifier, err = GenerateHostIdentifier(); err != nil {
		return
	}

	go client.dispatchLoop()

	return client, nil
}

// Security gets the client's security.
func (c *Client) Security() ClientSecurity {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.security
}

// SetSecurity sets the security used by the client.
//
// Existing connections are shut-down.
func (c *Client) SetSecurity(security ClientSecurity) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.security = security
	c.closeConns()
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
func (c *Client) Close() error {
	return c.transportConn.Close()
}

// Connect connects to the specified host.
func (c *Client) Connect(ctx context.Context, remoteAddr *Addr) (conn *Conn, err error) {
	var ok bool

	conn, ok = c.addConn(remoteAddr)

	if conn == nil {
		return nil, io.EOF
	}

	if ok {
		// A new connection was added: let's wait for it to be connected or the
		// context to expire, whichever happens first.

		select {
		case <-conn.closed:
			return nil, io.EOF
		case <-conn.connected:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return
}

func (c *Client) dispatchLoop() {
	defer c.finalize()
	defer close(c.backlog)

	b := make([]byte, 1500)

	for {
		n, addr, err := c.transportConn.ReadFrom(b)

		if err != nil {
			return
		}

		data := b[:n]
		remoteAddr := &Addr{TransportAddr: addr}
		conn, ok := c.addConn(remoteAddr)

		// A nil conn indicates that the client is closing, which means we will
		// soon exit from the incoming loop anyway.
		if conn == nil {
			continue
		}

		if ok {
			go func(conn *Conn) {
				select {
				case <-conn.connected:
				case <-conn.closed:
					// If we get there, it means the connection was closed
					// before it completed its handshake.
					return
				}

				select {
				case <-conn.closed:
					// If we get there, it means the connection was closed
					// right after it completed its handshake. This is rare,
					// but if it happens we might as well not add the
					// connection to the backlog.
				case c.backlog <- conn:
					// We added the connection to the backlog and can happily
					// move on.
				default:
					// If the backlog is full, we shut down the connection.
					conn.Close()
				}
			}(conn)
		}

		var reader lenReader = bytes.NewReader(data)

		if messageType, message, err := readMessage(reader); err == nil {
			select {
			case conn.incoming <- messageFrame{messageType, message}:
			default:
				// If the connection's incoming queue is full, we simply discard
				// the frame.
			}
		} else {
			debugPrintf("failed to read message: %s\n", err)
		}
	}
}

func (c *Client) finalize() {
	c.lock.Lock()
	defer c.lock.Unlock()

	// After that point (and the lock is released), addConn() can't add new
	// connections which means Connect() can't either.
	c.closed = true
	c.closeConns()
}

func (c *Client) addConn(remoteAddr *Addr) (conn *Conn, ok bool) {
	key := remoteAddr.String()

	c.lock.Lock()
	defer c.lock.Unlock()

	conn, ok = c.connsByAddr[key]

	if !ok {
		if c.closed {
			return nil, false
		}

		// This is a new peer so we start a new connection.
		writer := &clientWriter{c, remoteAddr.TransportAddr}
		conn = newConn(&Addr{TransportAddr: c.Addr()}, remoteAddr, writer, c.hostIdentifier, c.security)

		c.connsByAddr[key] = conn

		// Whatever happens, when the connection is closed, we unregister it.
		go func() {
			<-conn.closed
			c.removeConn(conn)
		}()
	}

	ok = !ok

	return
}

func (c *Client) removeConn(conn *Conn) {
	key := conn.RemoteAddr().String()

	c.lock.Lock()
	delete(c.connsByAddr, key)
	c.lock.Unlock()
}

// closeConns closes all the connections.
//
// The mutex *MUST* be held before calling this method.
func (c *Client) closeConns() {
	// Close all the remaining connections.
	for _, conn := range c.connsByAddr {
		conn.Close()
	}

	// Clear the map.
	c.connsByAddr = map[string]*Conn{}
}

type clientWriter struct {
	client     *Client
	remoteAddr net.Addr
}

func (w *clientWriter) Write(b []byte) (n int, err error) {
	return w.client.transportConn.WriteTo(b, w.remoteAddr)
}
