package verifycase

import (
	"testing"
	"time"

	"github.com/dyl-03/shadow/internal/clock"
	"github.com/dyl-03/shadow/internal/lifecycle"
	"github.com/dyl-03/shadow/internal/metrics"
	"github.com/dyl-03/shadow/internal/model"
	"github.com/dyl-03/shadow/internal/telemetry"
)

func TestTelemetryRetentionKeepsActive(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	clk := clock.NewManual(now)
	m := metrics.New()
	st := telemetry.NewStore(clk, m, 100)
	if err := st.Append(&model.Telemetry{DeviceID: "d1", TS: now.Add(-10 * time.Minute), Seq: 1}); err != nil {
		t.Fatal(err)
	}
	if err := st.Append(&model.Telemetry{DeviceID: "d1", TS: now.Add(-2 * time.Hour), Seq: 2}); err != nil {
		t.Fatal(err)
	}
	rt := lifecycle.NewRetention(st, clk, time.Hour, m)
	if err := rt.Purge([]string{"d1"}); err != nil {
		t.Fatal(err)
	}
	recent := st.Recent("d1")
	if len(recent) != 1 || recent[0].Seq != 1 {
		t.Fatalf("active sample was purged, surviving=%+v", recent)
	}
}
