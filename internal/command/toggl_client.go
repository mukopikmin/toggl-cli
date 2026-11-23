package command

import (
	"context"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
)

// TogglClient is the set of operations the CLI uses on an authenticated client.
type TogglClient interface {
	Clients(context.Context) ([]model.Client, error)
	Projects(context.Context) ([]model.Project, error)
	// TimeEntries requests a half-open interval of absolute times.
	TimeEntries(ctx context.Context, from, end time.Time) ([]model.TimeEntry, error)
}

func (a App) projects(ctx context.Context, cfg config.Config) ([]model.Project, error) {
	projects, err := a.NewClient(cfg.Workspace, cfg.Token).Projects(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]model.Project, len(projects))
	for i, project := range projects {
		preferences := cfg.Projects[project.ID]
		project.Alias = preferences.DisplayName
		project.DisplayName = preferences.DisplayName
		if project.DisplayName == "" {
			project.DisplayName = project.Name
		}
		project.Hidden = preferences.Hidden
		project.DisplayOrder = preferences.DisplayOrder
		views[i] = project
	}
	return views, nil
}
