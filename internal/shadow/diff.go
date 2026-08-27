package shadow

import "github.com/dyl-03/shadow/internal/model"

// Diff returns reported fields that diverge from the desired state.
func Diff(sh *model.Shadow) []string {
	var out []string
	for key, want := range sh.Desired {
		if got, ok := sh.Reported[key]; !ok || got != want {
			out = append(out, key)
		}
	}
	return out
}
