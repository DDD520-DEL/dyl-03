package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dyl-03/shadow/internal/api"
	"github.com/dyl-03/shadow/internal/audit"
	"github.com/dyl-03/shadow/internal/clock"
	"github.com/dyl-03/shadow/internal/command"
	"github.com/dyl-03/shadow/internal/config"
	"github.com/dyl-03/shadow/internal/device"
	"github.com/dyl-03/shadow/internal/healthz"
	"github.com/dyl-03/shadow/internal/idgen"
	"github.com/dyl-03/shadow/internal/lifecycle"
	"github.com/dyl-03/shadow/internal/metrics"
	"github.com/dyl-03/shadow/internal/model"
	"github.com/dyl-03/shadow/internal/ota"
	"github.com/dyl-03/shadow/internal/session"
	"github.com/dyl-03/shadow/internal/shadow"
	"github.com/dyl-03/shadow/internal/telemetry"
	"github.com/dyl-03/shadow/internal/wal"
)

func main() {
	configPath := flag.String("config", "", "path to JSON config file")
	flag.Parse()
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	clk := clock.Wall()
	m := metrics.New()
	ids := idgen.New(1)

	walMgr, err := wal.NewManager(cfg.WALDir, cfg.WALSegmentBytes, m)
	if err != nil {
		log.Fatalf("open wal: %v", err)
	}
	defer walMgr.Close()

	shadows := shadow.New(walMgr, clk, m)
	if err := shadows.Recover(); err != nil {
		log.Fatalf("recover wal: %v", err)
	}
	registry := device.NewRegistry()
	templates := device.NewTemplateStore()
	if err := templates.Register(&model.Template{Name: "thermostat", Schema: map[string]string{"mode": "string", "temp": "int"}}); err != nil {
		log.Fatalf("register template: %v", err)
	}
	if err := templates.Register(&model.Template{Name: "relay", Schema: map[string]string{"state": "bool"}}); err != nil {
		log.Fatalf("register template: %v", err)
	}

	groupIndex := device.NewGroupIndex(registry)
	routes := ota.NewRoutes()
	routes.Set("default", "v1.0.0")
	planner := ota.NewGroupPlanner(registry, groupIndex)

	buffers := session.NewBufferStore(m)
	heartbeat := session.NewHeartbeat(clk)
	sessions := session.NewManager(registry, buffers, heartbeat, clk, m)
	cmdBuffer := command.NewBuffer()
	ackTable := command.NewAckTable()
	queue := command.NewQueue(ids, clk, cmdBuffer, ackTable, m)
	dispatcher := command.NewDispatcher(groupIndex, routes, m)
	parser := command.NewParser(templates)
	retryPolicy := command.DefaultRetry()

	teleStore := telemetry.NewStore(clk, m, 10000)
	seqTracker := telemetry.NewSeqTracker()
	ingest := telemetry.NewIngest(teleStore, seqTracker, clk, m)
	retention := lifecycle.NewRetention(teleStore, clk, cfg.RetentionWindow, m)
	go runRetention(ctx, retention, registry)
	go runHeartbeat(ctx, sessions, heartbeat, cfg.SessionTimeout)
	go runCommandSweep(ctx, queue, sessions, retryPolicy)
	go runRollout(ctx, registry, routes)

	auditLog := audit.New(200, clk)
	apiServer := api.New(api.Deps{
		Shadows: shadows, Devices: registry, Templates: templates,
		Commands: queue, Dispatcher: dispatcher, Planner: planner,
		Parser: parser, IDs: ids, Audit: auditLog, Clock: clk,
		Metrics: m, Timeout: cfg.CommandTimeout, Ingest: ingest,
		Sessions: sessions, Health: healthz.New(shadows, registry, sessions, queue, clk),
	})

	log.Printf("shadow gateway listening on %s", cfg.ListenAddr)
	if err := apiServer.Serve(ctx, cfg.ListenAddr); err != nil && ctx.Err() == nil {
		log.Fatalf("serve: %v", err)
	}
}

func runRetention(ctx context.Context, r *lifecycle.Retention, reg *device.Registry) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			devices := reg.All()
			ids := make([]string, 0, len(devices))
			for _, d := range devices {
				ids = append(ids, d.ID)
			}
			if err := r.Purge(ids); err != nil {
				log.Printf("retention: %v", err)
			}
		}
	}
}

func runHeartbeat(ctx context.Context, mgr *session.Manager, hb *session.Heartbeat, timeout time.Duration) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, s := range mgr.Active() {
				if hb.Stale(s.ID, timeout) {
					mgr.Close(s.ID)
					hb.Drop(s.ID)
				}
			}
		}
	}
}

func runCommandSweep(ctx context.Context, q *command.Queue, mgr *session.Manager, policy *command.RetryPolicy) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			for _, s := range mgr.Active() {
				expired := command.SweepExpired(q.Pending(s.DeviceID), policy, now)
				for _, c := range expired {
					_ = c
				}
			}
		}
	}
}

func runRollout(ctx context.Context, reg *device.Registry, routes *ota.Routes) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			selected := ota.SelectGroup(reg, "default", 10)
			_, _ = routes.Resolve("default")
			_ = selected
		}
	}
}
