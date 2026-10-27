package model

import (
	"fmt"
	"regexp"
	"time"
)

// SummaryOptions describes a reusable summary without credentials or resolved dates.
type SummaryOptions struct {
	StartDate string  `toml:"start_date,omitempty"`
	EndDate   string  `toml:"end_date,omitempty"`
	Days      *int    `toml:"days,omitempty"`
	Format    string  `toml:"format,omitempty"`
	Separator *string `toml:"separator,omitempty"`
	NoProject bool    `toml:"no_project"`
	NoDate    bool    `toml:"no_date"`
	Clipboard bool    `toml:"clipboard"`
}

// Validate allows a missing period when storing an output-only template.
func (o SummaryOptions) Validate(requirePeriod bool) error {
	if o.Format != "" && o.Format != "csv" && o.Format != "json" && o.Format != "table" {
		return fmt.Errorf("format must be csv, json, or table")
	}
	if o.Format == "table" && (o.Separator != nil || o.NoProject || o.NoDate) {
		return fmt.Errorf("--separator, --no-project, and --no-date cannot be used with table format")
	}
	if o.Days != nil {
		if *o.Days < 0 || int64(*o.Days) > 9007199254740991 {
			return fmt.Errorf("days must be a non-negative safe integer")
		}
		if o.StartDate != "" || o.EndDate != "" {
			return fmt.Errorf("summary accepts either start and end date or --days, not both")
		}
		return nil
	}
	if o.StartDate == "" && o.EndDate == "" && !requirePeriod {
		return nil
	}
	if o.StartDate == "" || o.EndDate == "" {
		return fmt.Errorf("summary requires start and end date or --days")
	}
	from, err := summaryDate(o.StartDate)
	if err != nil {
		return err
	}
	to, err := summaryDate(o.EndDate)
	if err != nil {
		return err
	}
	if from.After(to) {
		return fmt.Errorf("start date must not be after end date")
	}
	return nil
}

func summaryDate(value string) (time.Time, error) {
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(value) {
		return time.Time{}, fmt.Errorf("start and end date must use YYYY-MM-DD")
	}
	d, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("start and end date must be valid dates")
	}
	return d, nil
}

// DateRange resolves relative dates at execution time in the configured timezone.
func (o SummaryOptions) DateRange(now time.Time, loc *time.Location) (time.Time, time.Time, error) {
	if err := o.Validate(true); err != nil {
		return time.Time{}, time.Time{}, err
	}
	if o.Days != nil {
		now = now.In(loc)
		to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		return to.AddDate(0, 0, -*o.Days), to, nil
	}
	from, _ := summaryDate(o.StartDate)
	to, _ := summaryDate(o.EndDate)
	return from, to, nil
}
