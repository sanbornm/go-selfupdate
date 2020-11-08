package main

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/kr/binarydist"
	"github.com/stretchr/testify/assert"
)

// Kind of my own sanity check that diffing and patching actually works..
func TestCreatingPatchThenApplyingPatchYeildsSameBinary(t *testing.T) {
	// ******************************* ARRANGE ********************************
	oldBinary, err := ioutil.ReadFile("../../golden_data/main-before.exe")
	assert.NoError(t, err)

	newBinary, err := ioutil.ReadFile("../../golden_data/main-new.exe")
	assert.NoError(t, err)

	// ********************************* ACT **********************************
	patch, err := getPatch(bytes.NewReader(oldBinary), bytes.NewReader(newBinary))

	// ******************************** ASSERT ********************************
	assert.NoError(t, err)
	var buf bytes.Buffer
	err = binarydist.Patch(bytes.NewReader(oldBinary), &buf, patch)
	assert.NoError(t, err)
	assert.True(t, bytes.Equal(newBinary, buf.Bytes()))
}

func TestCreatingPatchFromGZThenApplyingPatchYeildsSameBinary(t *testing.T) {
	// ******************************* ARRANGE ********************************
	oldBinary, err := ioutil.ReadFile("../../golden_data/main-before.exe")
	assert.NoError(t, err)

	newBinary, err := ioutil.ReadFile("../../golden_data/main-new.exe")
	assert.NoError(t, err)

	oldBinaryCompressed, err := compressFile("../../golden_data/main-before.exe")
	assert.NoError(t, err)

	newBinaryCompressed, err := compressFile("../../golden_data/main-new.exe")
	assert.NoError(t, err)

	// ********************************* ACT **********************************
	patch, err := getPatchFromGzFiles(
		ioutil.NopCloser(bytes.NewReader(oldBinaryCompressed)),
		ioutil.NopCloser(bytes.NewReader(newBinaryCompressed)),
	)

	// ******************************** ASSERT ********************************
	assert.NoError(t, err)
	var buf bytes.Buffer
	err = binarydist.Patch(bytes.NewReader(oldBinary), &buf, patch)
	assert.NoError(t, err)
	assert.True(t, bytes.Equal(newBinary, buf.Bytes()))
}
