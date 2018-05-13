package routing

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strings"
)

type routerImpl struct {
}

// NewRouter instanciates a new router.
func NewRouter() Router {
	return &routerImpl{}
}

func (r *routerImpl) AddRoute(network *net.IPNet, gateway net.IP) (bool, error) {
	args := []string{
		"-n",
		"add",
		"-net",
		network.String(),
		gateway.String(),
	}
	cmd := exec.Command("route", args...)
	b, err := cmd.CombinedOutput()

	if err != nil {
		return false, fmt.Errorf("adding route %s -> %s: %s", network, gateway, err)
	}

	scanner := bufio.NewScanner(bytes.NewBuffer(b))

	if !scanner.Scan() {
		return false, fmt.Errorf("adding route %s -> %s: unexpected output (%s)", network, gateway, string(b))
	}

	if strings.HasPrefix(scanner.Text(), "route: ") {
		// This is as reliable as it can be...
		if strings.HasSuffix(scanner.Text(), "File exists") {
			return false, nil
		}

		return false, fmt.Errorf("adding route %s -> %s: %s", network, gateway, scanner.Text())
	}

	return true, nil
}

func (r *routerImpl) DeleteRoute(network *net.IPNet, gateway net.IP) (bool, error) {
	args := []string{
		"-n",
		"delete",
		"-net",
		network.String(),
		gateway.String(),
	}
	cmd := exec.Command("route", args...)
	b, err := cmd.CombinedOutput()

	if err != nil {
		return false, fmt.Errorf("deleting route %s -> %s: %s", network, gateway, err)
	}

	scanner := bufio.NewScanner(bytes.NewBuffer(b))

	if !scanner.Scan() {
		return false, fmt.Errorf("deleting route %s -> %s: unexpected output (%s)", network, gateway, string(b))
	}

	if strings.HasPrefix(scanner.Text(), "route: ") {
		// This is as reliable as it can be...
		if strings.HasSuffix(scanner.Text(), "not in table") {
			return false, nil
		}

		return false, fmt.Errorf("deleting route %s -> %s: %s", network, gateway, scanner.Text())
	}

	return true, nil
}
