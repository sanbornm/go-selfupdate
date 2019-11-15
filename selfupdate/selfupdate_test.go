package selfupdate

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
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

func createTestVersionInfo(version string) string {
	return fmt.Sprintf(`{
		"Version": "%s",
		"Sha256": "qC5NwfTK+Y0y5a9/GtbAIJjwT5RtMviGzfESl+btu68="
	}`, version)
}

func TestUpdaterFetchMustReturnNonNilReaderCloser(t *testing.T) {
	// ******************************* ARRANGE ********************************
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mr := mocks.NewMockRequester(ctrl)
	mr.EXPECT().
		Fetch("http://updates.yourdomain.com/myapp/"+runtime.GOOS+"-"+runtime.GOARCH+".json").
		Return(nil, nil)

	updater := createUpdater("", mr, nil)

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
		Fetch("http://updates.yourdomain.com/myapp/"+runtime.GOOS+"-"+runtime.GOARCH+".json").
		Return(newTestReaderCloser(createTestVersionInfo("1.2")), nil)

	updater := createUpdater("1.2", mr, nil)

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
		Fetch("http://updates.yourdomain.com/myapp/"+runtime.GOOS+"-"+runtime.GOARCH+".json").
		Return(newTestReaderCloser(createTestVersionInfo("1.3")), nil)

	mr.EXPECT().
		Fetch("http://updates.yourdomain.com/myapp/1.2/1.3/" + runtime.GOOS + "-" + runtime.GOARCH).
		Return()

	f := cpFileAsTemp("golden_data/main-before.exe", "toUpdate")
	defer os.Remove(f.Name()) // clean up

	resolver := mocks.NewMockUpdatableResolver(ctrl)
	resolver.EXPECT().
		Resolve().
		Return(f.Name(), nil)

	updater := createUpdater("1.1", mr, resolver)

	// ********************************* ACT **********************************
	updated, err := updater.Run()

	// ******************************** ASSERT ********************************
	assert.NoError(t, err)
	assert.False(t, updated, "No update should have occured")
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

func createUpdater(version string, mr Requester, resolver UpdatableResolver) *Updater {
	return &Updater{
		currentVersion:     version,
		apiURL:             "http://updates.yourdomain.com/",
		binURL:             "http://updates.yourdomain.com/",
		diffURL:            "http://updates.yourdomain.com/",
		cacheDir:           "update",
		cmdName:            "myapp",
		requester:          mr,
		updateableResolver: resolver,
	}
}

func createUpdaterWithEscapedCharacters(mr Requester, resolver UpdatableResolver) *Updater {
	return &Updater{
		currentVersion:     "1.2+foobar",
		apiURL:             "http://updates.yourdomain.com/",
		binURL:             "http://updates.yourdomain.com/",
		diffURL:            "http://updates.yourdomain.com/",
		cacheDir:           "update",
		cmdName:            "myapp+foo",
		requester:          mr,
		updateableResolver: resolver,
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

// // mockRequester used for some mock testing to ensure the requester contract
// // works as specified.
// type mockRequester struct {
// 	currentIndex int
// 	fetches      []func(string) (io.ReadCloser, error)
// }

// func (mr *mockRequester) handleRequest(requestHandler func(string) (io.ReadCloser, error)) {
// 	if mr.fetches == nil {
// 		mr.fetches = []func(string) (io.ReadCloser, error){}
// 	}
// 	mr.fetches = append(mr.fetches, requestHandler)
// }

// func (mr *mockRequester) Fetch(url string) (io.ReadCloser, error) {
// 	if len(mr.fetches) <= mr.currentIndex {
// 		return nil, fmt.Errorf("No for currentIndex %d to mock", mr.currentIndex)
// 	}
// 	current := mr.fetches[mr.currentIndex]
// 	mr.currentIndex++

// 	return current(url)
// }
