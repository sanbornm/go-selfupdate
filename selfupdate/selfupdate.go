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
// updater := selfupdate.NewUpdater(version, "http://updates.yourdomain.com/", "myapp")
// updater.Run()
// ```
type Updater struct {
	currentVersion     string            // Currently running version.
	apiURL             string            // Base URL for API requests (json files).
	cmdName            string            // Command name is appended to the ApiURL like http://apiurl/CmdName/. This represents one binary.
	binURL             string            // Base URL for full binary downloads.
	diffURL            string            // Base URL for diff downloads.
	requester          Requester         // Optional parameter to override existing http request handler
	updateableResolver UpdatableResolver // Finds the thing that needs to be updated
	platformResolver   PlatformResolver  // Figures out what platform we need to update for
}

// NewUpdater creates a new updater with defaults that we're updating this
// executable and it's going to be for the current OS and Architecture
func NewUpdater(currentVersion, updateDataURL, urlPostfix string) Updater {
	return Updater{
		currentVersion:     currentVersion,
		apiURL:             updateDataURL,
		binURL:             updateDataURL,
		diffURL:            updateDataURL,
		requester:          HTTPRequester{},
		updateableResolver: CurrentExeUpdatableResolver{},
		platformResolver:   CurrentPlatformResolver{},
		cmdName:            urlPostfix,
	}
}

// SetUpdatableResolver sets what we use to determine which file needs to get
// updated.
//
// NOTICE: This does not change the current updater, but makes a new one with
// the resolver property changed
func (u Updater) SetUpdatableResolver(resolver UpdatableResolver) Updater {
	return Updater{
		currentVersion:     u.currentVersion,
		apiURL:             u.apiURL,
		binURL:             u.binURL,
		diffURL:            u.diffURL,
		requester:          u.requester,
		updateableResolver: resolver,
		platformResolver:   u.platformResolver,
		cmdName:            u.cmdName,
	}
}

// SetRequester sets what we use to make requests and get binaries. By default uses
// http. Can replace this with your own to add things like middleware.
//
// NOTICE: This does not change the current updater, but makes a new one with
// the requester property changed
func (u Updater) SetRequester(requester Requester) Updater {
	return Updater{
		currentVersion:     u.currentVersion,
		apiURL:             u.apiURL,
		binURL:             u.binURL,
		diffURL:            u.diffURL,
		requester:          requester,
		updateableResolver: u.updateableResolver,
		platformResolver:   u.platformResolver,
		cmdName:            u.cmdName,
	}
}

// SetPlatformResolver sets what we use to determine what platform the file
// we're trying to update is for
//
// NOTICE: This does not change the current updater, but makes a new one with
// the resolver property changed
func (u Updater) SetPlatformResolver(resolver PlatformResolver) Updater {
	return Updater{
		currentVersion:     u.currentVersion,
		apiURL:             u.apiURL,
		binURL:             u.binURL,
		diffURL:            u.diffURL,
		requester:          u.requester,
		updateableResolver: u.updateableResolver,
		platformResolver:   resolver,
		cmdName:            u.cmdName,
	}
}

// UpdateAvailable fetches info from the server specificed and and checks if
// what the version specified on the server matches what this program's version
// is.
func (u Updater) UpdateAvailable() (bool, error) {
	info, err := u.fetchInfo()
	if err != nil {
		return false, err
	}
	return info.Version != u.currentVersion, nil
}

// Run attempts to grab the latest version information and then applies the
// new patch if their is an update. If an update did occur, then we return
// true. If we did not update (already up to date) then we return false.
func (u *Updater) Run() (bool, error) {

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
	up.Target(oldPath)

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
			// If we indeed did have a url for discovering the binary diffs,
			// then we know we indeed did fail patching
			if u.diffURL != "" {
				log.Printf("error patching binary: %s", err)
			}
		}

		bin, err = u.getEntireBinaryForVersion(info)
		if err != nil {
			if err == ErrHashMismatch {
				log.Println("update: hash mismatch from full binary")
			} else {
				log.Printf("error fetching full binary: %s", err)
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

	u.currentVersion = info.Version

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
