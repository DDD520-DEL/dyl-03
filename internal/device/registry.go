package device

import (
	"fmt"
	"sync"

	"github.com/dyl-03/shadow/internal/model"
)

// Registry tracks device records and their current presence.
type Registry struct {
	mu      sync.RWMutex
	devices map[string]*model.Device
}

func NewRegistry() *Registry {
	return &Registry{devices: make(map[string]*model.Device)}
}

func (r *Registry) Register(d *model.Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.devices[d.ID]; ok {
		return fmt.Errorf("device %s already registered", d.ID)
	}
	r.devices[d.ID] = d.Clone()
	return nil
}

func (r *Registry) Get(id string) *model.Device {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.devices[id].Clone()
}

func (r *Registry) SetPresence(id string, presence model.Presence, sessionID string) *model.Device {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.devices[id]
	if !ok {
		return nil
	}
	d.Presence = presence
	d.SessionID = sessionID
	r.devices[id] = d
	return d.Clone()
}

func (r *Registry) MoveGroup(id, group string) (*model.Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.devices[id]
	if !ok {
		return nil, fmt.Errorf("device %s not found", id)
	}
	d.Group = group
	d.Version++
	r.devices[id] = d
	return d.Clone(), nil
}

func (r *Registry) Online() []*model.Device {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*model.Device, 0, len(r.devices))
	for _, d := range r.devices {
		if d.Presence == model.PresenceOnline {
			out = append(out, d.Clone())
		}
	}
	return out
}

func (r *Registry) All() []*model.Device {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*model.Device, 0, len(r.devices))
	for _, d := range r.devices {
		out = append(out, d.Clone())
	}
	return out
}

// ListByGroup returns the devices of one OTA group.
func (r *Registry) ListByGroup(group string) []*model.Device {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*model.Device
	for _, d := range r.devices {
		if d.Group == group {
			out = append(out, d.Clone())
		}
	}
	return out
}
