package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

func TestProjectListFields(t *testing.T) {
	first, second := 1.0, 2.0
	projects := []model.Project{
		{ID: 20, Name: "Unordered", DisplayName: "Unordered", Active: true},
		{ID: 30, Name: "Original, name", Alias: "Shown", DisplayName: "Shown", Active: true, DisplayOrder: &second},
		{ID: 10, Name: "Private", DisplayName: "Private", Active: true, Hidden: true, DisplayOrder: &first},
	}
	tests := []struct {
		name    string
		args    []string
		project []model.Project
		want    string
	}{
		{
			name:    "table includes all fields and hidden projects in display order",
			args:    []string{"--include-hidden"},
			project: projects,
			want: table([]string{"Toggl name", "ID", "Alias", "Display name", "Hidden", "Active", "Display order"}, [][]string{
				{"Private", "10", "", "Private", "true", "true", "1"},
				{"Original, name", "30", "Shown", "Shown", "false", "true", "2"},
				{"Unordered", "20", "", "Unordered", "false", "true", ""},
			}, tableLayout{}) + "\n",
		},
		{
			name:    "table selects only requested fields in order",
			args:    []string{"--fields", "id,alias"},
			project: projects,
			want:    table([]string{"ID", "Alias"}, [][]string{{"30", "Shown"}, {"20", ""}}, tableLayout{}) + "\n",
		},
		{
			name:    "csv quotes names and excludes hidden by default",
			args:    []string{"--format", "csv", "--fields=name, id,hidden"},
			project: projects,
			want:    "name,id,hidden\n\"Original, name\",30,false\nUnordered,20,false\n",
		},
		{
			name:    "tsv includes hidden",
			args:    []string{"--format", "tsv", "--fields", "hidden,displayName", "--include-hidden"},
			project: projects,
			want:    "hidden\tdisplayName\ntrue\tPrivate\nfalse\tShown\nfalse\tUnordered\n",
		},
		{
			name:    "csv single optional field keeps empty rows",
			args:    []string{"--format", "csv", "--fields", "alias", "--include-hidden"},
			project: projects,
			want:    "alias\n\"\"\nShown\n\"\"\n",
		},
		{
			name: "empty tsv still prints selected headers",
			args: []string{"--format", "tsv", "--fields", "id,hidden"},
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

func TestProjectListJSONFields(t *testing.T) {
	order := 1.5
	projects := []model.Project{
		{ID: 20, Name: "Unordered", DisplayName: "Unordered", Active: true},
		{ID: 9007199254740993, Name: "Private", Alias: "Alias", DisplayName: "Alias", Active: true, Hidden: true, DisplayOrder: &order},
	}
	for _, tt := range []struct {
		name string
		args []string
		want []map[string]json.RawMessage
	}{
		{
			name: "all fields by default",
			want: []map[string]json.RawMessage{{
				"id": json.RawMessage(`20`), "name": json.RawMessage(`"Unordered"`), "alias": json.RawMessage(`null`),
				"displayName": json.RawMessage(`"Unordered"`), "active": json.RawMessage(`true`),
				"hidden": json.RawMessage(`false`), "displayOrder": json.RawMessage(`null`),
			}},
		},
		{
			name: "selected fields preserve scalar types and include hidden projects",
			args: []string{"--fields", "displayOrder,alias,id,hidden", "--include-hidden"},
			want: []map[string]json.RawMessage{
				{"displayOrder": json.RawMessage(`1.5`), "alias": json.RawMessage(`"Alias"`), "id": json.RawMessage(`9007199254740993`), "hidden": json.RawMessage(`true`)},
				{"displayOrder": json.RawMessage(`null`), "alias": json.RawMessage(`null`), "id": json.RawMessage(`20`), "hidden": json.RawMessage(`false`)},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, out := configuredApp(t, &fakeClient{projects: projects}, "")
			if err := app.Run(context.Background(), append([]string{"project", "list", "--format", "json"}, tt.args...)); err != nil {
				t.Fatal(err)
			}
			var got []map[string]json.RawMessage
			if err := json.Unmarshal(out.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("unexpected JSON: %s", out.String())
			}
		})
	}
}

func TestProjectListInvalidOptionsBeforeConfiguration(t *testing.T) {
	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"--fields"}, "requires a value"},
		{[]string{"--fields="}, "comma-separated list"},
		{[]string{"--fields", "id,,hidden"}, "comma-separated list"},
		{[]string{"--fields", "clientName"}, "unknown project field"},
		{[]string{"--format", "json", "--fields", "id, id"}, "duplicate project field"},
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
