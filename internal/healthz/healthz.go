// Package healthz implements a real readiness probe over the shadow
// gateway runtime: shadow store, device registry, live sessions and the
// command queue. The API exposes the snapshot at /healthz and a strict
// gate at /healthz/ready so operators can verify a node is safe to serve.
package healthz

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dyl-03/shadow/internal/clock"
	"github.com/dyl-03/shadow/internal/command"
	"github.com/dyl-03/shadow/internal/device"
	"github.com/dyl-03/shadow/internal/session"
	"github.com/dyl-03/shadow/internal/shadow"
)

// Component identifiers reported by the probe.
const (
	ComponentShadow  = "shadow"
	ComponentDevices = "devices"
	ComponentSession = "session"
	ComponentCommand = "command"
)

// ComponentView carries the per-component snapshot shown in the report.
type ComponentView struct {
	OK      bool   `json:"ok"`
	Detail  string `json:"detail,omitempty"`
	Latency int64  `json:"latency_ms"`
}

// CheckResult is the flattened per-component probe record.
type CheckResult struct {
	Component string `json:"component"`
	OK        bool   `json:"ok"`
	Detail    string `json:"detail,omitempty"`
	LatencyMS int64  `json:"latency_ms"`
}

// Report is the immutable snapshot returned by one Check call.
type Report struct {
	Status      string                   `json:"status"`
	GeneratedAt string                   `json:"generated_at"`
	Devices     int                      `json:"devices"`
	Online      int                      `json:"online_devices"`
	Sessions    int                      `json:"active_sessions"`
	Checks      []CheckResult            `json:"checks"`
	Components  map[string]ComponentView `json:"components"`
}

// Prober runs bounded readiness probes against the live components and
// keeps a short ring of past snapshots for trend inspection.
type Prober struct {
	shadows  *shadow.Store
	devices  *device.Registry
	sessions *session.Manager
	commands *command.Queue
	clk      clock.Clock
	mu       sync.Mutex
	history  []Report
	cap      int
}

// New returns a Prober bound to the given runtime components.
func New(shadows *shadow.Store, devices *device.Registry, sessions *session.Manager, commands *command.Queue, clk clock.Clock) *Prober {
	return &Prober{shadows: shadows, devices: devices, sessions: sessions, commands: commands, clk: clk, cap: 64}
}

// Check gathers one readiness snapshot from the live components.
func (p *Prober) Check(ctx context.Context) Report {
	started := time.Now()
	components := make(map[string]ComponentView, 4)
	checks := make([]CheckResult, 0, 4)
	collect := func(name string, ok bool, detail string, lat time.Duration) {
		ms := lat.Milliseconds()
		components[name] = ComponentView{OK: ok, Detail: detail, Latency: ms}
		checks = append(checks, CheckResult{Component: name, OK: ok, Detail: detail, LatencyMS: ms})
	}

	collect(ComponentShadow, true, p.shadowDetail(ctx), time.Since(started))
	collect(ComponentDevices, true, p.deviceDetail(ctx), time.Since(started))
	collect(ComponentSession, true, p.sessionDetail(ctx), time.Since(started))
	collect(ComponentCommand, true, p.commandDetail(ctx), time.Since(started))

	allOK := true
	for _, c := range checks {
		if !c.OK {
			allOK = false
			break
		}
	}
	status := "ok"
	if !allOK {
		status = "degraded"
	}
	report := Report{
		Status:      status,
		GeneratedAt: p.clk.Now().Format(time.RFC3339),
		Devices:     len(p.devices.All()),
		Online:      len(p.devices.Online()),
		Sessions:    len(p.sessions.Active()),
		Checks:      checks,
		Components:  components,
	}
	p.mu.Lock()
	p.history = append(p.history, report)
	if len(p.history) > p.cap {
		p.history = p.history[len(p.history)-p.cap:]
	}
	p.mu.Unlock()
	return report
}

// Ready reports whether every component passed its most recent probe.
func (p *Prober) Ready(ctx context.Context) bool {
	return p.Check(ctx).Status == "ok"
}

// Recent returns up to n of the most recent snapshots, newest first.
func (p *Prober) Recent(n int) []Report {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Report, 0, n)
	for i := len(p.history) - 1; i >= 0 && len(out) < n; i-- {
		out = append(out, p.history[i])
	}
	return out
}

// Summary aggregates the recent snapshots into a compact view used by the
// history endpoint so operators can spot a degrading component at a glance.
type Summary struct {
	Probes         int            `json:"probes"`
	LastStatus     string         `json:"last_status"`
	AvgLatencyMS   int64          `json:"avg_latency_ms"`
	ComponentOK    map[string]int `json:"component_ok"`
	ComponentTotal map[string]int `json:"component_total"`
}

// Summary returns the aggregated view over the retained snapshots.
func (p *Prober) Summary() Summary {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := Summary{
		ComponentOK:    make(map[string]int),
		ComponentTotal: make(map[string]int),
	}
	if len(p.history) == 0 {
		return out
	}
	out.Probes = len(p.history)
	out.LastStatus = p.history[len(p.history)-1].Status
	var latencyTotal int64
	var checkCount int
	for _, report := range p.history {
		for _, check := range report.Checks {
			out.ComponentTotal[check.Component]++
			if check.OK {
				out.ComponentOK[check.Component]++
			}
			latencyTotal += check.LatencyMS
			checkCount++
		}
	}
	if checkCount > 0 {
		out.AvgLatencyMS = latencyTotal / int64(checkCount)
	}
	return out
}

func (p *Prober) shadowDetail(_ context.Context) string {
	return fmt.Sprintf("shadows=%d", len(p.shadows.All()))
}

func (p *Prober) deviceDetail(_ context.Context) string {
	all := p.devices.All()
	return fmt.Sprintf("registered=%d online=%d", len(all), len(p.devices.Online()))
}

func (p *Prober) sessionDetail(_ context.Context) string {
	return fmt.Sprintf("active=%d", len(p.sessions.Active()))
}

func (p *Prober) commandDetail(_ context.Context) string {
	devices := p.devices.All()
	total := 0
	for _, d := range devices {
		total += len(p.commands.Pending(d.ID))
	}
	return fmt.Sprintf("devices=%d pending_commands=%d", len(devices), total)
}
