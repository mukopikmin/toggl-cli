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

func TestSummaryDailyTotalFormats(t *testing.T) {
	one, two, hidden, absent := int64(1), int64(2), int64(3), int64(4)
	client := &fakeClient{
		projects: []model.Project{{ID: one, Name: "A"}, {ID: hidden, Name: "Hidden"}, {ID: two, Name: "B"}},
		entries: []model.TimeEntry{
			{ProjectID: &one, Start: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), DurationSeconds: 1},
			{ProjectID: &two, Start: time.Date(2026, 5, 1, 1, 0, 0, 0, time.UTC), DurationSeconds: 1},
			{ProjectID: &hidden, Start: time.Date(2026, 5, 1, 2, 0, 0, 0, time.UTC), DurationSeconds: 3600},
			{ProjectID: &absent, Start: time.Date(2026, 5, 1, 3, 0, 0, 0, time.UTC), DurationSeconds: 3600},
			{Start: time.Date(2026, 5, 1, 4, 0, 0, 0, time.UTC), DurationSeconds: 3600},
			{ProjectID: &one, Start: time.Date(2026, 5, 2, 11, 59, 30, 0, time.UTC), Running: true},
		},
	}
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"tsv", []string{"-ftsv"}, "Project\t2026-05-01\t2026-05-02\t2026-05-03\nA\t0.02\t0.5\t\nB\t0.02\t\t\nTotal\t0.03\t0.5\t0\n"},
		{"csv", []string{"--format", "csv"}, "Project,2026-05-01,2026-05-02,2026-05-03\nA,0.02,0.5,\nB,0.02,,\nTotal,0.03,0.5,0\n"},
		{"no project", []string{"-ftsv", "--no-project"}, "2026-05-01\t2026-05-02\t2026-05-03\n0.02\t0.5\t\n0.02\t\t\n0.03\t0.5\t0\n"},
		{"no date", []string{"-ftsv", "--no-date"}, "A\t0.02\t0.5\t\nB\t0.02\t\t\nTotal\t0.03\t0.5\t0\n"},
		{"no headers", []string{"-ftsv", "--no-project", "--no-date"}, "0.02\t0.5\t\n0.02\t\t\n0.03\t0.5\t0\n"},
		{"default table", nil, "┌─────────┬────────────┬────────────┬────────────┐\n│ Project │ 2026-05-01 │ 2026-05-02 │ 2026-05-03 │\n├─────────┼────────────┼────────────┼────────────┤\n│ A       │ 0.02       │ 0.5        │            │\n│ B       │ 0.02       │            │            │\n│ Total   │ 0.03       │ 0.5        │ 0          │\n└─────────┴────────────┴────────────┴────────────┘\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, out := configuredApp(t, client, "")
			appendProjectConfig(t, "\n[projects.3]\nhidden = true\n")
			args := append([]string{"summary", "2026-05-01", "2026-05-03", "--daily-total"}, tt.args...)
			if err := app.Run(context.Background(), args); err != nil {
				t.Fatal(err)
			}
			if out.String() != tt.want {
				t.Fatalf("got:\n%s\nwant:\n%s", out.String(), tt.want)
			}
		})
	}
	app, out := configuredApp(t, client, "")
	if err := app.Run(context.Background(), []string{"summary", "2026-05-01", "2026-05-03"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Total") {
		t.Fatalf("total must be opt-in: %q", out.String())
	}
}

func TestSummaryDailyTotalEmpty(t *testing.T) {
	for _, projects := range [][]model.Project{nil, {{ID: 1, Name: "A"}}} {
		app, out := configuredApp(t, &fakeClient{projects: projects}, "")
		if err := app.Run(context.Background(), []string{"summary", "2026-05-01", "2026-05-01", "--daily-total", "-ftsv"}); err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(out.String(), "Total\t0\n") {
			t.Fatal(out.String())
		}
	}
}

func TestSummaryDailyTotalMonthBoundaries(t *testing.T) {
	pid := int64(1)
	tests := []struct {
		zone, start, end, want string
	}{
		{"Asia/Tokyo", "2026-01-31T14:59:59Z", "2026-01-31T15:00:00Z", "Total\t1\t2\n"},
		{"America/Los_Angeles", "2026-02-01T07:59:59Z", "2026-02-01T08:00:00Z", "Total\t1\t2\n"},
		{"UTC", "2026-01-31T23:59:59Z", "2026-02-01T00:00:00Z", "Total\t1\t2\n"},
	}
	for _, tt := range tests {
		t.Run(tt.zone, func(t *testing.T) {
			start, _ := time.Parse(time.RFC3339, tt.start)
			end, _ := time.Parse(time.RFC3339, tt.end)
			client := &fakeClient{projects: []model.Project{{ID: pid, Name: "A"}}, entries: []model.TimeEntry{
				{ProjectID: &pid, Start: start, DurationSeconds: 60},
				{ProjectID: &pid, Start: end, DurationSeconds: 120},
			}}
			app, out := configuredApp(t, client, "")
			text := "workspace=\"1\"\ntoken=\"secret\"\ntimezone=\"" + tt.zone + "\"\n"
			if err := os.WriteFile(config.Path(os.Getenv("HOME")), []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			app.Now = func() time.Time { return time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC) }
			if err := app.Run(context.Background(), []string{"summary", "--days", "2", "--daily-total", "-ftsv"}); err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(out.String(), "Project\t2026-01-31\t2026-02-01\n") || !strings.HasSuffix(out.String(), tt.want) {
				t.Fatalf("%q", out.String())
			}
		})
	}
}

func TestSummaryDailyTotalInvalidUsage(t *testing.T) {
	tests := [][]string{
		{"summary", "--days", "1", "--daily-total", "--format", "json"},
		{"summary", "--days", "1", "--daily-total=false"},
		{"project", "list", "--daily-total"},
		{"time-entry", "list", "1", "2", "--daily-total"},
		{"config", "--daily-total"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			client := &fakeClient{}
			app := App{NewClient: useClient(client), Out: &bytes.Buffer{}}
			err := app.Run(context.Background(), args)
			var usageError UsageError
			if !errors.As(err, &usageError) || client.entryCalls != 0 || client.projectCalls != 0 {
				t.Fatalf("expected usage error before API access, got %v", err)
			}
		})
	}
}

type failingSummaryWriter struct {
	err error
}

func (w failingSummaryWriter) Write([]byte) (int, error) { return 0, w.err }

func TestSummaryReportsOutputFailure(t *testing.T) {
	want := errors.New("output unavailable")
	app, _ := configuredApp(t, &fakeClient{}, "")
	app.Out = failingSummaryWriter{err: want}
	if err := app.Run(context.Background(), []string{"summary", "--days", "1"}); !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}
