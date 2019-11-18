package selfupdate

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
)

func getExecRelativeDir(dir string) (string, error) {
	filename, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(filename), dir), nil
}

func verifySha(bin []byte, sha []byte) bool {
	h := sha256.New()
	h.Write(bin)
	return bytes.Equal(h.Sum(nil), sha)
}
