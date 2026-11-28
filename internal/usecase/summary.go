package usecase

import (
	"context"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

type TimeEntryReader interface {
	// TimeEntries requests a half-open interval of absolute times.
	TimeEntries(ctx context.Context, start, end time.Time) ([]model.TimeEntry, error)
}

// LoadSummary aggregates entries for the inclusive calendar dates from through to.
func LoadSummary(
	ctx context.Context,
	reader TimeEntryReader,
	from, to time.Time,
	loc *time.Location,
	now time.Time,
	projects map[int64]model.ProjectSettings,
) (model.Summary, error) {
	start := model.StartOfDate(from, loc)
	end := model.StartOfDate(to.AddDate(0, 0, 1), loc)
	entries, err := reader.TimeEntries(ctx, start, end)
	if err != nil {
		return nil, err
	}
	return model.Summarize(entries, loc, now, projects), nil
}
