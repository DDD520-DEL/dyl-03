package command

import (
	"time"

	"github.com/dyl-03/shadow/internal/clock"
	"github.com/dyl-03/shadow/internal/idgen"
	"github.com/dyl-03/shadow/internal/metrics"
	"github.com/dyl-03/shadow/internal/model"
)

// Queue accepts new commands and fans them to the offline buffer or the
// dispatcher depending on device presence.
type Queue struct {
	ids     *idgen.Generator
	clock   clock.Clock
	buffer  *Buffer
	ack     *AckTable
	metrics *metrics.Metrics
	seq     int64
}

func NewQueue(ids *idgen.Generator, clk clock.Clock, buf *Buffer, ack *AckTable, m *metrics.Metrics) *Queue {
	return &Queue{ids: ids, clock: clk, buffer: buf, ack: ack, metrics: m, seq: 1000}
}

func (q *Queue) Submit(deviceID string, payload []byte, timeout time.Duration) (*model.Command, error) {
	id, err := q.ids.Next()
	if err != nil {
		return nil, err
	}
	q.seq++
	cmd := &model.Command{
		ID:         id,
		DeviceID:   deviceID,
		Seq:        q.seq,
		Payload:    payload,
		State:      model.CommandQueued,
		EnqueuedAt: q.clock.Now(),
		Timeout:    timeout,
	}
	q.buffer.Enqueue(cmd)
	if q.ack != nil {
		q.ack.Track(cmd)
	}
	return cmd.Clone(), nil
}

// Pending returns the tracked commands of a device.
func (q *Queue) Pending(deviceID string) []*model.Command {
	if q.ack == nil {
		return nil
	}
	return q.ack.Pending(deviceID)
}
