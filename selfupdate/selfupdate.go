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

	"github.com/kr/binarydist"
	"gopkg.in/inconshreveable/go-update.v0"
)

// ErrHashMismatch returned whenever the new file's hash is mismatched after patch, indicating patch was unsuccesful.
var ErrHashMismatch = errors.New("new file hash mismatch after patch")

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
	requester          Requester         // Optional parameter to override existing http request handler
	updateableResolver UpdatableResolver // Finds the thing that needs to be updated
	platformResolver   PlatformResolver  // Figures out what platform we need to update for
}

// NewUpdater creates a new updater with defaults that we're updating this
// executable and it's going to be for the current OS and Architecture
func NewUpdater(currentVersion string, updateDataURL string) Updater {
	return Updater{
		currentVersion:     currentVersion,
		apiURL:             updateDataURL,
		binURL:             updateDataURL,
		diffURL:            updateDataURL,
		cacheDir:           "update",
		requester:          HTTPRequester{},
		updateableResolver: CurrentExeUpdatableResolver{},
		platformResolver:   CurrentPlatformResolver{},
	}
}

func (u Updater) UpdateAvailable() (bool, error) {
	return true, nil
}

// Run attempts to grab the latest version information and then applies the
// new patch if their is an update. If an update did occur, then we return
// true. If we did not update (already up to date) then we return false.
func (u Updater) Run() (updated bool, err error) {

	path, err := getExecRelativeDir(u.cacheDir)

	if err != nil {
		return false, err
	}

	// Create folder for storing updates if it doesn't exist
	if err := os.MkdirAll(path, 0777); err != nil {
		return false, err
	}

	return u.update()
}

func (u *Updater) update() (bool, error) {

	info, err := u.fetchInfo()
	if err != nil {
		return false, err
	}

	// No need to update
	if info.Version == u.currentVersion {
		return false, nil
	}

	oldPath, err := u.updateableResolver.Resolve()
	if err != nil {
		return false, err
	}

	up := update.New()

	if err := up.CanUpdate(); err != nil {
		return false, err
	}

	old, err := os.Open(oldPath)
	if err != nil {
		return false, err
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

		return false, err

		bin, err = u.getEntireBinaryForVersion(info)
		if err != nil {
			if err == ErrHashMismatch {
				log.Println("update: hash mismatch from full binary")
			} else {
				log.Println("update: fetching full binary,", err)
			}
			return false, err
		}
	}

	// close the old binary before installing because on windows
	// it can't be renamed if a handle to the file is still open
	old.Close()

	err, errRecover := up.FromStream(bytes.NewBuffer(bin))
	if errRecover != nil {
		return false, fmt.Errorf("update and recovery errors: %q %q", err, errRecover)
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (u *Updater) fetchInfo() (versionInfo, error) {
	if u.platformResolver == nil {
		return versionInfo{}, errors.New("Unable to reolve platform because resolver is nil")
	}

	platformToUpdate, err := u.platformResolver.Resolve()
	if err != nil {
		return versionInfo{}, err
	}

	r, err := u.fetch(fmt.Sprintf("%s%s/%s.json", u.apiURL, url.QueryEscape(u.cmdName), url.QueryEscape(platformToUpdate)))
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
		return versionInfo{}, errors.New("bad hash in info")
	}
	return info, nil
}

// getExeWithPatchForVersion retrives the patch for the current  version and
// applies it to the current executable passed in, and returns the results
func (u Updater) getExeWithPatchForVersion(old io.Reader, info versionInfo) ([]byte, error) {

	platformToUpdate, err := u.platformResolver.Resolve()
	if err != nil {
		return nil, err
	}

	r, err := u.fetch(u.diffURL + url.QueryEscape(u.cmdName) + "/" + url.QueryEscape(u.currentVersion) + "/" + url.QueryEscape(info.Version) + "/" + url.QueryEscape(platformToUpdate))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var buf bytes.Buffer
	err = binarydist.Patch(old, &buf, r)
	if err != nil {
		return nil, err
	}

	bin := buf.Bytes()
	log.Printf("bytes: %d", len(bin))

	if !verifySha(bin, info.Sha256) {
		return nil, ErrHashMismatch
	}

	return bin, nil
}

func (u Updater) getEntireBinaryForVersion(info versionInfo) ([]byte, error) {

	platformToUpdate, err := u.platformResolver.Resolve()
	if err != nil {
		return nil, err
	}

	r, err := u.fetch(fmt.Sprintf("%s%s/%s/%s.gz", u.binURL, url.QueryEscape(u.cmdName), url.QueryEscape(info.Version), url.QueryEscape(platformToUpdate)))
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
