package selfupdate

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/EliCDavis/go-selfupdate/selfupdate/mocks"

	"github.com/golang/mock/gomock"
)

var testHash = sha256.New()

func cpFileAsTemp(fileToCopy, newFlieName string) *os.File {
	content, err := ioutil.ReadFile(fileToCopy)
	if err != nil {
		log.Fatal(err)
	}

	tmpfile, err := ioutil.TempFile("", newFlieName)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := tmpfile.Write(content); err != nil {
		log.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
	}

	return tmpfile
}

func createTestVersionInfo(version, hash string) string {
	return fmt.Sprintf(`{
		"Version": "%s",
		"Sha256": "%s"
	}`, version, hash)
}

func TestUpdaterFetchMustReturnNonNilReaderCloser(t *testing.T) {
	// ******************************* ARRANGE ********************************
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mr := mocks.NewMockRequester(ctrl)
	mr.EXPECT().
		Fetch("http://updates.yourdomain.com/myapp/windows-amd64.json").
		Return(nil, nil)

	platformResolver := NewSpecificPlatformResolver("windows", "amd64")

	updater := createUpdater("", mr, nil, platformResolver)

	// ********************************* ACT **********************************
	updated, err := updater.Run()

	// ******************************** ASSERT ********************************
	assert.Error(t, err, "Fetch was expected to return non-nil ReadCloser")
	assert.False(t, updated, "No update should have occured")
}

func TestUpdaterNoUpdateOccursIfAtLatestVersion(t *testing.T) {
	// ******************************* ARRANGE ********************************
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Handle http requests
	mr := mocks.NewMockRequester(ctrl)
	mr.EXPECT().
		Fetch("http://updates.yourdomain.com/myapp/windows-amd64.json").
		Return(ioutil.NopCloser(strings.NewReader(createTestVersionInfo("1.2", "zv2MYEWhvfOnmwKXO6lDm2ATh0ehbnyixA8562FDtAE="))), nil)

	platformResolver := NewSpecificPlatformResolver("windows", "amd64")

	updater := createUpdater("1.2", mr, nil, platformResolver)

	// ********************************* ACT **********************************
	updated, err := updater.Run()

	// ******************************** ASSERT ********************************
	assert.NoError(t, err)
	assert.False(t, updated, "No update should have occured")
}

func TestUpdaterPatchesBinariesWhenVersionsDiffer(t *testing.T) {
	// ******************************* ARRANGE ********************************
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Handle http requests
	mr := mocks.NewMockRequester(ctrl) // http://updates.yourdomain.com/myapp/1.2/1.3/windows-amd64
	mr.EXPECT().
		Fetch("http://updates.yourdomain.com/myapp/windows-amd64.json").
		Return(ioutil.NopCloser(strings.NewReader(createTestVersionInfo("1.1", "GlGgXKYygTyZyUsCvyJX5QiK+ntuia8bVdX4iwWr/Dc="))), nil)

	mr.EXPECT().
		Fetch("http://updates.yourdomain.com/myapp/1.0/1.1/windows-amd64").
		Return(os.Open("../golden_data/public/1.0/1.1/windows-amd64"))

	f := cpFileAsTemp("../golden_data/main-before.exe", "toUpdate")

	// clean up temp files
	defer os.Remove(f.Name())

	updateResolver := NewSpecificFileUpdatableResolver(f.Name())

	platformResolver := NewSpecificPlatformResolver("windows", "amd64")

	updater := createUpdater("1.0", mr, updateResolver, platformResolver)

	// ********************************* ACT **********************************
	updated, err := updater.Run()

	// ******************************** ASSERT ********************************
	assert.NoError(t, err)
	assert.True(t, updated, "The patch should have been applied")

	goldenBytes, err := ioutil.ReadFile("../golden_data/main-new.exe")
	assert.NoError(t, err)
	whatWasPatched, err := ioutil.ReadFile(f.Name())
	assert.NoError(t, err)

	assert.True(t, bytes.Equal(goldenBytes, whatWasPatched), "The resulting binary after patch doesn't match what it was originally diffed against")
}

