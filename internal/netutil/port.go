package netutil

import (
	"fmt"
	"net"
	"strconv"
)

const maxLocalPortScan = 100

func LocalPortAvailable(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid local port: %d", port)
	}

	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return localPortInUseError(port)
	}
	_ = ln.Close()
	return nil
}

func NextAvailableLocalPort(preferred int) (int, error) {
	if preferred < 1 {
		preferred = 1
	}
	limit := preferred + maxLocalPortScan
	if limit > 65535 {
		limit = 65535
	}
	for port := preferred; port <= limit; port++ {
		if LocalPortAvailable(port) == nil {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available local port found near %d", preferred)
}

func localPortInUseError(port int) error {
	msg := fmt.Sprintf("local port %d is already in use", port)
	if next, err := NextAvailableLocalPort(port + 1); err == nil {
		msg += fmt.Sprintf("; try %d", next)
	}
	return fmt.Errorf("%s", msg)
}
