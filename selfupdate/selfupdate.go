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
	"log"
	"net/url"
	"os"
	"runtime"

	"github.com/kr/binarydist"
	"gopkg.in/inconshreveable/go-update.v0"
)

const (
	ourPlatform = runtime.GOOS + "-" + runtime.GOARCH
)

// ErrHashMismatch returned whenever the new file's hash is mismatched after patch, indicating patch was unsuccesful.
var ErrHashMismatch = errors.New("new file hash mismatch after patch")

var up = update.New()

// Updater is the configuration and runtime data for doing an update.
//
// Note that ApiURL, BinURL and DiffURL should have the same value if all files are available at the same location.
//
// Example:
//
// ```golang
//  updater := &selfupdate.Updater{
//  	CurrentVersion: version,
//  	ApiURL:         "http://updates.yourdomain.com/",
//  	BinURL:         "http://updates.yourdownmain.com/",
//  	DiffURL:        "http://updates.yourdomain.com/",
//  	CacheDir:            "update",
//  	CmdName:        "myapp", // app name
//  }
//  if updater != nil {
//  	go updater.BackgroundRun()
//  }
// ```
type Updater struct {
	currentVersion     string            // Currently running version.
	apiURL             string            // Base URL for API requests (json files).
	cmdName            string            // Command name is appended to the ApiURL like http://apiurl/CmdName/. This represents one binary.
	binURL             string            // Base URL for full binary downloads.
	diffURL            string            // Base URL for diff downloads.
	cacheDir           string            // Directory to store selfupdate state.
	forceCheck         bool              // Check for update regardless of cktime timestamp
	requester          Requester         // Optional parameter to override existing http request handler
	updateableResolver UpdatableResolver // Finds the thing that needs to be updated
}

// NewUpdater creates a new updater
func NewUpdater(currentVersion string, updateDataURL string) Updater {
	return Updater{
		currentVersion:     currentVersion,
		apiURL:             updateDataURL,
		binURL:             updateDataURL,
		diffURL:            updateDataURL,
		cacheDir:           "update",
		forceCheck:         false,
		requester:          HTTPRequester{},
		updateableResolver: CurrentExeUpdatableResolver{},
	}
}

// Run attempts to grab the latest version information and then applies
// the new patch if their is an update.
func (u *Updater) Run() error {

	path, err := getExecRelativeDir(u.cacheDir)

	if err != nil {
		return err
	}

	// Create folder for storing updates if it doesn't exist
	if err := os.MkdirAll(path, 0777); err != nil {
		return err
	}

	if err := up.CanUpdate(); err != nil {
		return err
	}

	return u.update()
}

func (u *Updater) update() error {

	old, err := u.updateableResolver.Resolve()

	info, err := u.fetchInfo()
	if err != nil {
		return err
	}

	// No need to update
	if info.Version == u.currentVersion {
		return nil
	}

	bin, err := u.getExeWithPatchForVersion(old, info)
	if err != nil {
		if err == ErrHashMismatch {
			log.Println("update: hash mismatch from patched binary")
		} else {
			if u.diffURL != "" {
				log.Println("update: patching binary,", err)
			}
		}

		bin, err = u.getEntireBinaryForVersion(info)
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

	return err
}

func (u *Updater) fetchInfo() (versionInfo, error) {
	r, err := u.fetch(fmt.Sprintf("%s%s/%s.json", u.apiURL, url.QueryEscape(u.cmdName), url.QueryEscape(ourPlatform)))
	if err != nil {
		return versionInfo{}, err
	}
	defer r.Close()

	info := versionInfo{}
	err = json.NewDecoder(r).Decode(&info)
	if err != nil {
		return versionInfo{}, err
	}
	if len(info.Sha256) != sha256.Size {
		return versionInfo{}, errors.New("bad cmd hash in info")
	}
	return info, nil
}

// getExeWithPatchForVersion retrives the patch for the current  version and
// applies it to the current executable passed in, and returns the results
func (u Updater) getExeWithPatchForVersion(old io.Reader, info versionInfo) ([]byte, error) {
	r, err := u.fetch(u.diffURL + url.QueryEscape(u.cmdName) + "/" + url.QueryEscape(u.currentVersion) + "/" + url.QueryEscape(info.Version) + "/" + url.QueryEscape(ourPlatform))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var buf bytes.Buffer
	err = binarydist.Patch(old, &buf, r)
	bin := buf.Bytes()

	if !verifySha(bin, info.Sha256) {
		return nil, ErrHashMismatch
	}

	return bin, nil
}

func (u Updater) getEntireBinaryForVersion(info versionInfo) ([]byte, error) {
	r, err := u.fetch(fmt.Sprintf("%s%s/%s/%s.gz", u.binURL, url.QueryEscape(u.cmdName), url.QueryEscape(info.Version), url.QueryEscape(ourPlatform)))
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

	bin := buf.Bytes()

	if !verifySha(bin, info.Sha256) {
		return nil, ErrHashMismatch
	}

	return bin, nil
}

func (u Updater) fetch(url string) (io.ReadCloser, error) {
	if u.requester == nil {
		return nil, errors.New("unable to fetch information with nil requester")
	}

	readCloser, err := u.requester.Fetch(url)
	if err != nil {
		return nil, err
	}

	if readCloser == nil {
		return nil, fmt.Errorf("Fetch was expected to return non-nil ReadCloser")
	}

	return readCloser, nil
}
