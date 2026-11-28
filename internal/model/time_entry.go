package model

import "time"

type TimeEntry struct {
	ID              int64
	ProjectID       *int64
	Start           time.Time
	Stop            *time.Time
	DurationSeconds int64
	Running         bool
	Description     string
}

// ElapsedSeconds returns the stored duration for finished entries and
// measures running entries against the caller's reporting time.
func (e TimeEntry) ElapsedSeconds(now time.Time) int64 {
	if e.Running {
		return max(0, now.Unix()-e.Start.Unix())
	}
	return e.DurationSeconds
}
