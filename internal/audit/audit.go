package audit

import (
	"sync"
	"time"

	"github.com/dyl-03/shadow/internal/clock"
)

// Entry is one recorded event in the audit trail.
type Entry struct {
	DeviceID string
	Action   string
	Detail   string
	At       time.Time
}

// Log keeps a bounded per-device event history.
type Log struct {
	mu      sync.Mutex
	entries map[string][]Entry
	limit   int
	clock   clock.Clock
}

func New(limit int, clk clock.Clock) *Log {
	return &Log{entries: make(map[string][]Entry), limit: limit, clock: clk}
}

func (l *Log) Record(deviceID, action, detail string) {
	if deviceID == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	hist := l.entries[deviceID]
	hist = append(hist, Entry{DeviceID: deviceID, Action: action, Detail: detail, At: l.clock.Now()})
	if len(hist) > l.limit {
		hist = hist[len(hist)-l.limit:]
	}
	l.entries[deviceID] = hist
}

func (l *Log) History(deviceID string) []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Entry, len(l.entries[deviceID]))
	copy(out, l.entries[deviceID])
	return out
}
