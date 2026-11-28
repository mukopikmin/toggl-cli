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

func LoadProjects(ctx context.Context, reader ProjectReader, settings map[int64]model.ProjectSettings) ([]model.Project, error) {
	projects, err := reader.Projects(ctx)
	if err != nil {
		return nil, err
	}
	resolved := make([]model.Project, len(projects))
	for i, project := range projects {
		resolved[i] = project.WithSettings(settings[project.ID])
	}
	return resolved, nil
}

func ProjectsToAdd(projects []model.Project, configured map[int64]model.ProjectSettings) []model.Project {
	var missing []model.Project
	seen := make(map[int64]bool)
	for _, project := range projects {
		if _, exists := configured[project.ID]; !exists && !seen[project.ID] {
			missing = append(missing, project)
			seen[project.ID] = true
		}
	}
	slices.SortFunc(missing, func(a, b model.Project) int { return cmp.Compare(a.ID, b.ID) })
	return missing
}
