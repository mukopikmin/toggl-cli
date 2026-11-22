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
			from, to, err := (SummaryOptions{Days: &tt.days}).DateRange(now, loc)
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

func TestSummaryOptionsRejectsZeroDays(t *testing.T) {
	days := 0
	for _, requirePeriod := range []bool{false, true} {
		if err := (SummaryOptions{Days: &days}).Validate(requirePeriod); err == nil {
			t.Fatal("accepted zero days")
		}
	}
}

func TestStartOfDate(t *testing.T) {
	for _, tt := range []struct{ zone, date, want string }{
		{"America/Sao_Paulo", "2018-11-04", "2018-11-04T01:00:00-02:00"},
		{"America/Sao_Paulo", "2018-11-05", "2018-11-05T00:00:00-02:00"},
		{"America/Havana", "2024-11-03", "2024-11-03T00:00:00-04:00"},
		{"Pacific/Apia", "2011-12-30", "2011-12-31T00:00:00+14:00"},
		{"America/New_York", "2040-12-31", "2040-12-31T00:00:00-05:00"},
	} {
		t.Run(tt.zone+"/"+tt.date, func(t *testing.T) {
			loc, err := time.LoadLocation(tt.zone)
			if err != nil {
				t.Fatal(err)
			}
			date, err := summaryDate(tt.date)
			if err != nil {
				t.Fatal(err)
			}
			if got := StartOfDate(date, loc).Format(time.RFC3339); got != tt.want {
				t.Fatalf("start = %s, want %s", got, tt.want)
			}
		})
	}
}
