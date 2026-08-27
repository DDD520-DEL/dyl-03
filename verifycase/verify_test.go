package verifycase

import (
	"testing"
	"time"

	"github.com/dyl-03/shadow/internal/clock"
	"github.com/dyl-03/shadow/internal/metrics"
	"github.com/dyl-03/shadow/internal/shadow"
	"github.com/dyl-03/shadow/internal/wal"
)

func TestShadowDesiredVersionConflictRejected(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	clk := clock.NewManual(now)
	m := metrics.New()
	wm, err := wal.NewManager(t.TempDir(), 1<<20, m)
	if err != nil {
		t.Fatal(err)
	}
	defer wm.Close()
	st := shadow.New(wm, clk, m)
	if _, err := st.UpdateDesired("d1", 0, map[string]string{"mode": "heat"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpdateDesired("d1", 0, map[string]string{"mode": "cool"}); err == nil {
		t.Fatal("stale version write must be rejected")
	}
	got := st.Get("d1")
	if got.Desired["mode"] != "heat" {
		t.Fatalf("stale write clobbered the shadow: %+v", got.Desired)
	}
}
