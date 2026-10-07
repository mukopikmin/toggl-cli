package model

import (
	"testing"
	"time"
)

func TestSummarizeTimeZonesAndMonthBoundaries(t *testing.T) {
	pid := int64(1)
	entries := []TimeEntry{{ProjectID: &pid, Start: "2026-01-31T14:59:59Z", DurationSeconds: 60}, {ProjectID: &pid, Start: "2026-01-31T15:00:00Z", DurationSeconds: 120}}
	tests := []struct {
		name, zone string
		jan, feb   float64
	}{{"UTC", "UTC", 3, 0}, {"Tokyo", "Asia/Tokyo", 1, 2}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, _ := time.LoadLocation(tt.zone)
			got, e := Summarize(entries, loc, time.Now())
			if e != nil {
				t.Fatal(e)
			}
			if got["2026-01-31"][1] != tt.jan || got["2026-02-01"][1] != tt.feb {
				t.Fatalf("%v", got)
			}
		})
	}
}
func TestDateRangeMonthEnds(t *testing.T) {
	a, _ := time.Parse(time.DateOnly, "2024-02-28")
	b, _ := time.Parse(time.DateOnly, "2024-03-01")
	got := DateRange(a, b)
	if len(got) != 3 || got[1] != "2024-02-29" {
		t.Fatal(got)
	}
}

func TestVisibleSortedDoesNotMutate(t *testing.T) {
	one, two := float64(1), float64(2)
	input := []Project{{ID: 1, Hidden: false, DisplayOrder: &two}, {ID: 2, Hidden: true}, {ID: 3, DisplayOrder: &one}}
	got := VisibleSorted(input)
	if len(got) != 2 || got[0].ID != 3 || got[1].ID != 1 {
		t.Fatalf("%+v", got)
	}
	if input[0].ID != 1 {
		t.Fatalf("input mutated: %+v", input)
	}
}

func TestSummarizeRunningAndUnassigned(t *testing.T) {
	pid := int64(7)
	start := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	entries := []TimeEntry{{ProjectID: &pid, Start: start.Format(time.RFC3339), DurationSeconds: -start.Unix()}, {ProjectID: nil, Start: start.Format(time.RFC3339), DurationSeconds: 3600}}
	got, err := Summarize(entries, time.UTC, start.Add(45*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if got["2026-05-01"][7] != 45 || len(got["2026-05-01"]) != 1 {
		t.Fatalf("%v", got)
	}
}
