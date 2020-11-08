package selfupdate

import (
	"fmt"
	"runtime"
)

// CurrentPlatformResolver will create a key that corresponds to the current
// OS and architecture this program is running on
type CurrentPlatformResolver struct {
}

// Resolve returns the current architecture and operating system
func (c CurrentPlatformResolver) Resolve() (string, error) {
	return fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH), nil
}
