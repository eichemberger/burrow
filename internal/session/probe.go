package session

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

const (
	defaultProbeInterval = 200 * time.Millisecond
	defaultProbeTimeout  = 15 * time.Second
)

func WaitForLocalPort(port int, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = defaultProbeTimeout
	}
	deadline := time.Now().Add(timeout)
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(defaultProbeInterval)
	}
	return fmt.Errorf("local port %d did not become available within %s", port, timeout)
}

func PortListening(port int) bool {
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
