package verifycase

import (
	"testing"

	"github.com/dyl-03/shadow/internal/device"
	"github.com/dyl-03/shadow/internal/model"
	"github.com/dyl-03/shadow/internal/ota"
)

func TestOtaRouteFollowsGroupMigration(t *testing.T) {
	reg := device.NewRegistry()
	if err := reg.Register(&model.Device{ID: "d1", Group: "A", Template: "t", Presence: model.PresenceOffline}); err != nil {
		t.Fatal(err)
	}
	idx := device.NewGroupIndex(reg)
	routes := ota.NewRoutes()
	routes.Set("A", "v1")
	routes.Set("B", "v2")
	planner := ota.NewGroupPlanner(reg, idx)
	if err := planner.Move("d1", "B"); err != nil {
		t.Fatal(err)
	}
	if got := idx.GroupOf("d1"); got != "B" {
		t.Fatalf("routing index still points at group %q after migration", got)
	}
	version, err := routes.Resolve(idx.GroupOf("d1"))
	if err != nil || version != "v2" {
		t.Fatalf("route did not follow migration: %q %v", version, err)
	}
}
