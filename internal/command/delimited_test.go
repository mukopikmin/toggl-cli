package command

import (
	"context"
	"encoding/csv"
	"reflect"
	"strings"
	"testing"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

func TestDelimitedEscaping(t *testing.T) {
	headers := []string{"plain", "comma", "tab", "quote", "newline", "carriage return", "empty"}
	rows := [][]string{{"日本語", "a,b", "a\tb", `a"b`, "a\nb", "a\rb", ""}}
	for _, format := range []string{"csv", "tsv"} {
		t.Run(format, func(t *testing.T) {
			text := delimited(headers, rows, format)
			r := csv.NewReader(strings.NewReader(text))
			if format == "tsv" {
				r.Comma = '\t'
			}
			got, err := r.ReadAll()
			want := append([][]string{headers}, rows...)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("got %q, %v; want %q", got, err, want)
			}
		})
	}
}

func TestDelimitedEmptyAndHeaderlessOutput(t *testing.T) {
	for _, format := range []string{"csv", "tsv"} {
		if got := delimited(nil, nil, format); got != "" {
			t.Fatalf("empty %s output: %q", format, got)
		}
		if got := delimited([]string{"Project"}, nil, format); got != "Project" {
			t.Fatalf("header-only %s output: %q", format, got)
		}
		if got := delimited(nil, [][]string{{"P"}}, format); got != "P" {
			t.Fatalf("headerless %s output: %q", format, got)
		}
	}
}

func TestCommandsUseFixedDelimiters(t *testing.T) {
	name := "Client, A\t\"quoted\"\nnext line"
	client := &fakeClient{
		projects: []model.Project{{ID: 1, DisplayName: name}},
		entries:  []model.TimeEntry{{ID: 1, Description: name, Start: "2026-05-01T00:00:00Z", DurationSeconds: 60}},
	}
	tests := []struct {
		name string
		args []string
		want [][]string
	}{
		{"summary", []string{"summary", "2026-05-01", "2026-05-01"}, [][]string{{"Project", "2026-05-01"}, {name, ""}}},
		{"time entry", []string{"time-entry", "list", "1", "1"}, [][]string{{"id", "description", "project_id", "start", "stop", "duration_minutes"}, {"1", name, "", "2026-05-01T00:00:00Z", "", "1"}}},
		{"project", []string{"project", "list"}, [][]string{{name}}},
		{"config", []string{"config"}, [][]string{{"Setting", "Value"}, {"WORKSPACE", "1"}, {"TIMEZONE", "UTC"}}},
	}
	for _, tt := range tests {
		for _, format := range []string{"default", "csv", "tsv"} {
			t.Run(tt.name+"/"+format, func(t *testing.T) {
				app, out := configuredApp(t, client, "")
				args := append([]string{}, tt.args...)
				if format != "default" {
					args = append(args, "--format", format)
				}
				if err := app.Run(context.Background(), args); err != nil {
					t.Fatal(err)
				}
				r := csv.NewReader(strings.NewReader(out.String()))
				if format != "csv" {
					r.Comma = '\t'
				}
				got, err := r.ReadAll()
				if err != nil || !reflect.DeepEqual(got, tt.want) {
					t.Fatalf("got %q, %v; want %q (output %q)", got, err, tt.want, out.String())
				}
			})
		}
	}
}
