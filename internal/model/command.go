package model

import "time"

// CommandState is the lifecycle state of a device command.
type CommandState string

const (
	CommandQueued    CommandState = "queued"
	CommandDelivered CommandState = "delivered"
	CommandAcked     CommandState = "acked"
	CommandExpired   CommandState = "expired"
)

// Command is an instruction sent to a device.
type Command struct {
	ID            string
	DeviceID      string
	Seq           int64
	Payload       []byte
	State         CommandState
	EnqueuedAt    time.Time
	LastDelivered time.Time
	Attempts      int
	Timeout       time.Duration
}

// Clone returns a deep copy.
func (c *Command) Clone() *Command {
	if c == nil {
		return nil
	}
	cp := *c
	cp.Payload = append([]byte(nil), c.Payload...)
	return &cp
}
