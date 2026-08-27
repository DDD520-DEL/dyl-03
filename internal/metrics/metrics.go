package metrics

import "sync/atomic"

// Metrics aggregates counters for operations tooling.
type Metrics struct {
	CommandsSent   atomic.Int64
	CommandsAcked  atomic.Int64
	CommandsRetry  atomic.Int64
	SessionsOpened atomic.Int64
	SessionsClosed atomic.Int64
	ShadowsUpdated atomic.Int64
	TelemetrySeen  atomic.Int64
	TelemetryDropped atomic.Int64
	PurgedSamples  atomic.Int64
	ActiveSessions atomic.Int64
	BufferEntries  atomic.Int64
}

func New() *Metrics { return &Metrics{} }
