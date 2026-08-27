package telemetry

import (
	"github.com/dyl-03/shadow/internal/clock"
	"github.com/dyl-03/shadow/internal/metrics"
	"github.com/dyl-03/shadow/internal/model"
)

// Ingest accepts telemetry samples and forwards them to the store.
type Ingest struct {
	store   *Store
	tracker *SeqTracker
	clock   clock.Clock
	metrics *metrics.Metrics
}

func NewIngest(s *Store, tracker *SeqTracker, clk clock.Clock, m *metrics.Metrics) *Ingest {
	return &Ingest{store: s, tracker: tracker, clock: clk, metrics: m}
}

func (i *Ingest) Accept(deviceID string, seq int64, fields map[string]float64) error {
	if i.metrics != nil {
		i.metrics.TelemetrySeen.Add(1)
	}
	if i.tracker != nil && !i.tracker.Accept(deviceID, seq) {
		if i.metrics != nil {
			i.metrics.TelemetryDropped.Add(1)
		}
		return nil
	}
	return i.store.Append(&model.Telemetry{DeviceID: deviceID, TS: i.clock.Now(), Seq: seq, Fields: fields})
}
