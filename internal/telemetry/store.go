package telemetry

import (
	"sync"

	"github.com/dyl-03/shadow/internal/clock"
	"github.com/dyl-03/shadow/internal/metrics"
	"github.com/dyl-03/shadow/internal/model"
)

// Store keeps a bounded per-device sample window.
type Store struct {
	mu      sync.Mutex
	samples map[string][]*model.Telemetry
	clock   clock.Clock
	metrics *metrics.Metrics
	limit   int
}

func NewStore(clk clock.Clock, m *metrics.Metrics, limit int) *Store {
	return &Store{samples: make(map[string][]*model.Telemetry), clock: clk, metrics: m, limit: limit}
}

func (s *Store) Append(t *model.Telemetry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	hist := s.samples[t.DeviceID]
	hist = append(hist, t.Clone())
	if len(hist) > s.limit {
		hist = hist[len(hist)-s.limit:]
	}
	s.samples[t.DeviceID] = hist
	return nil
}

func (s *Store) Recent(deviceID string) []*model.Telemetry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*model.Telemetry, len(s.samples[deviceID]))
	for i, t := range s.samples[deviceID] {
		out[i] = t.Clone()
	}
	return out
}

func (s *Store) Remove(deviceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.samples, deviceID)
}

// Replace overwrites the sample list of one device.
func (s *Store) Replace(deviceID string, samples []*model.Telemetry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*model.Telemetry, 0, len(samples))
	for _, t := range samples {
		out = append(out, t.Clone())
	}
	s.samples[deviceID] = out
}
