package command

import (
	"sort"
	"sync"

	"github.com/dyl-03/shadow/internal/model"
)

// Buffer holds commands queued for offline devices.
type Buffer struct {
	mu      sync.Mutex
	entries map[string][]*model.Command
	// lastDelivered tracks the highest sequence number handed to a device so
	// a command that was delivered once (possibly out of order, or re-buffered
	// by a retry) can never overwrite a newer command already applied.
	lastDelivered map[string]int64
}

func NewBuffer() *Buffer {
	return &Buffer{
		entries:       make(map[string][]*model.Command),
		lastDelivered: make(map[string]int64),
	}
}

func (b *Buffer) Enqueue(cmd *model.Command) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries[cmd.DeviceID] = append(b.entries[cmd.DeviceID], cmd.Clone())
}

// Deliver returns the queued commands for a device in sequence order and
// removes them from the buffer so each command is delivered exactly once.
// Commands are sorted by Seq so they reach the device in strictly ascending
// order, and any command whose Seq is not newer than the last one delivered to
// the device is dropped: an older command must never overwrite a newer one.
func (b *Buffer) Deliver(deviceID string) []*model.Command {
	b.mu.Lock()
	defer b.mu.Unlock()
	entries := b.entries[deviceID]
	// Operate on a copy so the sort does not mutate the stored slice (it is
	// about to be deleted anyway, but the in-place sort could race with a
	// concurrent Enqueue reading the same backing array).
	pending := make([]*model.Command, len(entries))
	copy(pending, entries)
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Seq < pending[j].Seq
	})
	last := b.lastDelivered[deviceID]
	out := make([]*model.Command, 0, len(entries))
	for _, c := range pending {
		if c.Seq <= last {
			// Stale relative to what the device has already been handed; skip it
			// so an old command cannot clobber a newer applied state.
			continue
		}
		out = append(out, c.Clone())
		last = c.Seq
	}
	if len(out) > 0 {
		b.lastDelivered[deviceID] = last
	}
	delete(b.entries, deviceID)
	return out
}

func (b *Buffer) Pending(deviceID string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries[deviceID])
}
