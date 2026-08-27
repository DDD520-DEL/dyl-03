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
	"github.com/dyl-03/shadow/internal/ota"
	"github.com/dyl-03/shadow/internal/shadow"
	"github.com/dyl-03/shadow/internal/wal"
)

func TestUnregisteredTemplateNoPanic(t *testing.T) {
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
	if _, err := srv.RegisterDevice("d1", "default", "missing-template", []string{"command"}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.SendCommand(context.Background(), "d1", []byte("x")); err == nil {
		t.Fatal("command to unregistered template must return an error, not panic")
	}
}
