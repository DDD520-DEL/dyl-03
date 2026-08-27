package model

import "time"

// Telemetry is one reported sample from a device.
type Telemetry struct {
	DeviceID string
	TS       time.Time
	Seq      int64
	Fields   map[string]float64
}

// Clone returns a deep copy.
func (t *Telemetry) Clone() *Telemetry {
	if t == nil {
		return nil
	}
	c := *t
	c.Fields = make(map[string]float64, len(t.Fields))
	for k, v := range t.Fields {
		c.Fields[k] = v
	}
	return &c
}
