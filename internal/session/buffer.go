package session

import (
	"sync"

	"github.com/dyl-03/shadow/internal/metrics"
)

// BufferStore holds per-device outbound message buffers while offline.
type BufferStore struct {
	mu      sync.Mutex
	buffers map[string][]string
	metrics *metrics.Metrics
}

func NewBufferStore(m *metrics.Metrics) *BufferStore {
	return &BufferStore{buffers: make(map[string][]string), metrics: m}
}

func (b *BufferStore) Enqueue(deviceID, message string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buffers[deviceID] = append(b.buffers[deviceID], message)
	if b.metrics != nil {
		b.metrics.BufferEntries.Add(1)
	}
}

func (b *BufferStore) Pending(deviceID string) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, len(b.buffers[deviceID]))
	copy(out, b.buffers[deviceID])
	return out
}

// Release removes the per-device buffer when the session closes.
func (b *BufferStore) Release(deviceID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if n := len(b.buffers[deviceID]); n > 0 && b.metrics != nil {
		b.metrics.BufferEntries.Add(-int64(n))
	}
	delete(b.buffers, deviceID)
}

func (b *BufferStore) Count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	total := 0
	for _, buf := range b.buffers {
		total += len(buf)
	}
	return total
}
