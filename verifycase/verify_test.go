package verifycase

import (
	"testing"
	"time"

	"github.com/dyl-03/shadow/internal/command"
	"github.com/dyl-03/shadow/internal/model"
)

func TestOfflineCommandDeliveredOnce(t *testing.T) {
	buf := command.NewBuffer()
	now := time.Unix(1_700_000_000, 0)
	buf.Enqueue(&model.Command{ID: "c1", DeviceID: "d1", Seq: 1, EnqueuedAt: now})
	buf.Enqueue(&model.Command{ID: "c2", DeviceID: "d1", Seq: 2, EnqueuedAt: now})
	if got := len(buf.Deliver("d1")); got != 2 {
		t.Fatalf("first delivery should return 2, got %d", got)
	}
	if got := len(buf.Deliver("d1")); got != 0 {
		t.Fatalf("second delivery must be empty, got %d", got)
	}
}
