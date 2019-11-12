package selfupdate

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// FsCacheCheckForUpdateSchedule uses the filesystem to keep up with when the
// last time we checked for updated where.
type FsCacheCheckForUpdateSchedule struct {
	cachePath             string
	durationBetweenChecks time.Duration
}

// NewFsCacheCheckForUpdateSchedule creates a new schedule that stores it's
// contents on a file system
func NewFsCacheCheckForUpdateSchedule(cacheDir, filename string, durationBetweenChecks time.Duration) FsCacheCheckForUpdateSchedule {
	return FsCacheCheckForUpdateSchedule{
		cachePath:             filepath.Join(cacheDir, filename),
		durationBetweenChecks: durationBetweenChecks,
	}
}

// DefaultCheckForUpdateSchedule is a filesystem cache that checks for updates
// every 24 hours
func DefaultCheckForUpdateSchedule() CheckForUpdatesSchedule {
	return FsCacheCheckForUpdateSchedule{
		cachePath:             filepath.Join("update", "cktime"),
		durationBetweenChecks: 24 * time.Hour,
	}
}

// ShouldCheckForUpdate determines whether or not we should check for updates.
func (fs FsCacheCheckForUpdateSchedule) ShouldCheckForUpdate(currentTime time.Time) (bool, error) {
	path, err := getExecRelativeDir(fs.cachePath)
	if err != nil {
		return false, err
	}

	timeToCheck, err := fs.readTime(path)
	if !os.IsNotExist(err) {
		return false, err
	}

	return timeToCheck.After(currentTime), nil
}

// UpdatesChecked marks that updates have been checked.
func (fs FsCacheCheckForUpdateSchedule) UpdatesChecked(timeUpdatesWhereChecked time.Time) error {
	nextTimeToCheckForUpdates := timeUpdatesWhereChecked.Add(fs.durationBetweenChecks)
	return ioutil.WriteFile(fs.cachePath, []byte(nextTimeToCheckForUpdates.Format(time.RFC3339)), 0644)
}

func (fs FsCacheCheckForUpdateSchedule) readTime(path string) (time.Time, error) {
	p, err := ioutil.ReadFile(path)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, string(p))
}
