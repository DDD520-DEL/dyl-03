package verifycase

import (
	"context"
	"testing"
	"time"

	"github.com/dyl-03/shadow/internal/api"
	"github.com/dyl-03/shadow/internal/audit"
	"github.com/dyl-03/shadow/internal/clock"
	"github.com/dyl-03/shadow/internal/command"
	"github.com/dyl-03/shadow/internal/device"
	"github.com/dyl-03/shadow/internal/idgen"
	"github.com/dyl-03/shadow/internal/metrics"
	"github.com/dyl-03/shadow/internal/model"
	"github.com/dyl-03/shadow/internal/ota"
	"github.com/dyl-03/shadow/internal/shadow"
	"github.com/dyl-03/shadow/internal/wal"
)

func TestBatchDesiredStateAllOrNothing(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	clk := clock.NewManual(now)
	m := metrics.New()
	wm, err := wal.NewManager(t.TempDir(), 1<<20, m)
	if err != nil {
		t.Fatal(err)
	}
	defer wm.Close()
	shadows := shadow.New(wm, clk, m)
	reg := device.NewRegistry()
	tpl := device.NewTemplateStore()
	if err := tpl.Register(&model.Template{Name: "thermostat", Schema: map[string]string{"mode": "string"}}); err != nil {
		t.Fatal(err)
	}
	idx := device.NewGroupIndex(reg)
	routes := ota.NewRoutes()
	routes.Set("default", "v1")
	planner := ota.NewGroupPlanner(reg, idx)
	buf := command.NewBuffer()
	ack := command.NewAckTable()
	queue := command.NewQueue(idgen.New(1), clk, buf, ack, m)
	disp := command.NewDispatcher(idx, routes, m)
	parser := command.NewParser(tpl)
	srv := api.New(api.Deps{
		Shadows: shadows, Devices: reg, Templates: tpl, Commands: queue,
		Dispatcher: disp, Planner: planner, Parser: parser, IDs: idgen.New(1),
		Audit: audit.New(10, clk), Clock: clk, Metrics: m, Timeout: time.Minute,
	})
	for _, id := range []string{"d1", "d2"} {
		if _, err := srv.RegisterDevice(id, "default", "thermostat", nil); err != nil {
			t.Fatal(err)
		}
	}
	_, err = srv.SubmitBatchDesired(context.Background(), []string{"d1", "d2", "missing"}, 0, map[string]string{"mode": "heat"})
	if err == nil {
		t.Fatal("expected batch to fail for unknown device")
	}
	if got := len(shadows.All()); got != 0 {
		t.Fatalf("batch failure left %d shadows written", got)
	}
}
