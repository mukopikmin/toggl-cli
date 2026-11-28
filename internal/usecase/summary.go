package usecase

import (
	"context"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

// LoadSummary aggregates time entries across an inclusive range of calendar dates.
func LoadSummary(
	ctx context.Context,
	reader TimeEntryReader,
	from, to time.Time,
	loc *time.Location,
	now time.Time,
	projects map[int64]model.ProjectSettings,
) (model.Summary, error) {
	entries, err := loadTimeEntries(ctx, reader, from, to, loc)
	if err != nil {
		return nil, err
	}
	return model.Summarize(entries, loc, now, projects), nil
}
