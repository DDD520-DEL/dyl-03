package device

import "github.com/dyl-03/shadow/internal/model"

// Supports reports whether a device declares a capability.
func Supports(d *model.Device, capability string) bool {
	if d == nil {
		return false
	}
	for _, c := range d.Capabilities {
		if c == capability {
			return true
		}
	}
	return false
}
