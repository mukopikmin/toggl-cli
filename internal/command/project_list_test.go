package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

func TestProjectListColumns(t *testing.T) {
	first, second := 1.0, 2.0
	projects := []model.Project{
		{ID: 20, Name: "Unordered", DisplayName: "Unordered", Active: true},
		{ID: 30, Name: "Original, name", DisplayName: "Shown", Active: true, DisplayOrder: &second},
		{ID: 10, Name: "Private", DisplayName: "Private", Active: true, Hidden: true, DisplayOrder: &first},
	}
	tests := []struct {
		name    string
		args    []string
		project []model.Project
		want    string
	}{
		{
			name:    "table includes hidden in display order",
			args:    []string{"--columns", "id,displayName,name,hidden", "--include-hidden"},
			project: projects,
			want: table([]string{"ID", "Project", "Toggl name", "Hidden"}, [][]string{
				{"10", "Private", "Private", "true"},
				{"30", "Shown", "Original, name", "false"},
				{"20", "Unordered", "Unordered", "false"},
			}, tableLayout{}) + "\n",
		},
		{
			name:    "csv quotes names and excludes hidden by default",
			args:    []string{"--format", "csv", "--columns=name, id,hidden"},
			project: projects,
			want:    "name,id,hidden\n\"Original, name\",30,false\nUnordered,20,false\n",
		},
		{
			name:    "tsv includes hidden",
			args:    []string{"--format", "tsv", "--columns", "hidden,displayName", "--include-hidden"},
			project: projects,
			want:    "hidden\tdisplayName\ntrue\tPrivate\nfalse\tShown\nfalse\tUnordered\n",
		},
		{
			name: "empty tsv still prints selected headers",
			args: []string{"--format", "tsv", "--columns", "id,hidden"},
			want: "id\thidden\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeClient{projects: tt.project}
			app, out := configuredApp(t, client, "")
			if err := app.Run(context.Background(), append([]string{"project", "list"}, tt.args...)); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
			if client.projectCalls != 1 || client.clientCalls != 0 {
				t.Fatalf("unexpected API calls: projects=%d clients=%d", client.projectCalls, client.clientCalls)
			}
		})
	}
}

func TestProjectListJSONIncludesHidden(t *testing.T) {
	hidden := model.Project{ID: 10, Name: "Private", DisplayName: "Alias", Active: true, Hidden: true}
	app, out := configuredApp(t, &fakeClient{projects: []model.Project{hidden}}, "")
	if err := app.Run(context.Background(), []string{"project", "list", "--format", "json", "--include-hidden"}); err != nil {
		t.Fatal(err)
	}
	var projects []model.Project
	if err := json.Unmarshal(out.Bytes(), &projects); err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0] != hidden {
		t.Fatalf("got %v, want [%v]", projects, hidden)
	}
}

func TestProjectListInvalidOptionsBeforeConfiguration(t *testing.T) {
	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"--columns"}, "requires a value"},
		{[]string{"--columns="}, "comma-separated list"},
		{[]string{"--columns", "id,,hidden"}, "comma-separated list"},
		{[]string{"--columns", "clientName"}, "unknown project column"},
		{[]string{"--format", "json", "--columns", "id"}, "requires table, csv, or tsv format"},
		{[]string{"--include-hidden=false"}, "does not take an argument"},
	} {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			client := &fakeClient{}
			app := App{Client: client, Out: &bytes.Buffer{}}
			err := app.Run(context.Background(), append([]string{"project", "list"}, tt.args...))
			var usageErr UsageError
			if !errors.As(err, &usageErr) || !strings.Contains(err.Error(), tt.want) || client.projectCalls != 0 {
				t.Fatalf("expected usage error containing %q before configuration and API access, got %v", tt.want, err)
			}
		})
	}
}
