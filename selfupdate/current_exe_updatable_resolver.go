package selfupdate

import (
	"io"
	"os"
)

type CurrentExeUpdatableResolver struct {
}

func (c CurrentExeUpdatableResolver) Resolve() (io.ReadCloser, error) {
	path, err := os.Executable()
	if err != nil {
		return nil, err
	}
	old, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return old, nil
}
