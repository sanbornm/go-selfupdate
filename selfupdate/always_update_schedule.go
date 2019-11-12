package selfupdate

import "time"

// AlwaysCheckForUpdatesSchedule will always want to update
type AlwaysCheckForUpdatesSchedule struct {
	secondsToWaitBetweenChecking int
}

// NewAlwaysCheckForUpdatesSchedule that will want to check if
func NewAlwaysCheckForUpdatesSchedule(secondsToWaitBetweenChecking int) AlwaysCheckForUpdatesSchedule {
	return AlwaysCheckForUpdatesSchedule{
		secondsToWaitBetweenChecking: secondsToWaitBetweenChecking,
	}
}

// NextTimeToCheck tells us the next time we should check if there are any updates available
func (a AlwaysCheckForUpdatesSchedule) NextTimeToCheck() (time.Time, error) {
	return time.Now().Add(time.Duration(a.secondsToWaitBetweenChecking) * time.Second), nil
}

// ShouldCheckForUpdate will always return true
func (a AlwaysCheckForUpdatesSchedule) ShouldCheckForUpdate() (bool, error) {
	return true, nil
}
