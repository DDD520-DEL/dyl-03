package ota

import "github.com/dyl-03/shadow/internal/device"

// GroupPlanner moves devices between groups and keeps the routing index in
// sync so upgrades follow the latest group membership.
type GroupPlanner struct {
	registry *device.Registry
	index    *device.GroupIndex
}

func NewGroupPlanner(reg *device.Registry, idx *device.GroupIndex) *GroupPlanner {
	return &GroupPlanner{registry: reg, index: idx}
}

func (g *GroupPlanner) Move(deviceID, group string) error {
	if _, err := g.registry.MoveGroup(deviceID, group); err != nil {
		return err
	}
	// Rebuild the routing index so commands resolve against the new group
	// immediately. Without this the dispatcher keeps using the stale group
	// mapping and devices still receive their old group's upgrade route.
	g.index.Refresh()
	return nil
}
