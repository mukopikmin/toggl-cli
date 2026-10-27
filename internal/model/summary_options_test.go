package model

import (
	"testing"
	"time"
)

func TestSummaryOptionsDateRange(t *testing.T) {
	for _, tt := range []struct {
		name, zone, now, start, end string
		days                        int
	}{
		{"today", "UTC", "2026-05-01T00:00:00Z", "2026-05-01", "2026-05-01", 0},
		{"seven inclusive days", "UTC", "2026-05-01T12:00:00Z", "2026-04-25", "2026-05-01", 6},
		{"east month start", "Asia/Tokyo", "2026-04-30T15:00:00Z", "2026-04-30", "2026-05-01", 1},
		{"west month end", "America/Los_Angeles", "2026-06-01T06:59:00Z", "2026-05-30", "2026-05-31", 1},
		{"leap month", "UTC", "2024-03-01T12:00:00Z", "2024-02-29", "2024-03-01", 1},
		{"DST start", "America/Los_Angeles", "2026-03-09T12:00:00Z", "2026-03-08", "2026-03-09", 1},
		{"DST end", "America/Los_Angeles", "2026-11-02T12:00:00Z", "2026-11-01", "2026-11-02", 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, tt.now)
			if err != nil {
				t.Fatal(err)
			}
			loc, err := time.LoadLocation(tt.zone)
			if err != nil {
				t.Fatal(err)
			}
			from, to, err := (SummaryOptions{Days: &tt.days}).DateRange(now, loc)
			if err != nil || from.Format(time.DateOnly) != tt.start || to.Format(time.DateOnly) != tt.end {
				t.Fatalf("range = %s to %s, %v", from, to, err)
			}
			if from.Hour() != 0 || to.Hour() != 0 || from.Location() != loc || to.Location() != loc {
				t.Fatal("range must use local midnight")
			}
		})
	}
}

func TestSummaryOptionsOptionalPeriod(t *testing.T) {
	o := SummaryOptions{Format: "table"}
	if err := o.Validate(false); err != nil {
		t.Fatal(err)
	}
	if err := o.Validate(true); err == nil {
		t.Fatal("execution accepted missing period")
	}
	if _, _, err := o.DateRange(time.Now(), time.UTC); err == nil {
		t.Fatal("resolved missing period")
	}
	o = SummaryOptions{StartDate: "2024-02-29", EndDate: "2024-03-01"}
	from, to, err := o.DateRange(time.Now(), time.UTC)
	if err != nil || from.Format(time.DateOnly) != o.StartDate || to.Format(time.DateOnly) != o.EndDate {
		t.Fatalf("fixed range = %s to %s, %v", from, to, err)
	}
}
