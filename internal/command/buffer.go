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
}

func NewBuffer() *Buffer {
	return &Buffer{entries: make(map[string][]*model.Command)}
}

func (b *Buffer) Enqueue(cmd *model.Command) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries[cmd.DeviceID] = append(b.entries[cmd.DeviceID], cmd.Clone())
}

// Deliver returns the queued commands for a device in sequence order and
// removes them from the buffer so each command is delivered exactly once.
func (b *Buffer) Deliver(deviceID string) []*model.Command {
	b.mu.Lock()
	defer b.mu.Unlock()
	entries := b.entries[deviceID]
	out := make([]*model.Command, 0, len(entries))
	for _, c := range entries {
		out = append(out, c.Clone())
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Seq < out[j].Seq
	})
	// Remove the delivered entries so a subsequent call does not return the
	// same commands again — each buffered command must be delivered exactly
	// once after a reconnect.
	delete(b.entries, deviceID)
	return out
}

func (b *Buffer) Pending(deviceID string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries[deviceID])
}
