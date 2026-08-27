package clock

import "time"

// Clock abstracts time for deterministic tests.
type Clock interface {
	Now() time.Time
}

// Wall returns the real clock.
func Wall() Clock { return wallClock{} }

type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now() }

// Manual is a controllable clock.
type Manual struct {
	now time.Time
}

func NewManual(t time.Time) *Manual { return &Manual{now: t} }
func (m *Manual) Now() time.Time   { return m.now }
func (m *Manual) Advance(d time.Duration) {
	m.now = m.now.Add(d)
}
func (m *Manual) Set(t time.Time) { m.now = t }
