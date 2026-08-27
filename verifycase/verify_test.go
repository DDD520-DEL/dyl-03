package verifycase

import (
	"testing"
	"time"

	"github.com/dyl-03/shadow/internal/clock"
	"github.com/dyl-03/shadow/internal/device"
	"github.com/dyl-03/shadow/internal/metrics"
	"github.com/dyl-03/shadow/internal/model"
	"github.com/dyl-03/shadow/internal/session"
)

func TestReconnectSweepsStaleSession(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	clk := clock.NewManual(now)
	m := metrics.New()
	reg := device.NewRegistry()
	if err := reg.Register(&model.Device{ID: "d1", Group: "g", Template: "t", Presence: model.PresenceOffline}); err != nil {
		t.Fatal(err)
	}
	hb := session.NewHeartbeat(clk)
	mg := session.NewManager(reg, session.NewBufferStore(m), hb, clk, m)
	mg.Open("d1", "s1")
	mg.Open("d1", "s2")
	active := mg.Active()
	if len(active) != 1 || active[0].ID != "s2" {
		t.Fatalf("expected only the new session to remain, got %+v", active)
	}
}
