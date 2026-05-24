package netutil

import "testing"

func TestDefaultPrivateNetworks(t *testing.T) {
	nets, err := DefaultPrivateNetworks()
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]bool{
		"10.0.1.5":       true,
		"10.255.255.255": true,
		"172.16.0.1":     true,
		"172.31.255.255": true,
		"192.168.1.1":    true,
		"8.8.8.8":        false,
		"":               false,
		"not-an-ip":      false,
	}
	for ip, want := range tests {
		if got := nets.ContainsString(ip); got != want {
			t.Fatalf("ContainsString(%q) = %v, want %v", ip, got, want)
		}
	}
}

func TestNewNetworkSetCustomCIDRs(t *testing.T) {
	nets, err := NewNetworkSet([]string{"10.0.0.0/16", "192.168.0.0/24"})
	if err != nil {
		t.Fatal(err)
	}
	if !nets.ContainsString("10.0.5.1") {
		t.Fatal("expected 10.0.5.1 in custom set")
	}
	if nets.ContainsString("10.1.0.1") {
		t.Fatal("expected 10.1.0.1 outside 10.0.0.0/16")
	}
	if nets.Label() == "RFC 1918 private ranges" {
		t.Fatal("expected custom label for overridden CIDRs")
	}
}

func TestNewNetworkSetRejectsInvalidCIDR(t *testing.T) {
	if _, err := NewNetworkSet([]string{"not-a-cidr"}); err == nil {
		t.Fatal("expected invalid CIDR error")
	}
}

func TestCIDRContainsIP(t *testing.T) {
	ok, err := CIDRContainsIP("10.0.0.0/16", ParseIP("10.0.5.1"))
	if err != nil || !ok {
		t.Fatalf("expected 10.0.5.1 in 10.0.0.0/16")
	}
	ok, err = CIDRContainsIP("10.1.0.0/16", ParseIP("10.0.5.1"))
	if err != nil || ok {
		t.Fatalf("expected 10.0.5.1 not in 10.1.0.0/16")
	}
}
