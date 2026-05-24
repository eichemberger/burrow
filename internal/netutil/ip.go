package netutil

import (
	"fmt"
	"net"
	"strings"
)

var defaultPrivateCIDRs = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
}

type NetworkSet struct {
	cidrs []string
	nets  []*net.IPNet
}

func DefaultPrivateNetworks() (*NetworkSet, error) {
	return NewNetworkSet(defaultPrivateCIDRs)
}

func NewNetworkSet(cidrs []string) (*NetworkSet, error) {
	if len(cidrs) == 0 {
		return DefaultPrivateNetworks()
	}

	nets := make([]*net.IPNet, 0, len(cidrs))
	unique := make([]string, 0, len(cidrs))
	seen := make(map[string]struct{}, len(cidrs))
	for _, cidr := range cidrs {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		if _, ok := seen[cidr]; ok {
			continue
		}
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
		}
		seen[cidr] = struct{}{}
		unique = append(unique, cidr)
		nets = append(nets, network)
	}
	if len(nets) == 0 {
		return DefaultPrivateNetworks()
	}

	return &NetworkSet{cidrs: unique, nets: nets}, nil
}

func (n *NetworkSet) Contains(ip net.IP) bool {
	if n == nil || ip == nil {
		return false
	}
	ip = ip.To4()
	if ip == nil {
		return false
	}
	for _, network := range n.nets {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func (n *NetworkSet) ContainsString(s string) bool {
	return n.Contains(ParseIP(s))
}

func (n *NetworkSet) Label() string {
	if n == nil || len(n.cidrs) == 0 {
		return "RFC 1918 private ranges"
	}
	if sameCIDRs(n.cidrs, defaultPrivateCIDRs) {
		return "RFC 1918 private ranges"
	}
	return strings.Join(n.cidrs, ", ")
}

func sameCIDRs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]struct{}, len(a))
	for _, cidr := range a {
		seen[cidr] = struct{}{}
	}
	for _, cidr := range b {
		if _, ok := seen[cidr]; !ok {
			return false
		}
	}
	return true
}

func ParseIP(s string) net.IP {
	return net.ParseIP(s)
}

func CIDRContainsIP(cidr string, ip net.IP) (bool, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, err
	}
	return network.Contains(ip), nil
}
