package model

import "time"

type Summary map[string]map[int64]float64

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
