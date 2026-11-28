package usecase

import (
	"cmp"
	"context"
	"slices"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

type ProjectReader interface {
	Projects(context.Context) ([]model.Project, error)
}

type ProjectWriter interface {
	AddProjects([]model.Project) error
}

// LoadProjects returns active projects with display settings applied and in display order.
func LoadProjects(ctx context.Context, reader ProjectReader, settings map[int64]model.ProjectSettings, includeHidden bool) ([]model.Project, error) {
	projects, err := reader.Projects(ctx)
	if err != nil {
		return nil, err
	}
	resolved := make([]model.Project, 0, len(projects))
	for _, project := range projects {
		preferences := settings[project.ID]
		if !project.Active || (preferences.Hidden && !includeHidden) {
			continue
		}
		resolved = append(resolved, project.WithSettings(preferences))
	}
	return model.SortedProjects(resolved), nil
}

func SyncProjects(ctx context.Context, reader ProjectReader, writer ProjectWriter, configured map[int64]model.ProjectSettings) (int, error) {
	projects, err := reader.Projects(ctx)
	if err != nil {
		return 0, err
	}
	missing := projectsToAdd(projects, configured)
	if len(missing) == 0 {
		return 0, nil
	}
	if err := writer.AddProjects(missing); err != nil {
		return 0, err
	}
	return len(missing), nil
}

func projectsToAdd(projects []model.Project, configured map[int64]model.ProjectSettings) []model.Project {
	var missing []model.Project
	seen := make(map[int64]bool)
	for _, project := range projects {
		if _, exists := configured[project.ID]; project.Active && !exists && !seen[project.ID] {
			missing = append(missing, project)
			seen[project.ID] = true
		}
	}
	slices.SortFunc(missing, func(a, b model.Project) int { return cmp.Compare(a.ID, b.ID) })
	return missing
}
