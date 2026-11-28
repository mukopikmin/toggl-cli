package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
)

func TestDefaultOutputIsTable(t *testing.T) {
	pid := int64(1)
	client := &fakeClient{
		projects: []model.Project{{ID: 1, Name: "Shown", Active: true}},
		entries:  []model.TimeEntry{{ID: 2, ProjectID: &pid, Description: "Work", Start: time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC), DurationSeconds: 90}},
	}
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"summary", []string{"summary", "2026-05-01", "2026-05-02"}, []string{"Project", "2026-05-01", "Shown", "1.5"}},
		{"time entries", []string{"time-entry", "list", "1", "2"}, []string{"description", "duration_minutes", "Work", "1.5"}},
		{"projects", []string{"project", "list"}, []string{"Toggl name", "Display name", "Shown"}},
		{"config", []string{"config"}, []string{"Setting", "Value", "WORKSPACE", "UTC"}},
		{"empty projects", []string{"project", "list"}, []string{"Toggl name", "Display name"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := client
			if tt.name == "empty projects" {
				fixture = &fakeClient{}
			}
			app, out := configuredApp(t, fixture, "")
			if err := app.Run(context.Background(), tt.args); err != nil {
				t.Fatal(err)
			}
			got := out.String()
			if !strings.HasPrefix(got, "┌") || !strings.HasSuffix(got, "┘\n") || strings.Contains(got, "secret") {
				t.Fatalf("expected bordered table without token, got %q", got)
			}
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Fatalf("missing %q in %q", want, got)
				}
			}
		})
	}
}

func TestRemovedSeparatorRejectedBeforeConfiguration(t *testing.T) {
	commands := [][]string{
		{"summary", "2026-05-01", "2026-05-02"},
		{"time-entry", "list", "1", "2"},
		{"project", "list"},
		{"config"},
	}
	for _, command := range commands {
		for _, flags := range [][]string{{"--separator", ","}, {"--separator="}, {"-s", ","}, {"-s,"}} {
			args := append(append([]string{}, command...), flags...)
			t.Run(strings.Join(args, " "), func(t *testing.T) {
				t.Setenv("HOME", t.TempDir())
				client := &fakeClient{}
				app := App{NewClient: useClient(client), Out: &bytes.Buffer{}}
				err := app.Run(context.Background(), args)
				var usageError UsageError
				if !errors.As(err, &usageError) || !strings.Contains(err.Error(), "unknown option") || client.entryCalls != 0 || client.projectCalls != 0 {
					t.Fatalf("expected unknown option before configuration/API access, got %v", err)
				}
			})
		}
	}
}

func TestCalendarBoundsAtMidnightDSTTransition(t *testing.T) {
	for _, args := range [][]string{
		{"summary", "2018-11-04", "2018-11-04", "--format", "tsv"},
		{"summary", "--days", "1", "--format", "tsv"},
		{"time-entry", "list", "4", "4"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			fake := &fakeClient{}
			client := &dateRangeClient{fakeClient: fake}
			app, out := configuredApp(t, fake, "")
			app.NewClient = useClient(client)
			app.Now = func() time.Time { return time.Date(2018, 11, 4, 12, 0, 0, 0, time.UTC) }
			text, err := config.EncodeInitial("1", "test-token", "America/Sao_Paulo")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(config.Path(os.Getenv("HOME")), []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			if err := app.Run(context.Background(), args); err != nil {
				t.Fatal(err)
			}
			if from, to := client.from.UTC().Format(time.RFC3339), client.to.UTC().Format(time.RFC3339); from != "2018-11-04T03:00:00Z" || to != "2018-11-05T02:00:00Z" {
				t.Fatalf("request from %s to %s", from, to)
			}
			if args[0] == "summary" && strings.TrimSpace(out.String()) != "Project	2018-11-04" {
				t.Fatalf("unexpected date columns: %q", out.String())
			}
		})
	}
}
