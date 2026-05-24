package services

import "sync"

var (
	registryMu sync.RWMutex
	registry   []Provider
)

func Register(p Provider) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = append(registry, p)
}

func All() []Provider {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Provider, len(registry))
	copy(out, registry)
	return out
}
