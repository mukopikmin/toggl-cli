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
	if len(projects) != 0 {
		visible := make([]model.TimeEntry, 0, len(entries))
		for _, entry := range entries {
			if entry.ProjectID != nil && projects[*entry.ProjectID].Hidden {
				continue
			}
			visible = append(visible, entry)
		}
		entries = visible
	}
	return model.Summarize(entries, loc, now), nil
}
