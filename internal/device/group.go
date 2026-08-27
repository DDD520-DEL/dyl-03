package device

import "sync"

// GroupIndex maps device IDs to their OTA group for command routing.
type GroupIndex struct {
	registry *Registry
	index    map[string]string
	mu       sync.RWMutex
}

func NewGroupIndex(r *Registry) *GroupIndex {
	idx := &GroupIndex{registry: r, index: make(map[string]string)}
	for _, d := range r.All() {
		idx.index[d.ID] = d.Group
	}
	return idx
}

// Refresh rebuilds the index from the registry so moves take effect.
func (g *GroupIndex) Refresh() {
	g.mu.Lock()
	defer g.mu.Unlock()
	next := make(map[string]string)
	for _, d := range g.registry.All() {
		next[d.ID] = d.Group
	}
	g.index = next
}

func (g *GroupIndex) GroupOf(id string) string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.index[id]
}
