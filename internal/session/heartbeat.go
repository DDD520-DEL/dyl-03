package session

import (
	"time"

	"github.com/dyl-03/shadow/internal/clock"
)

// Heartbeat tracks the last seen time of each session so stale sessions can
// be closed after a timeout.
type Heartbeat struct {
	last map[string]time.Time
	clock clock.Clock
}

func NewHeartbeat(clk clock.Clock) *Heartbeat {
	return &Heartbeat{last: make(map[string]time.Time), clock: clk}
}

func (h *Heartbeat) Touch(sessionID string) {
	h.last[sessionID] = h.clock.Now()
}

func (h *Heartbeat) Stale(sessionID string, timeout time.Duration) bool {
	last, ok := h.last[sessionID]
	if !ok {
		return true
	}
	return h.clock.Now().Sub(last) > timeout
}

func (h *Heartbeat) Drop(sessionID string) {
	delete(h.last, sessionID)
}
