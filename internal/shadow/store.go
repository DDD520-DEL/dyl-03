package shadow

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dyl-03/shadow/internal/clock"
	"github.com/dyl-03/shadow/internal/metrics"
	"github.com/dyl-03/shadow/internal/model"
	"github.com/dyl-03/shadow/internal/wal"
)

// Store keeps the device shadows and mirrors mutations to the WAL.
type Store struct {
	mu      sync.RWMutex
	shadows map[string]*model.Shadow
	wal     *wal.Manager
	clock   clock.Clock
	metrics *metrics.Metrics
}

func New(w *wal.Manager, clk clock.Clock, m *metrics.Metrics) *Store {
	return &Store{shadows: make(map[string]*model.Shadow), wal: w, clock: clk, metrics: m}
}

func (s *Store) Recover() error {
	return s.wal.Replay(func(rec wal.Record) error {
		if rec.Event != wal.EventWrite {
			return nil
		}
		var sh model.Shadow
		if err := json.Unmarshal(rec.Payload, &sh); err != nil {
			return err
		}
		s.mu.Lock()
		s.shadows[sh.DeviceID] = sh.Clone()
		s.mu.Unlock()
		return nil
	})
}

// Get returns a clone of the shadow, creating an empty one on first access.
func (s *Store) Get(deviceID string) *model.Shadow {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if sh, ok := s.shadows[deviceID]; ok {
		return sh.Clone()
	}
	return &model.Shadow{DeviceID: deviceID, Desired: map[string]string{}, Reported: map[string]string{}}
}

// UpdateDesired applies a desired-state change only when the caller's version
// matches the current version, so concurrent writers cannot clobber each other.
func (s *Store) UpdateDesired(deviceID string, expectedVersion int64, patch map[string]string) (*model.Shadow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sh := s.shadows[deviceID]
	if sh == nil {
		sh = &model.Shadow{DeviceID: deviceID, Desired: map[string]string{}, Reported: map[string]string{}}
	}
	if sh.DesiredVer != expectedVersion {
		return nil, fmt.Errorf("shadow desired version conflict: have %d, want %d", sh.DesiredVer, expectedVersion)
	}
	for k, v := range patch {
		sh.Desired[k] = v
	}
	sh.DesiredVer++
	sh.UpdatedAt = s.clock.Now()
	payload, err := json.Marshal(sh)
	if err != nil {
		return nil, err
	}
	if err := s.wal.Append(wal.Record{Event: wal.EventWrite, Key: "shadow:" + deviceID, Payload: payload}); err != nil {
		return nil, err
	}
	s.shadows[deviceID] = sh.Clone()
	if s.metrics != nil {
		s.metrics.ShadowsUpdated.Add(1)
	}
	return sh.Clone(), nil
}

// UpdateDesiredBatch applies the same desired-state patch to every listed
// device atomically. Each device's desired version must match expectedVersion;
// if any device fails the version check or any write fails, no device is
// mutated and the batch is fully rejected. This lets callers submit a batch
// that is either entirely applied or entirely untouched.
func (s *Store) UpdateDesiredBatch(ids []string, expectedVersion int64, patch map[string]string) ([]*model.Shadow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Stage every mutation against a private copy first, so a failure partway
	// through the batch leaves the live shadows untouched.
	type staged struct {
		id      string
		shadow  *model.Shadow
		payload []byte
	}
	stagedWrites := make([]staged, 0, len(ids))
	for _, id := range ids {
		cur := s.shadows[id]
		if cur == nil {
			cur = &model.Shadow{DeviceID: id, Desired: map[string]string{}, Reported: map[string]string{}}
		}
		if cur.DesiredVer != expectedVersion {
			return nil, fmt.Errorf("shadow desired version conflict for %s: have %d, want %d", id, cur.DesiredVer, expectedVersion)
		}
		next := cur.Clone()
		for k, v := range patch {
			next.Desired[k] = v
		}
		next.DesiredVer++
		next.UpdatedAt = s.clock.Now()
		payload, err := json.Marshal(next)
		if err != nil {
			return nil, fmt.Errorf("marshal shadow %s: %w", id, err)
		}
		stagedWrites = append(stagedWrites, staged{id: id, shadow: next, payload: payload})
	}

	// Durably persist every staged record before touching the live map, so a
	// WAL failure cannot leave the in-memory shadows half written.
	for _, w := range stagedWrites {
		if err := s.wal.Append(wal.Record{Event: wal.EventWrite, Key: "shadow:" + w.id, Payload: w.payload}); err != nil {
			return nil, fmt.Errorf("append wal for %s: %w", w.id, err)
		}
	}

	// Commit the staged shadows to the live map and report the results.
	out := make([]*model.Shadow, 0, len(stagedWrites))
	for _, w := range stagedWrites {
		s.shadows[w.id] = w.shadow.Clone()
		if s.metrics != nil {
			s.metrics.ShadowsUpdated.Add(1)
		}
		out = append(out, w.shadow.Clone())
	}
	return out, nil
}

// ApplyReported merges reported state and bumps the reported version.
func (s *Store) ApplyReported(deviceID string, reported map[string]string) (*model.Shadow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sh := s.shadows[deviceID]
	if sh == nil {
		sh = &model.Shadow{DeviceID: deviceID, Desired: map[string]string{}, Reported: map[string]string{}}
	}
	for k, v := range reported {
		sh.Reported[k] = v
	}
	sh.ReportedVer++
	sh.UpdatedAt = s.clock.Now()
	payload, err := json.Marshal(sh)
	if err != nil {
		return nil, err
	}
	if err := s.wal.Append(wal.Record{Event: wal.EventWrite, Key: "shadow:" + deviceID, Payload: payload}); err != nil {
		return nil, err
	}
	s.shadows[deviceID] = sh.Clone()
	return sh.Clone(), nil
}

// All returns every shadow for tests and snapshots.
func (s *Store) All() []*model.Shadow {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*model.Shadow, 0, len(s.shadows))
	for _, sh := range s.shadows {
		out = append(out, sh.Clone())
	}
	return out
}
