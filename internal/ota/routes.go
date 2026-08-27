package ota

import (
	"fmt"
	"sync"
)

// Routes resolves each OTA group to its active upgrade version.
type Routes struct {
	mu      sync.RWMutex
	byGroup map[string]string
}

func NewRoutes() *Routes {
	return &Routes{byGroup: make(map[string]string)}
}

func (r *Routes) Set(group, version string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byGroup[group] = version
}

func (r *Routes) Resolve(group string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.byGroup[group]
	if !ok {
		return "", fmt.Errorf("no upgrade route for group %q", group)
	}
	return v, nil
}
