package ota

import (
	"sort"

	"github.com/dyl-03/shadow/internal/device"
)

// Plan controls a staged upgrade rollout for a group.
// SelectGroup returns the first `percent` percent of the group's devices in
// stable order.
func SelectGroup(registry *device.Registry, group string, percent int) []string {
	devices := registry.ListByGroup(group)
	ids := make([]string, 0, len(devices))
	for _, d := range devices {
		ids = append(ids, d.ID)
	}
	sort.Strings(ids)
	count := len(ids) * percent / 100
	if count > len(ids) {
		count = len(ids)
	}
	return ids[:count]
}
