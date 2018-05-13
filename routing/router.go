package routing

import "net"

// A Router provides facilities to manipulate the operating-system's routing
// table.
type Router interface {
	// AddRoute adds a network route.
	//
	// The first returned value indicates whether the route was added.
	AddRoute(network *net.IPNet, gateway net.IP) (bool, error)

	// DeleteRoute deletes a network route.
	//
	// The first returned value indicates whether the route was added.
	DeleteRoute(network *net.IPNet, gateway net.IP) (bool, error)
}
