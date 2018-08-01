package fscp

import (
	"fmt"
	"net"
)

// Client represents a FSCP connection.
type Client struct {
	TransportConn net.PacketConn
}

// Addr returns the listener address.
func (c *Client) Addr() net.Addr {
	return &Addr{TransportAddr: c.TransportConn.LocalAddr()}
}

// Accept a new connection.
func (c *Client) Accept() (net.Conn, error) {
	// TODO: Create an ingoing connection.
	return nil, nil
}

// Close the listener.
func (c *Client) Close() error {
	// TODO: Close the listener.
	return nil
}

// Dial dials a new connection.
func (c *Client) Dial(network, addr string) (net.Conn, error) {
	switch network {
	case Network:
		addr, err := ResolveFSCPAddr(network, addr)

		if err != nil {
			return nil, &net.OpError{Op: "dial", Net: network, Err: err}
		}

		return c.DialFSCP(network, nil, addr)
	default:
		return net.Dial(network, addr)
	}
}

// DialFSCP dials a new FSCP connection.
func (c *Client) DialFSCP(network string, laddr *Addr, raddr *Addr) (*Conn, error) {
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
