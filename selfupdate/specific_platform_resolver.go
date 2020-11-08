package selfupdate

import (
	"fmt"
)

// SpecificPlatformResolver will create a key that based on the operating
// system and architecture passed in.
type SpecificPlatformResolver struct {
	os   string
	arch string
}

func NewSpecificPlatformResolver(os, arch string) SpecificPlatformResolver {
	return SpecificPlatformResolver{os: os, arch: arch}
}

// Resolve returns the architecture and operating system used to build this
// resolver
func (c SpecificPlatformResolver) Resolve() (string, error) {
	return fmt.Sprintf("%s-%s", c.os, c.arch), nil
}
