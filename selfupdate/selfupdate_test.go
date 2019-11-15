package selfupdate

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/EliCDavis/go-selfupdate/selfupdate/mocks"

	"github.com/golang/mock/gomock"
)

var testHash = sha256.New()

func cpFileAsTemp(fileToCopy, newFlieName string) *os.File {
	content := []byte("temporary file's content")
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
		Return(newTestReaderCloser(createTestVersionInfo("1.2", "zv2MYEWhvfOnmwKXO6lDm2ATh0ehbnyixA8562FDtAE=")), nil)

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
		Return(newTestReaderCloser(createTestVersionInfo("1.1", "GlGgXKYygTyZyUsCvyJX5QiK+ntuia8bVdX4iwWr/Dc=")), nil)

	mr.EXPECT().
		Fetch("http://updates.yourdomain.com/myapp/1.0/1.1/windows-amd64").
		Return(os.Open("golden_data/public/1.0/1.1/windows-amd64"))

	f := cpFileAsTemp("golden_data/main-before.exe", "toUpdate")

	// clean up temp files
	defer os.Remove(f.Name())

	updateResolver := mocks.NewMockUpdatableResolver(ctrl)
	updateResolver.EXPECT().
		Resolve().
		Return(f.Name(), nil)

	platformResolver := NewSpecificPlatformResolver("windows", "amd64")

	updater := createUpdater("1.0", mr, updateResolver, platformResolver)

	// ********************************* ACT **********************************
	updated, err := updater.Run()

	// ******************************** ASSERT ********************************
	assert.NoError(t, err)
	assert.True(t, updated, "The patch should have been applied")
}

// func TestUpdaterWithEmptyPayloadNoErrorNoUpdateEscapedPath(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()
// 	mr := mocks.NewMockRequester(ctrl)
// 	mr.EXPECT().
// 		Fetch("http://updates.yourdomain.com/myapp%2Bfoo/"+runtime.GOOS+"-"+runtime.GOARCH+".json").
// 		Return(newTestReaderCloser(exampleVersionInfo), nil)

// 	updater := createUpdaterWithEscapedCharacters(mr)

// 	err := updater.Run()
// 	if err != nil {
// 		t.Errorf("Error occurred: %#v", err)
// 	}
// }

func createUpdater(
	version string,
	mr Requester,
	updateResolver UpdatableResolver,
	platformResolver PlatformResolver,
) *Updater {
	return &Updater{
		currentVersion:     version,
		apiURL:             "http://updates.yourdomain.com/",
		binURL:             "http://updates.yourdomain.com/",
		diffURL:            "http://updates.yourdomain.com/",
		cacheDir:           "update",
		cmdName:            "myapp",
		requester:          mr,
		platformResolver:   platformResolver,
		updateableResolver: updateResolver,
	}
}

func createUpdaterWithEscapedCharacters(
	mr Requester,
	updateResolver UpdatableResolver,
	platformResolver PlatformResolver,
) *Updater {
	return &Updater{
		currentVersion:     "1.2+foobar",
		apiURL:             "http://updates.yourdomain.com/",
		binURL:             "http://updates.yourdomain.com/",
		diffURL:            "http://updates.yourdomain.com/",
		cacheDir:           "update",
		cmdName:            "myapp+foo",
		requester:          mr,
		updateableResolver: updateResolver,
		platformResolver:   platformResolver,
	}
}

func equals(t *testing.T, expected, actual interface{}) {
	if expected != actual {
		t.Log(fmt.Sprintf("Expected: %#v %#v\n", expected, actual))
		t.Fail()
	}
}

type testReadCloser struct {
	buffer *bytes.Buffer
}

func newTestReaderCloser(payload string) io.ReadCloser {
	return &testReadCloser{buffer: bytes.NewBufferString(payload)}
}

func (trc *testReadCloser) Read(p []byte) (n int, err error) {
	return trc.buffer.Read(p)
}

func (trc *testReadCloser) Close() error {
	return nil
}
