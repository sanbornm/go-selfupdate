package selfupdate

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"runtime"
	"testing"
)

var testHash = sha256.New()

func getPlatformName() string {
	return runtime.GOOS + "-" + runtime.GOARCH
}

// mockRequester used for some mock testing to ensure the requester contract
// works as specified.
type mockRequester struct {
	currentIndex int
	fetches      []func(string) (io.ReadCloser, error)
}

func (mr *mockRequester) handleRequest(requestHandler func(string) (io.ReadCloser, error)) {
	if mr.fetches == nil {
		mr.fetches = []func(string) (io.ReadCloser, error){}
	}
	mr.fetches = append(mr.fetches, requestHandler)
}

func (mr *mockRequester) Fetch(url string) (io.ReadCloser, error) {
	if len(mr.fetches) <= mr.currentIndex {
		return nil, fmt.Errorf("No for currentIndex %d to mock", mr.currentIndex)
	}
	current := mr.fetches[mr.currentIndex]
	mr.currentIndex++

	return current(url)
}

func TestUpdaterFetchMustReturnNonNilReaderCloser(t *testing.T) {
	mr := &mockRequester{}
	mr.handleRequest(
		func(url string) (io.ReadCloser, error) {
			return nil, nil
		})
	updater := createUpdater(mr, "1.2")
	err := updater.BackgroundRun()
	if err != nil {
		equals(t, "Fetch was expected to return non-nil ReadCloser", err.Error())
	} else {
		t.Log("Expected an error")
		t.Fail()
	}
}

func TestUpdaterWithEmptyPayloadNoErrorNoUpdate(t *testing.T) {
	mr := &mockRequester{}
	mr.handleRequest(
		func(url string) (io.ReadCloser, error) {
			equals(t, fmt.Sprintf("http://updates.yourdomain.com/myapp/%s.json", getPlatformName()), url)
			return newTestReaderCloser("{}"), nil
		})
	updater := createUpdater(mr, "1.2")

	err := updater.BackgroundRun()
	if err != nil {
		t.Errorf("Error occurred: %#v", err)
	}
}

func TestUpdaterWithEmptyPayloadNoErrorNoUpdateEscapedPath(t *testing.T) {
	mr := &mockRequester{}
	mr.handleRequest(
		func(url string) (io.ReadCloser, error) {
			equals(t, fmt.Sprintf("http://updates.yourdomain.com/myapp%%2Bfoo/%s.json", getPlatformName()), url)
			return newTestReaderCloser("{}"), nil
		})
	updater := createUpdaterWithEscapedCharacters(mr)

	err := updater.BackgroundRun()
	if err != nil {
		t.Errorf("Error occurred: %#v", err)
	}
}

func createUpdater(mr *mockRequester, version string) *Updater {
	return &Updater{
		CurrentVersion: version,
		ApiURL:         "http://updates.yourdomain.com/",
		BinURL:         "http://updates.yourdownmain.com/",
		DiffURL:        "http://updates.yourdomain.com/",
		Dir:            "update/",
		CmdName:        "myapp", // app name
		Requester:      mr,
		ForceCheck:     true,
	}
}

func createUpdaterWithEscapedCharacters(mr *mockRequester) *Updater {
	return &Updater{
		CurrentVersion: "1.2+foobar",
		ApiURL:         "http://updates.yourdomain.com/",
		BinURL:         "http://updates.yourdownmain.com/",
		DiffURL:        "http://updates.yourdomain.com/",
		Dir:            "update/",
		CmdName:        "myapp+foo", // app name
		Requester:      mr,
		ForceCheck:     true,
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
