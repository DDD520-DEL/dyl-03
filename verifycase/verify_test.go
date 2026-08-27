package verifycase

import (
	"testing"
	"time"

	"github.com/dyl-03/shadow/internal/command"
	"github.com/dyl-03/shadow/internal/model"
)

func TestCommandAppliedInSequenceOrder(t *testing.T) {
	buf := command.NewBuffer()
	now := time.Unix(1_700_000_000, 0)
	buf.Enqueue(&model.Command{ID: "c2", DeviceID: "d1", Seq: 2, EnqueuedAt: now})
	buf.Enqueue(&model.Command{ID: "c1", DeviceID: "d1", Seq: 1, EnqueuedAt: now})
	got := buf.Deliver("d1")
	if len(got) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(got))
	}
	if got[0].Seq != 1 || got[1].Seq != 2 {
		t.Fatalf("commands not delivered in sequence order: %+v", got)
	}
}
