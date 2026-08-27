package session

import (
	"sync"
	"time"

	"github.com/dyl-03/shadow/internal/clock"
	"github.com/dyl-03/shadow/internal/device"
	"github.com/dyl-03/shadow/internal/metrics"
	"github.com/dyl-03/shadow/internal/model"
)

// Session is one live device connection.
type Session struct {
	ID       string
	DeviceID string
	OpenedAt time.Time
}

// Manager tracks live sessions and keeps the device registry consistent.
type Manager struct {
	mu        sync.Mutex
	sessions  map[string]*Session
	registry  *device.Registry
	buffers   *BufferStore
	heartbeat *Heartbeat
	clock     clock.Clock
	metrics   *metrics.Metrics
}

func NewManager(reg *device.Registry, bufs *BufferStore, hb *Heartbeat, clk clock.Clock, m *metrics.Metrics) *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
		registry: reg,
		buffers:  bufs,
		heartbeat: hb,
		clock:    clk,
		metrics:  m,
	}
}

// Open establishes a session for a device, replacing any stale previous
// session so a reconnecting device keeps exactly one live connection.
func (m *Manager) Open(deviceID, sessionID string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	// A reconnecting device must end up with a single live session, so evict
	// every prior session for this device before installing the new one. The
	// new sessionID itself is kept (reinstalled) below. Eviction does not
	// release the per-device offline buffer, which belongs to the new session.
	for id, existing := range m.sessions {
		if existing.DeviceID == deviceID && id != sessionID {
			m.evictLocked(id)
		}
	}
	_, reopened := m.sessions[sessionID]
	sess := &Session{ID: sessionID, DeviceID: deviceID, OpenedAt: m.clock.Now()}
	m.sessions[sessionID] = sess
	if m.heartbeat != nil {
		m.heartbeat.Touch(sessionID)
	}
	m.registry.SetPresence(deviceID, model.PresenceOnline, sessionID)
	if m.metrics != nil && !reopened {
		m.metrics.SessionsOpened.Add(1)
		m.metrics.ActiveSessions.Add(1)
	}
	return sess
}

// Close terminates a session and releases its offline buffers.
func (m *Manager) Close(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.evictLocked(sessionID)
	if s == nil {
		return
	}
	m.registry.SetPresence(s.DeviceID, model.PresenceOffline, "")
	m.buffers.Release(s.DeviceID)
}

// evictLocked removes a session from the in-memory map, drops its heartbeat
// entry, and adjusts metrics. It does not touch the device registry or the
// per-device offline buffer, leaving presence and buffered-message lifecycle
// to the caller. The manager mutex must be held.
func (m *Manager) evictLocked(sessionID string) *Session {
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil
	}
	delete(m.sessions, sessionID)
	if m.heartbeat != nil {
		m.heartbeat.Drop(sessionID)
	}
	if m.metrics != nil {
		m.metrics.SessionsClosed.Add(1)
		m.metrics.ActiveSessions.Add(-1)
	}
	return s
}

func (m *Manager) Get(sessionID string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.sessions[sessionID]
	if s == nil {
		return nil
	}
	cp := *s
	return &cp
}

func (m *Manager) Active() []*Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		cp := *s
		out = append(out, &cp)
	}
	return out
}
