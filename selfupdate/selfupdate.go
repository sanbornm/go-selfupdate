// Package selfupdate update protocol:
//
//   GET hk.heroku.com/hk/linux-amd64.json
//
//   200 ok
//   {
//       "Version": "2",
//       "Sha256": "..." // base64
//   }
//
// then
//
//   GET hkpatch.s3.amazonaws.com/hk/1/2/linux-amd64
//
//   200 ok
//   [bsdiff data]
//
// or
//
//   GET hkdist.s3.amazonaws.com/hk/2/linux-amd64.gz
//
//   200 ok
//   [gzipped executable data]
//
//
package selfupdate

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/kardianos/osext"
	"github.com/kr/binarydist"
	"gopkg.in/inconshreveable/go-update.v0"
)

const (
	upcktimePath = "cktime"
	ourPlatform  = runtime.GOOS + "-" + runtime.GOARCH
)

// ErrHashMismatch returned whenever the new file's hash is mismatched after patch, indicating patch was unsuccesful.
var ErrHashMismatch = errors.New("new file hash mismatch after patch")

var up = update.New()
var defaultHTTPRequester = HTTPRequester{}

// Updater is the configuration and runtime data for doing an update.
//
// Note that ApiURL, BinURL and DiffURL should have the same value if all files are available at the same location.
//
// Example:
//
//  updater := &selfupdate.Updater{
//  	CurrentVersion: version,
//  	ApiURL:         "http://updates.yourdomain.com/",
//  	BinURL:         "http://updates.yourdownmain.com/",
//  	DiffURL:        "http://updates.yourdomain.com/",
//  	Dir:            "update/",
//  	CmdName:        "myapp", // app name
//  }
//  if updater != nil {
//  	go updater.BackgroundRun()
//  }
type Updater struct {
	CurrentVersion string    // Currently running version.
	ApiURL         string    // Base URL for API requests (json files).
	CmdName        string    // Command name is appended to the ApiURL like http://apiurl/CmdName/. This represents one binary.
	BinURL         string    // Base URL for full binary downloads.
	DiffURL        string    // Base URL for diff downloads.
	Dir            string    // Directory to store selfupdate state.
	ForceCheck     bool      // Check for update regardless of cktime timestamp
	Requester      Requester //Optional parameter to override existing http request handler
	Info           struct {
		Version string
		Sha256  []byte
	}
}

func (u *Updater) getExecRelativeDir(dir string) (string, error) {
	filename, err := osext.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(filename), dir), nil
}

// BackgroundRun first checks to see if it's time to check for updates. If it
// is time to check for updates, it does that
func (u *Updater) BackgroundRun() error {

	path, err := u.getExecRelativeDir(u.Dir)

	if err != nil {
		return err
	}

	// Create folder for storing updates if it doesn't exist
	if err := os.MkdirAll(path, 0777); err != nil {
		return err
	}

	wantUpdate, err := u.wantUpdate()

	if err != nil {
		return err
	}

	if wantUpdate {
		if err := up.CanUpdate(); err != nil {
			return err
		}

		return u.update()
	}
	return nil
}

func (u *Updater) wantUpdate() (bool, error) {

	if u.CurrentVersion == "dev" {
		return false, nil
	}

	path, err := u.getExecRelativeDir(u.Dir + upcktimePath)
	if err != nil {
		return false, err
	}

	wait := 24 * time.Hour

	if u.ForceCheck {
		return true, writeTime(path, time.Now().Add(wait))
	}

	timeToCheck, err := readTime(path)
	if err != nil {
		return false, err
	}

	if timeToCheck.After(time.Now()) {
		return false, nil
	}

	return true, writeTime(path, time.Now().Add(wait))
}

