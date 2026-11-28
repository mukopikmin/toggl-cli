package model

import (
	"errors"
	"testing"
	"time"
)

func TestSummaryOptionsDateRange(t *testing.T) {
	for _, tt := range []struct {
		name, zone, now, start, end string
		days                        int
	}{
		{"today", "UTC", "2026-05-01T00:00:00Z", "2026-05-01", "2026-05-01", 1},
		{"seven inclusive days", "UTC", "2026-05-01T12:00:00Z", "2026-04-25", "2026-05-01", 7},
		{"east month start", "Asia/Tokyo", "2026-04-30T15:00:00Z", "2026-04-30", "2026-05-01", 2},
		{"west month end", "America/Los_Angeles", "2026-06-01T06:59:00Z", "2026-05-30", "2026-05-31", 2},
		{"leap month", "UTC", "2024-03-01T12:00:00Z", "2024-02-29", "2024-03-01", 2},
		{"DST start", "America/Los_Angeles", "2026-03-09T12:00:00Z", "2026-03-08", "2026-03-09", 2},
		{"DST end", "America/Los_Angeles", "2026-11-02T12:00:00Z", "2026-11-01", "2026-11-02", 2},
		{"midnight DST start", "America/Sao_Paulo", "2018-11-04T12:00:00Z", "2018-11-04", "2018-11-04", 1},
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
			from, to, err := (SummaryPeriod{Days: &tt.days}).DateRange(now, loc)
			if err != nil || from.Format(time.DateOnly) != tt.start || to.Format(time.DateOnly) != tt.end {
				t.Fatalf("range = %s to %s, %v", from, to, err)
			}
			if from.Hour() != 0 || to.Hour() != 0 || from.Location() != time.UTC || to.Location() != time.UTC {
				t.Fatal("calendar dates must use UTC midnight")
			}
		})
	}
}

func TestSummaryOptionsOptionalPeriod(t *testing.T) {
	o := SummaryOptions{Output: SummaryOutput{Format: "table"}}
	if err := o.Validate(false); err != nil {
		t.Fatal(err)
	}
	if err := o.Validate(true); err == nil {
		t.Fatal("execution accepted missing period")
	}
	if _, _, err := o.Period.DateRange(time.Now(), time.UTC); err == nil {
		t.Fatal("resolved missing period")
	}
	period := SummaryPeriod{StartDate: "2024-02-29", EndDate: "2024-03-01"}
	from, to, err := period.DateRange(time.Now(), time.UTC)
	if err != nil || from.Format(time.DateOnly) != period.StartDate || to.Format(time.DateOnly) != period.EndDate {
		t.Fatalf("fixed range = %s to %s, %v", from, to, err)
	}
}

func TestSummaryOptionsRejectsZeroDays(t *testing.T) {
	days := 0
	for _, requirePeriod := range []bool{false, true} {
		if err := (SummaryPeriod{Days: &days}).Validate(requirePeriod); err == nil {
			t.Fatal("accepted zero days")
		}
	}
}

func TestSummaryPeriodResolvesIndependentlyOfOutput(t *testing.T) {
	options := SummaryOptions{
		Period: SummaryPeriod{StartDate: "2026-05-01", EndDate: "2026-05-02"},
		Output: SummaryOutput{Format: "table", NoDate: true},
	}
	if err := options.Validate(true); !errors.Is(err, ErrSummaryTableColumns) {
		t.Fatalf("expected a table output error, got %v", err)
	}
	if _, _, err := options.Period.DateRange(time.Time{}, time.UTC); err != nil {
		t.Fatalf("valid period could not be resolved: %v", err)
	}
}
