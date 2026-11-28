package usecase

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

type projectReaderStub struct {
	projects []model.Project
	err      error
}

func (r projectReaderStub) Projects(context.Context) ([]model.Project, error) {
	return r.projects, r.err
}

type projectWriterStub struct {
	projects []model.Project
	calls    int
	err      error
}

func (w *projectWriterStub) AddProjects(projects []model.Project) error {
	w.projects = projects
	w.calls++
	return w.err
}

func TestSyncProjectsWritesUniqueActiveUnconfiguredProjectsInIDOrder(t *testing.T) {
	projects := []model.Project{{ID: 3, Active: true}, {ID: 2, Active: true}, {ID: 3, Active: true}, {ID: 4}, {ID: 1, Active: true}}
	configured := map[int64]model.ProjectSettings{2: {DisplayName: "Existing"}}
	writer := &projectWriterStub{}
	count, err := SyncProjects(context.Background(), projectReaderStub{projects: projects}, writer, configured)
	if err != nil || count != 2 || writer.calls != 1 || len(writer.projects) != 2 || writer.projects[0].ID != 1 || writer.projects[1].ID != 3 {
		t.Fatalf("count=%d, err=%v, writer=%+v", count, err, writer)
	}
	if projects[0].ID != 3 || projects[1].ID != 2 {
		t.Fatal("project selection changed the reader's projects")
	}
}

func TestSyncProjectsEmptyAndErrors(t *testing.T) {
	readError := errors.New("reader unavailable")
	writeError := errors.New("configuration unavailable")
	for _, tt := range []struct {
		name      string
		reader    projectReaderStub
		writerErr error
		want      error
		calls     int
	}{
		{name: "no unconfigured projects", reader: projectReaderStub{projects: []model.Project{{ID: 1, Active: true}}}},
		{name: "failed read", reader: projectReaderStub{err: readError}, want: readError},
		{name: "failed write", reader: projectReaderStub{projects: []model.Project{{ID: 2, Active: true}}}, writerErr: writeError, want: writeError, calls: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writer := &projectWriterStub{err: tt.writerErr}
			count, err := SyncProjects(context.Background(), tt.reader, writer, map[int64]model.ProjectSettings{1: {}})
			if count != 0 || !errors.Is(err, tt.want) || writer.calls != tt.calls {
				t.Fatalf("count=%d, err=%v, writes=%d", count, err, writer.calls)
			}
		})
	}
}

func TestLoadProjectsSelectsAndOrdersDisplayProjects(t *testing.T) {
	reader := projectReaderStub{projects: []model.Project{
		{ID: 1, Name: "API name", Active: true},
		{ID: 2, Name: "Inactive"},
		{ID: 3, Name: "Hidden", Active: true},
		{ID: 4, Name: "Earlier", Active: true},
	}}
	first, second := 1.0, 2.0
	settings := map[int64]model.ProjectSettings{
		1: {DisplayName: "Alias"},
		3: {Hidden: true, DisplayOrder: &second},
		4: {DisplayOrder: &first},
	}
	for _, tt := range []struct {
		name          string
		includeHidden bool
		wantIDs       []int64
	}{
		{"visible", false, []int64{4, 1}},
		{"include hidden", true, []int64{4, 3, 1}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			projects, err := LoadProjects(context.Background(), reader, settings, tt.includeHidden)
			var ids []int64
			for _, project := range projects {
				ids = append(ids, project.ID)
			}
			if err != nil || !slices.Equal(ids, tt.wantIDs) {
				t.Fatalf("got IDs %v, error %v; want %v", ids, err, tt.wantIDs)
			}
			if got := projects[len(projects)-1]; got.DisplayName != "Alias" || got.Name != "API name" {
				t.Fatalf("alias not applied: %+v", got)
			}
		})
	}
}
