package command

import (
	"github.com/dyl-03/shadow/internal/device"
	"github.com/dyl-03/shadow/internal/metrics"
	"github.com/dyl-03/shadow/internal/model"
	"github.com/dyl-03/shadow/internal/ota"
)

// Dispatcher routes commands to the correct device group and marks delivery.
type Dispatcher struct {
	groupIndex *device.GroupIndex
	routes     *ota.Routes
	metrics    *metrics.Metrics
}

func NewDispatcher(g *device.GroupIndex, r *ota.Routes, m *metrics.Metrics) *Dispatcher {
	return &Dispatcher{groupIndex: g, routes: r, metrics: m}
}

func (d *Dispatcher) Dispatch(cmd *model.Command) error {
	group := d.groupIndex.GroupOf(cmd.DeviceID)
	target, err := d.routes.Resolve(group)
	if err != nil {
		return err
	}
	cmd.State = model.CommandDelivered
	if d.metrics != nil {
		d.metrics.CommandsSent.Add(1)
	}
	_ = target
	return nil
}
