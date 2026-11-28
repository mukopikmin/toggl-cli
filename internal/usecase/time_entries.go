package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

var ErrInvalidMonthDays = errors.New("start and end day must be valid dates")

type TimeEntryReader interface {
	// TimeEntries requests a half-open interval of absolute times.
	TimeEntries(ctx context.Context, start, end time.Time) ([]model.TimeEntry, error)
}

// LoadMonthTimeEntries retrieves entries for inclusive days of the current month in loc.
func LoadMonthTimeEntries(ctx context.Context, reader TimeEntryReader, startDay, endDay int, now time.Time, loc *time.Location) ([]model.TimeEntry, error) {
	year, month, _ := now.In(loc).Date()
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if startDay < 1 || endDay < startDay || endDay > lastDay {
		return nil, ErrInvalidMonthDays
	}
	from := time.Date(year, month, startDay, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, month, endDay, 0, 0, 0, 0, time.UTC)
	return loadTimeEntries(ctx, reader, from, to, loc)
}

// loadTimeEntries takes inclusive calendar dates represented as UTC midnights
// and requests the corresponding half-open interval in loc.
func loadTimeEntries(ctx context.Context, reader TimeEntryReader, from, to time.Time, loc *time.Location) ([]model.TimeEntry, error) {
	start := startOfDate(from, loc)
	end := startOfDate(to.AddDate(0, 0, 1), loc)
	return reader.TimeEntries(ctx, start, end)
}

// startOfDate returns the first instant of a calendar date in loc. If the
// date was skipped, it returns the start of the first date that follows it.
func startOfDate(date time.Time, loc *time.Location) time.Time {
	// IANA timezone offsets are less than 24 hours in magnitude. Start
	// before any possible local midnight, then walk the offset intervals.
	cursor := date.Add(-24 * time.Hour).In(loc)
	for {
		_, offset := cursor.Zone()
		_, next := cursor.ZoneBounds()
		if !next.IsZero() && !next.After(cursor) {
			// Go can report December 31 UTC as the end of a rule
			// interval in a future leap year. Advance to January 1.
			next = time.Date(cursor.UTC().Year()+1, time.January, 1, 0, 0, 0, 0, time.UTC)
		}
		start := date.Add(-time.Duration(offset) * time.Second).In(loc)
		if start.Before(cursor) {
			start = cursor
		}
		if next.IsZero() || start.Before(next) {
			return start
		}
		cursor = next
	}
}