func (u *Updater) update() error {
	path, err := osext.Executable()
	if err != nil {
		return err
	}
	old, err := os.Open(path)
	if err != nil {
		return err
	}
	defer old.Close()

	err = u.fetchInfo()
	if err != nil {
		return err
	}
	if u.Info.Version == u.CurrentVersion {
		return nil
	}
	bin, err := u.fetchAndVerifyPatch(old)
	if err != nil {
		if err == ErrHashMismatch {
			log.Println("update: hash mismatch from patched binary")
		} else {
			if u.DiffURL != "" {
				log.Println("update: patching binary,", err)
			}
		}

		bin, err = u.fetchAndVerifyFullBin()
		if err != nil {
			if err == ErrHashMismatch {
				log.Println("update: hash mismatch from full binary")
			} else {
				log.Println("update: fetching full binary,", err)
			}
			return err
		}
	}

	// close the old binary before installing because on windows
	// it can't be renamed if a handle to the file is still open
	old.Close()

	err, errRecover := up.FromStream(bytes.NewBuffer(bin))
	if errRecover != nil {
		return fmt.Errorf("update and recovery errors: %q %q", err, errRecover)
	}
	if err != nil {
		return err
	}
	return nil
}

func (u *Updater) fetchInfo() error {
	r, err := u.fetch(fmt.Sprintf("%s%s/%s.json", u.ApiURL, url.QueryEscape(u.CmdName), url.QueryEscape(ourPlatform)))
	if err != nil {
		return err
	}
	defer r.Close()
	err = json.NewDecoder(r).Decode(&u.Info)
	if err != nil {
		return err
	}
	if len(u.Info.Sha256) != sha256.Size {
		return errors.New("bad cmd hash in info")
	}
	return nil
}

func (u *Updater) fetchAndVerifyPatch(old io.Reader) ([]byte, error) {
	bin, err := u.fetchAndApplyPatch(old)
	if err != nil {
		return nil, err
	}
	if !verifySha(bin, u.Info.Sha256) {
		return nil, ErrHashMismatch
	}
	return bin, nil
}

func (u *Updater) fetchAndApplyPatch(old io.Reader) ([]byte, error) {
	r, err := u.fetch(u.DiffURL + url.QueryEscape(u.CmdName) + "/" + url.QueryEscape(u.CurrentVersion) + "/" + url.QueryEscape(u.Info.Version) + "/" + url.QueryEscape(ourPlatform))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var buf bytes.Buffer
	err = binarydist.Patch(old, &buf, r)
	return buf.Bytes(), err
}

func (u *Updater) fetchAndVerifyFullBin() ([]byte, error) {
	bin, err := u.fetchBin()
	if err != nil {
		return nil, err
	}
	verified := verifySha(bin, u.Info.Sha256)
	if !verified {
		return nil, ErrHashMismatch
	}
	return bin, nil
}

func (u *Updater) fetchBin() ([]byte, error) {
	r, err := u.fetch(fmt.Sprintf("%s%s/%s/%s.gz", u.BinURL, url.QueryEscape(u.CmdName), url.QueryEscape(u.Info.Version), url.QueryEscape(ourPlatform)))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	buf := new(bytes.Buffer)
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(buf, gz); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (u *Updater) fetch(url string) (io.ReadCloser, error) {
	if u.Requester == nil {
		return defaultHTTPRequester.Fetch(url)
	}

	readCloser, err := u.Requester.Fetch(url)
	if err != nil {
		return nil, err
	}

	if readCloser == nil {
		return nil, fmt.Errorf("Fetch was expected to return non-nil ReadCloser")
	}

	return readCloser, nil
}

func readTime(path string) (time.Time, error) {
	p, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return time.Time{}, err
	}
	if err != nil {
		return time.Time{}, err
	}
	t, err := time.Parse(time.RFC3339, string(p))
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func verifySha(bin []byte, sha []byte) bool {
	h := sha256.New()
	h.Write(bin)
	return bytes.Equal(h.Sum(nil), sha)
}

func writeTime(path string, t time.Time) error {
	return ioutil.WriteFile(path, []byte(t.Format(time.RFC3339)), 0644)
}
