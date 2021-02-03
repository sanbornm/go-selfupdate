package selfupdate

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sanbornm/go-selfupdate/selfupdate/mocks"
)

var testHash = sha256.New()

func TestUpdaterFetchMustReturnNonNilReaderCloser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mr := mocks.NewMockRequester(ctrl)
	mr.EXPECT().Fetch("http://api.updates.yourdomain.com/myapp/darwin-amd64.json").Return(nil, nil).Times(1)

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
	mr.EXPECT().Fetch("http://api.updates.yourdomain.com/myapp/darwin-amd64.json").Return(newTestReaderCloser("{}"), nil).Times(1)
	mr.EXPECT().Fetch(gomock.Any()).Times(0)

	updater := createUpdater(mr)
	updater.CheckTime = 24
	updater.RandomizeTime = 24

	err := updater.BackgroundRun()
	if err != nil {
		t.Errorf("Error occurred: %#v", err)
	}
}

func TestUpdaterWithNewVersionAndMissingBinaryReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mr := mocks.NewMockRequester(ctrl)
	h := sha256.New()
	h.Write([]byte("Test"))
	c := Info{Version: "1.3", Sha256: h.Sum(nil)}

	b, err := json.MarshalIndent(c, "", "    ")
	mr.EXPECT().Fetch("http://api.updates.yourdomain.com/myapp/darwin-amd64.json").Return(newTestReaderCloser(string(b)), nil).Times(1)
	mr.EXPECT().Fetch("http://diff.updates.yourdomain.com/myapp/1.2/1.3/darwin-amd64").Return(newTestReaderCloser("{}"), fmt.Errorf("Bad status code on diff: 404")).Times(1)
	mr.EXPECT().Fetch("http://bin.updates.yourdownmain.com/myapp/1.3/darwin-amd64.gz").Return(newTestReaderCloser("{}"), fmt.Errorf("Bad status code on binary: 404")).Times(1)
	mr.EXPECT().Fetch(gomock.Any()).Times(0)

	updater := createUpdater(mr)
	updater.ForceCheck = true

	err = updater.BackgroundRun()
	if err != nil {
		equals(t, "Bad status code on binary: 404", err.Error())
	} else {
		t.Log("Expected an error")
		t.Fail()
	}
}

func TestUpdaterCheckTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mr := mocks.NewMockRequester(ctrl)
	mr.EXPECT().Fetch("http://api.updates.yourdomain.com/myapp/darwin-amd64.json").Return(newTestReaderCloser("{}"), nil).Times(4)
	mr.EXPECT().Fetch(gomock.Any()).Times(0) // no additional calls

	// Run test with various time
	runTestTimeChecks(t, mr, 0, 0, false)
	runTestTimeChecks(t, mr, 0, 5, true)
	runTestTimeChecks(t, mr, 1, 0, true)
	runTestTimeChecks(t, mr, 100, 100, true)
}

// Helper function to run check time tests
func runTestTimeChecks(t *testing.T, mr Requester, checkTime int, randomizeTime int, expectUpdate bool) {
	updater := createUpdater(mr)
	updater.ClearUpdateState()
	updater.CheckTime = checkTime
	updater.RandomizeTime = randomizeTime

	updater.BackgroundRun()

	if updater.WantUpdate() == expectUpdate {
		t.Errorf("WantUpdate returned %v; want %v", updater.WantUpdate(), expectUpdate)
	}

	maxHrs := time.Duration(updater.CheckTime+updater.RandomizeTime) * time.Hour
	maxTime := time.Now().Add(maxHrs)

	if !updater.NextUpdate().Before(maxTime) {
		t.Errorf("NextUpdate should less than %s hrs (CheckTime + RandomizeTime) from now; now %s; next update %s", maxHrs, time.Now(), updater.NextUpdate())
	}

	if maxHrs > 0 && !updater.NextUpdate().After(time.Now()) {
		t.Errorf("NextUpdate should be after now")
	}
}

func TestUpdaterWithEmptyPayloadNoErrorNoUpdateEscapedPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mr := mocks.NewMockRequester(ctrl)
	mr.EXPECT().Fetch("http://api.updates.yourdomain.com/myapp%2Bfoo/darwin-amd64.json").Return(newTestReaderCloser("{}"), nil).Times(1)
	mr.EXPECT().Fetch(gomock.Any()).Times(0) // no additional calls

	updater := createUpdaterWithEscapedCharacters(mr)
	updater.ForceCheck = true

	err := updater.BackgroundRun()
	if err != nil {
		t.Errorf("Error occurred: %#v", err)
	}
}

func createUpdater(mr Requester) *Updater {
	return &Updater{
		CurrentVersion: "1.2",
		ApiURL:         "http://api.updates.yourdomain.com/",
		BinURL:         "http://bin.updates.yourdownmain.com/",
		DiffURL:        "http://diff.updates.yourdomain.com/",
		Dir:            "update/",
		CmdName:        "myapp", // app name
		Requester:      mr,
	}
}

func createUpdaterWithEscapedCharacters(mr Requester) *Updater {
	return &Updater{
		CurrentVersion: "1.2+foobar",
		ApiURL:         "http://api.updates.yourdomain.com/",
		BinURL:         "http://bin.updates.yourdownmain.com/",
		DiffURL:        "http://diff.updates.yourdomain.com/",
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
