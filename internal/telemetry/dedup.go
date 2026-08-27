package telemetry

// SeqTracker rejects out-of-order or duplicate telemetry sequences.
type SeqTracker struct {
	last map[string]int64
}

func NewSeqTracker() *SeqTracker {
	return &SeqTracker{last: make(map[string]int64)}
}

// Accept returns true when the sequence is newer than the previous one.
func (s *SeqTracker) Accept(deviceID string, seq int64) bool {
	last, ok := s.last[deviceID]
	if ok && seq <= last {
		return false
	}
	s.last[deviceID] = seq
	return true
}