func TestUpdaterWholeReplacesBinariesWhenVersionsDifferAndPatchingFails(t *testing.T) {
	// ******************************* ARRANGE ********************************
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Handle http requests
	mr := mocks.NewMockRequester(ctrl) // http://updates.yourdomain.com/myapp/1.2/1.3/windows-amd64
	mr.EXPECT().
		Fetch("http://updates.yourdomain.com/myapp/windows-amd64.json").
		Return(ioutil.NopCloser(strings.NewReader(createTestVersionInfo("1.1", "GlGgXKYygTyZyUsCvyJX5QiK+ntuia8bVdX4iwWr/Dc="))), nil)

	// Force patching to fail
	mr.EXPECT().
		Fetch("http://updates.yourdomain.com/myapp/1.0/1.1/windows-amd64").
		Return(nil, errors.New("Fake Error"))

	mr.EXPECT().
		Fetch("http://updates.yourdomain.com/myapp/1.1/windows-amd64.gz").
		Return(os.Open("../golden_data/public/1.1/windows-amd64.gz"))

	f := cpFileAsTemp("../golden_data/main-before.exe", "toUpdate")

	// clean up temp files
	defer os.Remove(f.Name())

	updateResolver := NewSpecificFileUpdatableResolver(f.Name())

	platformResolver := NewSpecificPlatformResolver("windows", "amd64")

	updater := createUpdater("1.0", mr, updateResolver, platformResolver)

	// ********************************* ACT **********************************
	updated, err := updater.Run()

	// ******************************** ASSERT ********************************
	assert.NoError(t, err)
	assert.True(t, updated, "The patch should have been applied")

	goldenBytes, err := ioutil.ReadFile("../golden_data/main-new.exe")
	assert.NoError(t, err)
	whatWasPatched, err := ioutil.ReadFile(f.Name())
	assert.NoError(t, err)

	assert.True(t, bytes.Equal(goldenBytes, whatWasPatched), "The resulting binary after patch doesn't match what it was originally diffed against")
}

func TestUpdaterReturnsFailureWhenPatchAndWholeSwapFails(t *testing.T) {
	// ******************************* ARRANGE ********************************
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Handle http requests
	mr := mocks.NewMockRequester(ctrl) // http://updates.yourdomain.com/myapp/1.2/1.3/windows-amd64
	mr.EXPECT().
		Fetch("http://updates.yourdomain.com/myapp/windows-amd64.json").
		Return(ioutil.NopCloser(strings.NewReader(createTestVersionInfo("1.1", "GlGgXKYygTyZyUsCvyJX5QiK+ntuia8bVdX4iwWr/Dc="))), nil)

	// Force patching to fail
	mr.EXPECT().
		Fetch("http://updates.yourdomain.com/myapp/1.0/1.1/windows-amd64").
		Return(nil, errors.New("Fake Error"))

	mr.EXPECT().
		Fetch("http://updates.yourdomain.com/myapp/1.1/windows-amd64.gz").
		Return(nil, errors.New("Fake Error"))

	f := cpFileAsTemp("../golden_data/main-before.exe", "toUpdate")

	// clean up temp files
	defer os.Remove(f.Name())

	updateResolver := NewSpecificFileUpdatableResolver(f.Name())

	platformResolver := NewSpecificPlatformResolver("windows", "amd64")

	updater := createUpdater("1.0", mr, updateResolver, platformResolver)

	// ********************************* ACT **********************************
	updated, err := updater.Run()

	// ******************************** ASSERT ********************************
	assert.Error(t, err)
	assert.False(t, updated, "The patch should have been applied")

	goldenBytes, err := ioutil.ReadFile("../golden_data/main-new.exe")
	assert.NoError(t, err)
	whatWasPatched, err := ioutil.ReadFile(f.Name())
	assert.NoError(t, err)

	assert.False(t, bytes.Equal(goldenBytes, whatWasPatched), "The resulting binary after patch doesn't match what it was originally diffed against")
}

func TestUpdaterShowsNoUpdateIsAvailableWhenVersionsMatch(t *testing.T) {
	// ******************************* ARRANGE ********************************
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Handle http requests
	mr := mocks.NewMockRequester(ctrl) // http://updates.yourdomain.com/myapp/1.2/1.3/windows-amd64
	mr.EXPECT().
		Fetch("http://updates.yourdomain.com/myapp/windows-amd64.json").
		Return(ioutil.NopCloser(strings.NewReader(createTestVersionInfo("1.1", "GlGgXKYygTyZyUsCvyJX5QiK+ntuia8bVdX4iwWr/Dc="))), nil)

	platformResolver := NewSpecificPlatformResolver("windows", "amd64")

	updater := createUpdater("1.1", mr, nil, platformResolver)

	// ********************************* ACT **********************************
	updateAvailable, err := updater.UpdateAvailable()

	// ******************************** ASSERT ********************************
	assert.NoError(t, err)
	assert.False(t, updateAvailable)
}

func createUpdater(
	version string,
	mr Requester,
	updateResolver UpdatableResolver,
	platformResolver PlatformResolver,
) Updater {
	return NewUpdater(version, "http://updates.yourdomain.com/", "myapp").
		SetUpdatableResolver(updateResolver).
		SetRequester(mr).
		SetPlatformResolver(platformResolver)
}
