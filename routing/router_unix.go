package routing

import "net"

type routerImpl struct{}

// NewRouter instanciates a new router.
func NewRouter() Router {
	return &routerImpl{}
}

func (r *routerImpl) AddRoute(network *net.IPNet, gateway net.IP) (bool, error) {
	return false, nil
}

func (r *routerImpl) DeleteRoute(network *net.IPNet, gateway net.IP) (bool, error) {
	return false, nil
}
