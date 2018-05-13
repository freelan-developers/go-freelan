package tuntap

import "net"

func (a *adapterImpl) IPv4() (*net.IPNet, error) {
	addrs, err := a.Interface().Addrs()

	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		if ip, ipnet, err := net.ParseCIDR(addr.String()); err == nil {
			if ipv4 := ip.To4(); ipv4 != nil {
				return &net.IPNet{
					IP:   ipv4,
					Mask: ipnet.Mask,
				}, nil
			}
		}
	}

	return nil, nil
}

func (a *adapterImpl) IPv6() (*net.IPNet, error) {
	addrs, err := a.Interface().Addrs()

	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		if ip, ipnet, err := net.ParseCIDR(addr.String()); err == nil {
			if ipv6 := ip.To16(); ipv6 != nil {
				return &net.IPNet{
					IP:   ipv6,
					Mask: ipnet.Mask,
				}, nil
			}
		}
	}

	return nil, nil
}
