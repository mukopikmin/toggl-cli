package model

import "testing"

func TestSortedProjectsDoesNotMutate(t *testing.T) {
	one, two := float64(1), float64(2)
	input := []Project{{ID: 1, Hidden: false, DisplayOrder: &two}, {ID: 2, Hidden: true}, {ID: 3, DisplayOrder: &one}}
	got := SortedProjects(input)
	if len(got) != 3 || got[0].ID != 3 || got[1].ID != 1 || got[2].ID != 2 {
		t.Fatalf("%+v", got)
	}
	if input[0].ID != 1 {
		t.Fatalf("input mutated: %+v", input)
	}
}

func TestProjectWithSettings(t *testing.T) {
	project := Project{ID: 42, Name: "Backend", Active: true}
	if got := project.WithSettings(ProjectSettings{}); got.DisplayName != "Backend" || got.Alias != "" {
		t.Fatalf("unexpected default project: %+v", got)
	}
	order := float64(3)
	got := project.WithSettings(ProjectSettings{DisplayName: "Infrastructure", Hidden: true, DisplayOrder: &order})
	if got.ID != 42 || !got.Active || got.Name != "Backend" || got.Alias != "Infrastructure" || got.DisplayName != got.Alias || !got.Hidden || got.DisplayOrder == nil || *got.DisplayOrder != 3 {
		t.Fatalf("unexpected resolved project: %+v", got)
	}
}
