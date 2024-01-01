package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
)

func TestUpdater(t *testing.T) {
}

func TestGenerateSha256(t *testing.T) {
	t.Parallel()
	tempFile, err := os.CreateTemp("", "test_file")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	content := []byte("Hello, World!")
	if _, err := tempFile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatal(err)
	}

	expectedHash := sha256.Sum256(content)

	result := generateSha256(tempFile.Name())

	resultStr := hex.EncodeToString(result)
	expectedStr := hex.EncodeToString(expectedHash[:])

	if resultStr != expectedStr {
		t.Errorf("Expected '%s', but got '%s'", expectedStr, resultStr)
	}
}
