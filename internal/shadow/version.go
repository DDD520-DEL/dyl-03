package shadow

import (
	"errors"

	"github.com/dyl-03/shadow/internal/model"
)

// ErrVersionConflict is returned by UpdateDesired when the caller's expected
// version does not match the shadow's current version — another writer has
// advanced the state since the caller last read it. The write is rejected
// without mutating state, so concurrent writers cannot clobber each other.
var ErrVersionConflict = errors.New("shadow: version conflict")

// Bump returns the next desired version for a shadow.
func Bump(sh *model.Shadow) int64 {
	return sh.DesiredVer + 1
}
