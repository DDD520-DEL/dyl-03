package command

import (
	"time"

	"github.com/dyl-03/shadow/internal/model"
)

// RetryPolicy controls command re-delivery deadlines.
type RetryPolicy struct {
	MaxAttempts int
	Timeout     time.Duration
}

func DefaultRetry() *RetryPolicy {
	return &RetryPolicy{MaxAttempts: 3, Timeout: 60 * time.Second}
}

// Deadline computes the ack deadline from the most recent delivery, so queued
// commands are not instantly considered expired.
func (p *RetryPolicy) Deadline(cmd *model.Command) time.Time {
	base := cmd.LastDelivered
	if base.IsZero() {
		base = cmd.EnqueuedAt
	}
	return base.Add(p.Timeout)
}

func (p *RetryPolicy) CanRetry(cmd *model.Command) bool {
	return cmd.Attempts < p.MaxAttempts
}
