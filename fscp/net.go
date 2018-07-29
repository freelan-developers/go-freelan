package fscp

import (
	"fmt"
	"net"
	"time"
)

// Network is the default network.
const Network = "fscp"

// Addr is a FSCP address.
type Addr struct {
	TransportAddr net.Addr
}

// Network returns the network associated to the address.
func (a *Addr) Network() string { return Network }

func (a *Addr) String() string { return a.TransportAddr.String() }

// ResolveFSCPAddr parses a FSCP address.
func ResolveFSCPAddr(network, address string) (*Addr, error) {
	switch network {
	case Network:
		udpAddr, err := net.ResolveUDPAddr("udp", address)

		if err != nil {
			return nil, fmt.Errorf("parsing FSCP address: %s", err)
		}

		return &Addr{
			TransportAddr: udpAddr,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported network: %s", network)
	}
}

// Listener represents a FSCP listener.
type Listener struct {
	TransportConn net.PacketConn
}

// Listen listens to a FSCP address.
func Listen(network string, addr string) (net.Listener, error) {
	switch network {
	case Network:
		addr, err := ResolveFSCPAddr(network, addr)

		if err != nil {
			return nil, &net.OpError{Op: "listen", Net: network, Err: err}
		}

		return ListenFSCP(network, addr)
	default:
		return net.Listen(network, addr)
	}
}

// ListenFSCP listens to a FSCP address.
func ListenFSCP(network string, addr *Addr) (*Listener, error) {
	switch network {
	case Network:
		switch taddr := addr.TransportAddr.(type) {
		case *net.UDPAddr:
			conn, err := net.ListenUDP("udp", taddr)

			if err != nil {
				return nil, err
			}

			return &Listener{
				TransportConn: conn,
			}, nil
		default:
			return nil, &net.OpError{Op: "listen", Net: network, Addr: addr, Err: fmt.Errorf("unsupported transport address for FSCP: %#v", addr)}
		}
	default:
		return nil, &net.OpError{Op: "listen", Net: network, Addr: addr, Err: fmt.Errorf("unsupported network: %s", network)}
	}
}

// Addr returns the listener address.
func (l *Listener) Addr() net.Addr {
	return &Addr{TransportAddr: l.TransportConn.LocalAddr()}
}

// Accept a new connection.
func (l *Listener) Accept() (net.Conn, error) {
	// TODO: Create an ingoing connection.
	return nil, nil
}

// Close the listener.
func (l *Listener) Close() error {
	// TODO: Close the listener.
	return nil
}

// Dialer is a FSCP dialer.
//
// A dialer will try to multiplex connections as much as possible.
type Dialer struct {
}

// Dial dials a new connection.
func (d *Dialer) Dial(network, addr string) (net.Conn, error) {
	switch network {
	case Network:
		addr, err := ResolveFSCPAddr(network, addr)

		if err != nil {
			return nil, &net.OpError{Op: "dial", Net: network, Err: err}
		}

		return d.DialFSCP(network, nil, addr)
	default:
		return net.Dial(network, addr)
	}
}

// DialFSCP dials a new FSCP connection.
func (d *Dialer) DialFSCP(network string, laddr *Addr, raddr *Addr) (*Conn, error) {
	switch network {
	case Network:
		switch rtaddr := raddr.TransportAddr.(type) {
		case *net.UDPAddr:
			ltaddr, ok := laddr.TransportAddr.(*net.UDPAddr)

			if !ok {
				return nil, &net.OpError{Op: "dial", Net: network, Addr: ltaddr, Err: fmt.Errorf("unsupported transport address for FSCP: %#v", ltaddr)}
			}

			conn, err := net.DialUDP("udp", ltaddr, rtaddr)

			if err != nil {
				return nil, err
			}

			return &Conn{
				transportConn: conn,
				remoteAddr:    raddr,
			}, nil
		default:
			return nil, &net.OpError{Op: "dial", Net: network, Addr: raddr, Err: fmt.Errorf("unsupported transport address for FSCP: %#v", raddr)}
		}
	default:
		return nil, &net.OpError{Op: "dial", Net: network, Addr: raddr, Err: fmt.Errorf("unsupported network: %s", network)}
	}
}

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
