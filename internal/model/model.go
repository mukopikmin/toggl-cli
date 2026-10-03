package model

import (
	"sort"
	"time"
)

type Project struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	DisplayName  string   `json:"displayName"`
	Active       bool     `json:"active"`
	Hidden       bool     `json:"hidden"`
	DisplayOrder *float64 `json:"displayOrder,omitempty"`
}
type TimeEntry struct {
	ID              int64
	ProjectID       *int64
	Start           string
	Stop            *string
	DurationSeconds int64
	Description     string
}
type Summary map[string]map[int64]float64

func VisibleSorted(projects []Project) []Project {
	r := append([]Project(nil), projects...)
	r = slicesDeleteHidden(r)
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
func slicesDeleteHidden(p []Project) []Project {
	r := p[:0]
	for _, v := range p {
		if !v.Hidden {
			r = append(r, v)
		}
	}
	return r
}

func Summarize(entries []TimeEntry, location *time.Location, now time.Time) (Summary, error) {
	r := Summary{}
	for _, e := range entries {
		if e.ProjectID == nil {
			continue
		}
		start, err := time.Parse(time.RFC3339Nano, e.Start)
		if err != nil {
			return nil, err
		}
		day := start.In(location).Format(time.DateOnly)
		if r[day] == nil {
			r[day] = map[int64]float64{}
		}
		seconds := e.DurationSeconds
		if seconds < 0 {
			seconds = now.Unix() + seconds
		}
		r[day][*e.ProjectID] += float64(seconds) / 60
	}
	return r, nil
}

func DateRange(start, end time.Time) []string {
	var r []string
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		r = append(r, d.Format(time.DateOnly))
	}
	return r
}
