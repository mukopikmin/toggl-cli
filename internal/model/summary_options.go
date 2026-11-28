package model

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

var (
	ErrSummaryPeriodRequired = errors.New("summary period requires a date range or a day count")
	ErrSummaryPeriodConflict = errors.New("summary period cannot specify both a date range and a day count")
	ErrSummaryTableColumns   = errors.New("project and date columns cannot be omitted in table format")
)

// SummaryOptions describes the period and output preferences of a reusable summary.
type SummaryOptions struct {
	Period SummaryPeriod
	Output SummaryOutput
}

// Validate allows a missing period when storing an output-only template.
func (o SummaryOptions) Validate(requirePeriod bool) error {
	if err := o.Output.Validate(); err != nil {
		return err
	}
	return o.Period.Validate(requirePeriod)
}

type SummaryOutput struct {
	Format    string
	NoProject bool
	NoDate    bool
	Clipboard bool
}

func (o SummaryOutput) Validate() error {
	if o.Format != "" && o.Format != "csv" && o.Format != "tsv" && o.Format != "json" && o.Format != "table" {
		return fmt.Errorf("format must be csv, tsv, json, or table")
	}
	if o.Format == "table" && (o.NoProject || o.NoDate) {
		return ErrSummaryTableColumns
	}
	return nil
}

type SummaryPeriod struct {
	StartDate string
	EndDate   string
	Days      *int
}

func (p SummaryPeriod) Validate(required bool) error {
	if p.Days != nil {
		if *p.Days <= 0 || int64(*p.Days) > 9007199254740991 {
			return fmt.Errorf("days must be a positive safe integer")
		}
		if p.StartDate != "" || p.EndDate != "" {
			return ErrSummaryPeriodConflict
		}
		return nil
	}
	if p.StartDate == "" && p.EndDate == "" && !required {
		return nil
	}
	if p.StartDate == "" || p.EndDate == "" {
		return ErrSummaryPeriodRequired
	}
	from, err := summaryDate(p.StartDate)
	if err != nil {
		return err
	}
	to, err := summaryDate(p.EndDate)
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

// DateRange returns the inclusive calendar dates as UTC midnights for date arithmetic.
// Relative dates use the current day in loc.
func (p SummaryPeriod) DateRange(now time.Time, loc *time.Location) (time.Time, time.Time, error) {
	if err := p.Validate(true); err != nil {
		return time.Time{}, time.Time{}, err
	}
	if p.Days != nil {
		now = now.In(loc)
		to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return to.AddDate(0, 0, 1-*p.Days), to, nil
	}
	from, _ := summaryDate(p.StartDate)
	to, _ := summaryDate(p.EndDate)
	return from, to, nil
}
