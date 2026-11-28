package model

import "time"

func DateRange(start, end time.Time) []string {
	var r []string
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		r = append(r, d.Format(time.DateOnly))
	}
	return r
}

// StartOfDate returns the first instant of a calendar date in loc. If the
// date was skipped, it returns the start of the first date that follows it.
func StartOfDate(date time.Time, loc *time.Location) time.Time {
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

// MonthDayRange resolves days of the current month into a half-open interval.
func MonthDayRange(now time.Time, startDay, endDay int, loc *time.Location) (start, end time.Time, valid bool) {
	year, month, _ := now.In(loc).Date()
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if startDay < 1 || endDay < startDay || endDay > lastDay {
		return time.Time{}, time.Time{}, false
	}
	start = StartOfDate(time.Date(year, month, startDay, 0, 0, 0, 0, time.UTC), loc)
	end = StartOfDate(time.Date(year, month, endDay+1, 0, 0, 0, 0, time.UTC), loc)
	return start, end, true
}
