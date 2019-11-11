package selfupdate

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"testing"

	"github.com/EliCDavis/go-selfupdate/selfupdate/mocks"

	"github.com/golang/mock/gomock"
)

var testHash = sha256.New()

func TestUpdaterFetchMustReturnNonNilReaderCloser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mr := mocks.NewMockRequester(ctrl)
	mr.EXPECT().
		Fetch("http://updates.yourdomain.com/myapp/darwin-amd64.json").
		Return(nil, nil)

	updater := createUpdater(mr)
	err := updater.BackgroundRun()
	if err != nil {
		equals(t, "Fetch was expected to return non-nil ReadCloser", err.Error())
	} else {
		t.Log("Expected an error")
		t.Fail()
	}
}

func TestUpdaterWithEmptyPayloadNoErrorNoUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mr := mocks.NewMockRequester(ctrl)
	mr.EXPECT().
		Fetch("http://updates.yourdomain.com/myapp/darwin-amd64.json").
		Return(newTestReaderCloser("{}"), nil)

	updater := createUpdater(mr)

	err := updater.BackgroundRun()
	if err != nil {
		t.Errorf("Error occurred: %#v", err)
	}
}

func TestUpdaterWithEmptyPayloadNoErrorNoUpdateEscapedPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mr := mocks.NewMockRequester(ctrl)
	mr.EXPECT().
		Fetch("http://updates.yourdomain.com/myapp%2Bfoo/darwin-amd64.json").
		Return(newTestReaderCloser("{}"), nil)

	// mr.handleRequest(
	// 	func(url string) (io.ReadCloser, error) {
	// 		equals(t, "http://updates.yourdomain.com/myapp%2Bfoo/darwin-amd64.json", url)
	// 		return newTestReaderCloser("{}"), nil
	// 	})
	updater := createUpdaterWithEscapedCharacters(mr)

	err := updater.BackgroundRun()
	if err != nil {
		t.Errorf("Error occurred: %#v", err)
	}
}

func createUpdater(mr Requester) *Updater {
	return &Updater{
		CurrentVersion: "1.2",
		ApiURL:         "http://updates.yourdomain.com/",
		BinURL:         "http://updates.yourdownmain.com/",
		DiffURL:        "http://updates.yourdomain.com/",
		Dir:            "update/",
		CmdName:        "myapp", // app name
		Requester:      mr,
	}
}

func createUpdaterWithEscapedCharacters(mr Requester) *Updater {
	return &Updater{
		CurrentVersion: "1.2+foobar",
		ApiURL:         "http://updates.yourdomain.com/",
		BinURL:         "http://updates.yourdownmain.com/",
		DiffURL:        "http://updates.yourdomain.com/",
		Dir:            "update/",
		CmdName:        "myapp+foo", // app name
		Requester:      mr,
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
