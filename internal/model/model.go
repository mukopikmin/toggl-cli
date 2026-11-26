package model

import (
	"sort"
	"time"
)

type Client struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Project struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Alias        string   `json:"alias,omitempty"`
	DisplayName  string   `json:"displayName"`
	Active       bool     `json:"active"`
	Hidden       bool     `json:"hidden"`
	DisplayOrder *float64 `json:"displayOrder,omitempty"`
}
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
		return now.Unix() - e.Start.Unix()
	}
	return e.DurationSeconds
}

type Summary map[string]map[int64]float64

func VisibleSorted(projects []Project) []Project {
	return SortedProjects(projects, false)
}

func SortedProjects(projects []Project, includeHidden bool) []Project {
	r := make([]Project, 0, len(projects))
	for _, p := range projects {
		if includeHidden || !p.Hidden {
			r = append(r, p)
		}
	}
	sort.SliceStable(r, func(i, j int) bool {
		a, b := r[i].DisplayOrder, r[j].DisplayOrder
		if a == nil {
			return false
		}
		if b == nil {
			return true
		}
		return *a < *b
	})
	return r
}

func Summarize(entries []TimeEntry, location *time.Location, now time.Time) Summary {
	r := Summary{}
	for _, e := range entries {
		if e.ProjectID == nil {
			continue
		}
		day := e.Start.In(location).Format(time.DateOnly)
		if r[day] == nil {
			r[day] = map[int64]float64{}
		}
		r[day][*e.ProjectID] += float64(e.ElapsedSeconds(now)) / 60
	}
	return r
}

// DailyTotals sums unrounded minutes for the supplied projects and dates.
func DailyTotals(summary Summary, projects []Project, days []string) map[string]float64 {
	totals := make(map[string]float64, len(days))
	for _, day := range days {
		var total float64
		for _, p := range projects {
			total += summary[day][p.ID]
		}
		totals[day] = total
	}
	return totals
}

func DateRange(start, end time.Time) []string {
	var r []string
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		r = append(r, d.Format(time.DateOnly))
	}
	return r
}
