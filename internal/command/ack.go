package command

import (
	"fmt"
	"sync"

	"github.com/dyl-03/shadow/internal/model"
)

// AckTable tracks command acknowledgement state.
type AckTable struct {
	mu       sync.Mutex
	byID     map[string]*model.Command
	byDevice map[string][]string
}

func NewAckTable() *AckTable {
	return &AckTable{byID: make(map[string]*model.Command), byDevice: make(map[string][]string)}
}

func (a *AckTable) Track(cmd *model.Command) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.byID[cmd.ID] = cmd.Clone()
	a.byDevice[cmd.DeviceID] = append(a.byDevice[cmd.DeviceID], cmd.ID)
}

func (a *AckTable) Ack(cmdID string) (*model.Command, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	cmd, ok := a.byID[cmdID]
	if !ok {
		return nil, fmt.Errorf("unknown command %s", cmdID)
	}
	cmd.State = model.CommandAcked
	a.byID[cmdID] = cmd
	return cmd.Clone(), nil
}

func (a *AckTable) Pending(deviceID string) []*model.Command {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []*model.Command
	for _, id := range a.byDevice[deviceID] {
		if c := a.byID[id]; c != nil {
			out = append(out, c.Clone())
		}
	}
	return out
}

func (a *AckTable) Count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.byID)
}
