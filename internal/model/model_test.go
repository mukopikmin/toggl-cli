package model

import (
	"reflect"
	"testing"
	"time"
)

func TestDailyTotals(t *testing.T) {
	summary := Summary{
		"2026-01-31": {1: 1.5, 2: 2.25, 3: 100},
		"2026-02-01": {1: 0.004, 2: 0.004},
		"2026-02-03": {1: 50},
	}
	projects := []Project{{ID: 1}, {ID: 2}}
	days := []string{"2026-01-31", "2026-02-01", "2026-02-02"}
	got := DailyTotals(summary, projects, days)
	want := map[string]float64{"2026-01-31": 3.75, "2026-02-01": 0.008, "2026-02-02": 0}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for _, input := range []struct {
		summary  Summary
		projects []Project
	}{{nil, projects}, {summary, nil}} {
		got := DailyTotals(input.summary, input.projects, days)
		want := map[string]float64{"2026-01-31": 0, "2026-02-01": 0, "2026-02-02": 0}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if got := DailyTotals(summary, projects, nil); len(got) != 0 {
		t.Fatalf("expected no totals without dates, got %v", got)
	}
}

func TestSummarizeTimeZonesAndMonthBoundaries(t *testing.T) {
	pid := int64(1)
	entries := []TimeEntry{{ProjectID: &pid, Start: time.Date(2026, 1, 31, 14, 59, 59, 0, time.UTC), DurationSeconds: 60}, {ProjectID: &pid, Start: time.Date(2026, 1, 31, 15, 0, 0, 0, time.UTC), DurationSeconds: 120}}
	tests := []struct {
		name, zone string
		jan, feb   float64
	}{{"UTC", "UTC", 3, 0}, {"Tokyo", "Asia/Tokyo", 1, 2}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, _ := time.LoadLocation(tt.zone)
			got := Summarize(entries, loc, time.Now(), nil)
			if got["2026-01-31"][1] != tt.jan || got["2026-02-01"][1] != tt.feb {
				t.Fatalf("%v", got)
			}
		})
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
	entries := []TimeEntry{{ProjectID: &pid, Start: start, Running: true}, {ProjectID: nil, Start: start, DurationSeconds: 3600}}
	got := Summarize(entries, time.UTC, start.Add(45*time.Minute), nil)
	if got["2026-05-01"][7] != 45 || len(got["2026-05-01"]) != 1 {
		t.Fatalf("%v", got)
	}
}

func TestElapsedSecondsClampsRunningEntriesAfterReportTime(t *testing.T) {
	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	entry := TimeEntry{Start: now.Add(10 * time.Second), Running: true}
	if got := entry.ElapsedSeconds(now); got != 0 {
		t.Fatalf("got %d seconds, want zero", got)
	}
}

func TestProjectWithSettings(t *testing.T) {
	project := Project{ID: 42, Name: "Backend", Active: true}
	if got := project.WithSettings(ProjectSettings{}); got.DisplayName != "Backend" || got.Alias != "" {
		t.Fatalf("unexpected default project: %+v", got)
	}
	order := float64(3)
	got := project.WithSettings(ProjectSettings{DisplayName: "Infrastructure", Hidden: true, DisplayOrder: &order})
	if got.ID != 42 || !got.Active || got.Name != "Backend" || got.Alias != "Infrastructure" || got.DisplayName != got.Alias || !got.Hidden || got.DisplayOrder == nil || *got.DisplayOrder != 3 {
		t.Fatalf("unexpected resolved project: %+v", got)
	}
}
