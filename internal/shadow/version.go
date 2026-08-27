package shadow

import "github.com/dyl-03/shadow/internal/model"

// Bump returns the next desired version for a shadow.
func Bump(sh *model.Shadow) int64 {
	return sh.DesiredVer + 1
}
