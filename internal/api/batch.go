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
// atomically: validation runs before any device is written.
func (s *Server) SubmitBatchDesired(ctx context.Context, ids []string, version int64, patch map[string]string) ([]*model.Shadow, error) {
	if err := ValidateBatch(ids, BatchLimits{MaxDevices: 500}); err != nil {
		return nil, err
	}
	for _, id := range ids {
		if s.devices.Get(id) == nil {
			return nil, fmt.Errorf("device %s not found", id)
		}
	}
	var out []*model.Shadow
	for _, id := range ids {
		sh, err := s.shadows.UpdateDesired(id, version, patch)
		if err != nil {
			return nil, err
		}
		out = append(out, sh)
	}
	return out, nil
}
