package fscp

import (
	"fmt"
	"net"
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

			return &Client{
				TransportConn: conn,
			}, nil
		default:
			return nil, &net.OpError{Op: "listen", Net: network, Addr: addr, Err: fmt.Errorf("unsupported transport address for FSCP: %#v", addr)}
		}
	default:
		return nil, &net.OpError{Op: "listen", Net: network, Addr: addr, Err: fmt.Errorf("unsupported network: %s", network)}
	}
}

// DefaultClient is the default FSCP client.
var DefaultClient = &Client{}

// Dial dials a new connection.
func Dial(network, addr string) (net.Conn, error) { return DefaultClient.Dial(network, addr) }

// DialFSCP dials a new FSCP connection.
func DialFSCP(network string, laddr *Addr, raddr *Addr) (*Conn, error) {
	return DefaultClient.DialFSCP(network, laddr, raddr)
}
