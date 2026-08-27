package api

import (
	"context"
	"fmt"

	"github.com/dyl-03/shadow/internal/model"
)

// BatchLimits bounds a batch desired-state submission.
type BatchLimits struct {
	MaxDevices int
}

// ValidateBatch checks the batch without mutating anything.
func ValidateBatch(ids []string, limits BatchLimits) error {
	if len(ids) == 0 {
		return fmt.Errorf("batch must contain at least one device")
	}
	if len(ids) > limits.MaxDevices {
		return fmt.Errorf("batch size %d exceeds limit %d", len(ids), limits.MaxDevices)
	}
	return nil
}

// SubmitBatchDesired applies a desired-state patch to many devices
// atomically: every check (size, duplicates, device existence) runs before
// any device is written, and the writes themselves are committed as a single
// all-or-nothing batch so a mid-batch failure cannot leave some devices
// updated and others untouched.
func (s *Server) SubmitBatchDesired(ctx context.Context, ids []string, version int64, patch map[string]string) ([]*model.Shadow, error) {
	if err := ValidateBatch(ids, BatchLimits{MaxDevices: 500}); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			return nil, fmt.Errorf("duplicate device %s in batch", id)
		}
		seen[id] = struct{}{}
	}
	for _, id := range ids {
		if s.devices.Get(id) == nil {
			return nil, fmt.Errorf("device %s not found", id)
		}
	}
	return s.shadows.UpdateDesiredBatch(ids, version, patch)
}
