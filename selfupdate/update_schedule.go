package selfupdate

import "time"

//go:generate mockgen -destination=./mocks/update_schedule.go -package=mocks -source=update_schedule.go

// CheckForUpdatesSchedule denotes when it's appropriate to check for an update
type CheckForUpdatesSchedule interface {
	ShouldCheckForUpdate(currentTime time.Time) (bool, error)
	UpdatesChecked(timeUpdatesWhereChecked time.Time) error
}
