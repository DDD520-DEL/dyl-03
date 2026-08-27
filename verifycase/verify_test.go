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

func TestSessionBuffersReleasedOnClose(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	clk := clock.NewManual(now)
	m := metrics.New()
	reg := device.NewRegistry()
	if err := reg.Register(&model.Device{ID: "d1", Group: "g", Template: "t", Presence: model.PresenceOffline}); err != nil {
		t.Fatal(err)
	}
	bufs := session.NewBufferStore(m)
	bufs.Enqueue("d1", "msg")
	mg := session.NewManager(reg, bufs, session.NewHeartbeat(clk), clk, m)
	sess := mg.Open("d1", "s1")
	mg.Close(sess.ID)
	if bufs.Count() != 0 || len(bufs.Pending("d1")) != 0 {
		t.Fatalf("session close must release the device buffer, count=%d", bufs.Count())
	}
}
