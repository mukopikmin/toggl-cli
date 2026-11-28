package usecase

import (
	"testing"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

func TestProjectsToAddSelectsUniqueUnconfiguredProjectsInIDOrder(t *testing.T) {
	projects := []model.Project{{ID: 3}, {ID: 2}, {ID: 3}, {ID: 1}}
	configured := map[int64]model.ProjectSettings{2: {DisplayName: "Existing"}}
	got := ProjectsToAdd(projects, configured)
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 3 {
		t.Fatalf("unexpected projects to add: %+v", got)
	}
	if projects[0].ID != 3 || projects[1].ID != 2 {
		t.Fatal("project selection changed the reader's projects")
	}
}
