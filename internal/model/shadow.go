package model

import "time"

// Shadow holds the desired and reported state of one device.
type Shadow struct {
	DeviceID     string
	Desired      map[string]string
	Reported     map[string]string
	DesiredVer   int64
	ReportedVer  int64
	UpdatedAt    time.Time
}

// Clone returns a deep copy of the shadow.
func (s *Shadow) Clone() *Shadow {
	if s == nil {
		return nil
	}
	c := *s
	c.Desired = cloneMap(s.Desired)
	c.Reported = cloneMap(s.Reported)
	return &c
}

func cloneMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
