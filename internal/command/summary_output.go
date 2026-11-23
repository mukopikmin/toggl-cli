package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

func (a App) writeSummary(ctx context.Context, settings model.SummaryOptions, from, to time.Time, sum model.Summary, projects []model.Project, watch, dailyTotal bool) error {
	if settings.Format == "json" {
		data, err := json.MarshalIndent(sum, "", "  ")
		if err != nil {
			return err
		}
		return a.writeSummaryText(string(data), settings.Clipboard)
	}

	days := model.DateRange(from, to)
	rows := summaryRows(sum, model.VisibleSorted(projects), days, dailyTotal)
	headers := append([]string{"Project"}, days...)
	if settings.Format == "table" {
		options := tableOutputOptions{watch: watch, clipboard: settings.Clipboard, truncateProjects: true}
		return a.writeTable(ctx, options, headers, rows, summaryHeaderVariants(days, from, to)...)
	}
	if settings.NoProject {
		headers = days
		for i := range rows {
			rows[i] = rows[i][1:]
		}
	}
	if settings.NoDate {
		headers = nil
	}
	return a.writeSummaryText(delimited(headers, rows, settings.Format), settings.Clipboard)
}

func (a App) writeSummaryText(text string, copyToClipboard bool) error {
	if _, err := fmt.Fprintln(a.Out, text); err != nil {
		return err
	}
	if copyToClipboard {
		return clipboard(text)
	}
	return nil
}

func summaryRows(sum model.Summary, projects []model.Project, days []string, dailyTotal bool) [][]string {
	rows := make([][]string, 0, len(projects)+1)
	formatMinutes := func(minutes float64) string {
		return strconv.FormatFloat(float64(int(minutes*100+0.5))/100, 'f', -1, 64)
	}
	for _, project := range projects {
		row := make([]string, len(days)+1)
		row[0] = project.DisplayName
		for i, day := range days {
			if minutes := sum[day][project.ID]; minutes != 0 {
				row[i+1] = formatMinutes(minutes)
			}
		}
		rows = append(rows, row)
	}
	if dailyTotal {
		totals := model.DailyTotals(sum, projects, days)
		row := make([]string, len(days)+1)
		row[0] = "Total"
		for i, day := range days {
			row[i+1] = formatMinutes(totals[day])
		}
		rows = append(rows, row)
	}
	return rows
}

func summaryHeaderVariants(days []string, from, to time.Time) [][]string {
	compact := append([]string{"Proj."}, days...)
	minimal := append([]string{"Proj."}, days...)
	sameYear := from.Year() == to.Year()
	sameMonth := sameYear && from.Month() == to.Month()
	var previousMonth string
	for i, day := range days {
		if sameYear {
			compact[i+1] = day[len(day)-len("01-02"):]
		}
		month := day[:len(day)-len("-02")]
		// Keep the first date of each month identifiable in a multi-month table.
		if sameMonth || month == previousMonth {
			minimal[i+1] = strings.TrimPrefix(day[len(day)-len("02"):], "0")
		} else {
			minimal[i+1] = compact[i+1]
		}
		previousMonth = month
	}
	return [][]string{compact, minimal}
}
