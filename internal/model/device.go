package model

import "time"

// Presence describes whether a device currently holds a live session.
type Presence string

const (
	PresenceOnline  Presence = "online"
	PresenceOffline Presence = "offline"
)

// Device is the registry record for one connected unit.
type Device struct {
	ID          string
	Group       string
	Template    string
	Presence    Presence
	OnlineSince time.Time
	SessionID   string
	Version     int64
	Capabilities []string
}

// Clone returns a deep copy.
func (d *Device) Clone() *Device {
	if d == nil {
		return nil
	}
	c := *d
	c.Capabilities = append([]string(nil), d.Capabilities...)
	return &c
}
