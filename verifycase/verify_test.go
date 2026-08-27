package verifycase

import (
	"testing"
	"time"

	"github.com/dyl-03/shadow/internal/command"
	"github.com/dyl-03/shadow/internal/model"
)

func TestCommandRetryFromLastDelivery(t *testing.T) {
	policy := command.DefaultRetry()
	enqueued := time.Unix(100, 0)
	delivered := time.Unix(200, 0)
	cmd := &model.Command{ID: "c1", DeviceID: "d1", EnqueuedAt: enqueued, LastDelivered: delivered, Attempts: 1}
	want := delivered.Add(policy.Timeout)
	if got := policy.Deadline(cmd); !got.Equal(want) {
		t.Fatalf("deadline %s should be based on last delivery %s, not enqueue %s", got, want, enqueued)
	}
}
