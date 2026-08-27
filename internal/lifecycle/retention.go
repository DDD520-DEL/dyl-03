package lifecycle

import (
	"time"

	"github.com/dyl-03/shadow/internal/clock"
	"github.com/dyl-03/shadow/internal/metrics"
	"github.com/dyl-03/shadow/internal/model"
	"github.com/dyl-03/shadow/internal/telemetry"
)

// Retention purges telemetry samples that fall outside the window.
type Retention struct {
	store   *telemetry.Store
	clock   clock.Clock
	window  time.Duration
	metrics *metrics.Metrics
}

func NewRetention(s *telemetry.Store, clk clock.Clock, window time.Duration, m *metrics.Metrics) *Retention {
	return &Retention{store: s, clock: clk, window: window, metrics: m}
}

// Expired reports whether a sample is completely older than the window.
func (r *Retention) Expired(t *model.Telemetry, now time.Time) bool {
	if r.window <= 0 {
		return false
	}
	return t.TS.Before(now.Add(-r.window))
}

// Purge removes old samples per device.
func (r *Retention) Purge(devices []string) error {
	now := r.clock.Now()
	for _, id := range devices {
		keep := r.store.Recent(id)
		var kept []*model.Telemetry
		for _, t := range keep {
			if !r.Expired(t, now) {
				kept = append(kept, t)
			}
		}
		if len(kept) != len(keep) {
			r.store.Replace(id, kept)
			if r.metrics != nil {
				r.metrics.PurgedSamples.Add(int64(len(keep) - len(kept)))
			}
		}
	}
	return nil
}
