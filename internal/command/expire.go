package command

import (
	"time"

	"github.com/dyl-03/shadow/internal/model"
)

// SweepExpired returns commands whose ack deadline has passed, marking them
// expired so the caller can retry or drop them.
func SweepExpired(commands []*model.Command, policy *RetryPolicy, now time.Time) []*model.Command {
	var expired []*model.Command
	for _, c := range commands {
		if c.State == model.CommandAcked || c.State == model.CommandExpired {
			continue
		}
		if now.After(policy.Deadline(c)) {
			c.State = model.CommandExpired
			expired = append(expired, c)
		}
	}
	return expired
}
