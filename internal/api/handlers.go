package api

import (
	"context"
	"fmt"
	"time"

	"github.com/dyl-03/shadow/internal/audit"
	"github.com/dyl-03/shadow/internal/clock"
	"github.com/dyl-03/shadow/internal/command"
	"github.com/dyl-03/shadow/internal/device"
	"github.com/dyl-03/shadow/internal/healthz"
	"github.com/dyl-03/shadow/internal/idgen"
	"github.com/dyl-03/shadow/internal/metrics"
	"github.com/dyl-03/shadow/internal/model"
	"github.com/dyl-03/shadow/internal/ota"
	"github.com/dyl-03/shadow/internal/session"
	"github.com/dyl-03/shadow/internal/shadow"
	"github.com/dyl-03/shadow/internal/telemetry"
)

// Server exposes the control-plane API.
type Server struct {
	shadows   *shadow.Store
	devices   *device.Registry
	templates *device.TemplateStore
	commands  *command.Queue
	dispatcher *command.Dispatcher
	planner   *ota.GroupPlanner
	parser    *command.Parser
	ids       *idgen.Generator
	audit     *audit.Log
	clock     clock.Clock
	metrics   *metrics.Metrics
	timeout   time.Duration
	ingest    *telemetry.Ingest
	sessions  *session.Manager
	health    *healthz.Prober
}

type Deps struct {
	Shadows    *shadow.Store
	Devices    *device.Registry
	Templates  *device.TemplateStore
	Commands   *command.Queue
	Dispatcher *command.Dispatcher
	Planner    *ota.GroupPlanner
	Parser     *command.Parser
	IDs        *idgen.Generator
	Audit      *audit.Log
	Clock      clock.Clock
	Metrics    *metrics.Metrics
	Timeout    time.Duration
	Ingest     *telemetry.Ingest
	Sessions   *session.Manager
	Health     *healthz.Prober
}

func New(d Deps) *Server {
	return &Server{
		shadows: d.Shadows, devices: d.Devices, templates: d.Templates,
		commands: d.Commands, dispatcher: d.Dispatcher, planner: d.Planner,
		parser: d.Parser, ids: d.IDs, audit: d.Audit, clock: d.Clock,
		metrics: d.Metrics, timeout: d.Timeout, ingest: d.Ingest,
		sessions: d.Sessions, health: d.Health,
	}
}

// RegisterDevice registers a device and optional template.
func (s *Server) RegisterDevice(id, group, template string, caps []string) (*model.Device, error) {
	d := &model.Device{ID: id, Group: group, Template: template, Presence: model.PresenceOffline, Capabilities: caps}
	if err := s.devices.Register(d); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(id, "register", group)
	}
	return s.devices.Get(id), nil
}

// GetShadow returns the current shadow.
func (s *Server) GetShadow(id string) (*model.Shadow, error) {
	if s.devices.Get(id) == nil {
		return nil, fmt.Errorf("device %s not found", id)
	}
	return s.shadows.Get(id), nil
}

// UpdateDesired updates one device's desired state with version check.
func (s *Server) UpdateDesired(ctx context.Context, id string, version int64, patch map[string]string) (*model.Shadow, error) {
	if s.devices.Get(id) == nil {
		return nil, fmt.Errorf("device %s not found", id)
	}
	sh, err := s.shadows.UpdateDesired(id, version, patch)
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(id, "desired", fmt.Sprintf("v%d", sh.DesiredVer))
	}
	return sh, nil
}

// MoveDevice migrates a device to a new OTA group.
func (s *Server) MoveDevice(id, group string) (*model.Device, error) {
	if err := s.planner.Move(id, group); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(id, "move", group)
	}
	return s.devices.Get(id), nil
}

// SendCommand enqueues a command for a device and parses it against its
// template when the device is online.
func (s *Server) SendCommand(ctx context.Context, id string, payload []byte) (*model.Command, error) {
	dev := s.devices.Get(id)
	if dev == nil {
		return nil, fmt.Errorf("device %s not found", id)
	}
	if len(dev.Capabilities) > 0 && !device.Supports(dev, "command") {
		return nil, fmt.Errorf("device %s does not support commands", id)
	}
	if _, err := s.parser.Parse(dev, payload); err != nil {
		return nil, err
	}
	cmd, err := s.commands.Submit(id, payload, s.timeout)
	if err != nil {
		return nil, err
	}
	if dev.Presence == model.PresenceOnline {
		if err := s.dispatcher.Dispatch(cmd); err != nil {
			return nil, err
		}
	}
	if s.audit != nil {
		s.audit.Record(id, "command", cmd.ID)
	}
	return cmd, nil
}

// History returns the audit trail of a device.
func (s *Server) History(id string) ([]audit.Entry, error) {
	if s.devices.Get(id) == nil {
		return nil, fmt.Errorf("device %s not found", id)
	}
	return s.audit.History(id), nil
}

// Diff returns reported fields that diverge from the desired state.
func (s *Server) Diff(id string) ([]string, error) {
	if s.devices.Get(id) == nil {
		return nil, fmt.Errorf("device %s not found", id)
	}
	return shadow.Diff(s.shadows.Get(id)), nil
}

// SubmitTelemetry ingests a telemetry sample for a device.
func (s *Server) SubmitTelemetry(id string, seq int64, fields map[string]float64) error {
	if s.devices.Get(id) == nil {
		return fmt.Errorf("device %s not found", id)
	}
	if s.ingest == nil {
		return fmt.Errorf("telemetry ingest disabled")
	}
	return s.ingest.Accept(id, seq, fields)
}

// Health reports service liveness.
func (s *Server) Health() map[string]any {
	return map[string]any{"status": "ok", "time": s.clock.Now().Format(time.RFC3339)}
}

// Stats returns operation counters for the gateway.
func (s *Server) Stats() map[string]any {
	return map[string]any{
		"commands_sent":   s.metrics.CommandsSent.Load(),
		"commands_acked":  s.metrics.CommandsAcked.Load(),
		"commands_retry":  s.metrics.CommandsRetry.Load(),
		"sessions_open":   s.metrics.SessionsOpened.Load(),
		"sessions_close":  s.metrics.SessionsClosed.Load(),
		"active_sessions": s.metrics.ActiveSessions.Load(),
		"shadows":         s.metrics.ShadowsUpdated.Load(),
		"telemetry_seen":  s.metrics.TelemetrySeen.Load(),
		"telemetry_dropped": s.metrics.TelemetryDropped.Load(),
		"purged_samples":  s.metrics.PurgedSamples.Load(),
		"buffer_entries":  s.metrics.BufferEntries.Load(),
	}
}
