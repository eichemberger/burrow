package netutil

import (
	"net"
	"strconv"
	"testing"
)

func TestLocalPortAvailable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}

	if err := LocalPortAvailable(port); err == nil {
		t.Fatalf("expected port %d to be unavailable", port)
	}
	if err := LocalPortAvailable(port); err.Error() == "" {
		t.Fatal("expected non-empty in-use error")
	}
}

func TestNextAvailableLocalPortFindsFreePort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	busyPort, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}

	next, err := NextAvailableLocalPort(busyPort)
	if err != nil {
		t.Fatal(err)
	}
	if next == busyPort {
		t.Fatalf("expected a different port than busy port %d", busyPort)
	}
	if err := LocalPortAvailable(next); err != nil {
		t.Fatalf("expected port %d to be available: %v", next, err)
	}
}
