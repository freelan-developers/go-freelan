package fscp

import (
	"context"
	"fmt"
	"net"
	"time"
)

const (
	// Network is the default network.
	Network = "fscp"
)

var (
	// DefaultAddr is the default listening address.
	DefaultAddr = &Addr{
		TransportAddr: &net.UDPAddr{
			Port: 5000,
		},
	}
)

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
func ListenFSCP(network string, addr *Addr) (*Client, error) {
	switch network {
	case Network:
		switch taddr := addr.TransportAddr.(type) {
		case *net.UDPAddr:
			conn, err := net.ListenUDP("udp", taddr)

			if err != nil {
				return nil, err
			}

			return NewClient(conn)
		default:
			return nil, &net.OpError{Op: "listen", Net: network, Addr: addr, Err: fmt.Errorf("unsupported transport address for FSCP: %#v", addr)}
		}
	default:
		return nil, &net.OpError{Op: "listen", Net: network, Addr: addr, Err: fmt.Errorf("unsupported network: %s", network)}
	}
}

// A Dialer offers connection dialing primitives.
type Dialer struct {
	Timeout time.Duration
}

// DefaultTimeout is the default time to wait for dialing connections.
const DefaultTimeout = time.Second * 5

// DefaultDialer is the default dialer backing the free-form dialing functions.
var DefaultDialer = &Dialer{}

func (d Dialer) getTimeout() time.Duration {
	if d.Timeout < 0 {
		return DefaultTimeout
	}

	return d.Timeout
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
		if laddr == nil {
			laddr = DefaultAddr
		}

		client, err := ListenFSCP(network, laddr)

		if err != nil {
			return nil, err
		}

		ctx, cancel := context.WithTimeout(context.Background(), d.getTimeout())
		defer cancel()

		return client.Connect(ctx, raddr)
	default:
		return nil, &net.OpError{Op: "dial", Net: network, Addr: raddr, Err: fmt.Errorf("unsupported network: %s", network)}
	}
}

// Dial dials a new FSCP connection using the default Dialer.
func Dial(network, addr string) (net.Conn, error) {
	return DefaultDialer.Dial(network, addr)
}

// DialFSCP dials a new FSCP connection.
func DialFSCP(network string, laddr *Addr, raddr *Addr) (*Conn, error) {
	return DefaultDialer.DialFSCP(network, laddr, raddr)
}
