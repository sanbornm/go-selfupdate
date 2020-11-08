package selfupdate

import (
	"os"
)

// CurrentExeUpdatableResolver resolves the thing to be updated as the current
// executable running this program
type CurrentExeUpdatableResolver struct {
}

// Resolve attempts to find the current executable running this program
func (c CurrentExeUpdatableResolver) Resolve() (string, error) {
	return os.Executable()
}
