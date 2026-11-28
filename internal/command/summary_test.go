package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
)

func TestSummaryDailyTotalFormats(t *testing.T) {
	one, two, hidden, absent := int64(1), int64(2), int64(3), int64(4)
	client := &fakeClient{
		projects: []model.Project{{ID: one, Name: "A", Active: true}, {ID: hidden, Name: "Hidden", Active: true}, {ID: two, Name: "B", Active: true}},
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
	for _, projects := range [][]model.Project{nil, {{ID: 1, Name: "A", Active: true}}} {
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
			client := &fakeClient{projects: []model.Project{{ID: pid, Name: "A", Active: true}}, entries: []model.TimeEntry{
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

func TestDefaultTableRejectsCSVOptionsBeforeConfiguration(t *testing.T) {
	tests := [][]string{
		{"summary", "--days", "1", "--no-project"},
		{"summary", "--days", "1", "--no-date"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			client := &fakeClient{}
			app := App{NewClient: useClient(client), Out: &bytes.Buffer{}}
			err := app.Run(context.Background(), args)
			var usageError UsageError
			if !errors.As(err, &usageError) || !strings.Contains(err.Error(), "cannot be used with table format") || client.entryCalls != 0 || client.projectCalls != 0 {
				t.Fatalf("expected table option error before configuration/API access, got %v", err)
			}
		})
	}
}

func TestSummaryFormatsAndBoundaries(t *testing.T) {
	pid := int64(1)
	client := &fakeClient{projects: []model.Project{{ID: 1, Name: "P", Active: true}}, entries: []model.TimeEntry{{ID: 1, ProjectID: &pid, Start: time.Date(2026, 5, 1, 23, 30, 0, 0, time.UTC), DurationSeconds: 90}}}
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"tsv", []string{"summary", "2026-05-01", "2026-05-02", "--format", "tsv"}, "Project\t2026-05-01\t2026-05-02\nP\t1.5\t"},
		{"csv", []string{"summary", "2026-05-01", "2026-05-02", "--format", "csv"}, "Project,2026-05-01,2026-05-02\nP,1.5,"},
		{"json", []string{"summary", "--days", "2", "--format", "json"}, `"2026-05-01"`},
		{"table", []string{"summary", "--days", "2", "--format", "table"}, "│ Project"},
		{"no headers tsv", []string{"summary", "2026-05-01", "2026-05-02", "-ftsv", "--no-project", "--no-date"}, "1.5\t"},
		{"no headers csv", []string{"summary", "2026-05-01", "2026-05-02", "-fcsv", "--no-project", "--no-date"}, "1.5,"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, out := configuredApp(t, client, "")
			if err := app.Run(context.Background(), tt.args); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), tt.want) {
				t.Fatalf("%q", out.String())
			}
		})
	}
	app, _ := configuredApp(t, client, "")
	for _, flag := range []string{"--no-project", "--no-date"} {
		if err := app.Run(context.Background(), []string{"summary", "2026-05-01", "2026-05-02", "--format", "table", flag}); err == nil {
			t.Fatalf("expected table/%s error", flag)
		}
	}
}

func TestJSONSummaryOmitsHiddenProjects(t *testing.T) {
	hidden, visible := int64(42), int64(17)
	client := &fakeClient{entries: []model.TimeEntry{{ProjectID: &hidden, Start: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), DurationSeconds: 60}, {ProjectID: &visible, Start: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), DurationSeconds: 60}}}
	app, out := configuredApp(t, client, "")
	if err := os.WriteFile(config.Path(os.Getenv("HOME")), []byte("workspace='1'\ntoken='secret'\n[projects.42]\nhidden=true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(context.Background(), []string{"summary", "--days", "2", "-fjson"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "42") || !strings.Contains(out.String(), "17") {
		t.Fatal(out.String())
	}
	if client.projectCalls != 0 {
		t.Fatalf("got %d project requests for a JSON summary", client.projectCalls)
	}
}

func TestSummaryUsageBeforeConfiguration(t *testing.T) {
	tests := [][]string{
		{"summary"}, {"summary", "2026-05-01"},
		{"summary", "--days", "0"},
		{"summary", "2026-05-01", "2026-05-02", "--days", "1"},
		{"summary", "2026-02-30", "2026-03-01"},
		{"summary", "2026-05-02", "2026-05-01"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			client := &fakeClient{}
			app := App{NewClient: useClient(client), Out: &bytes.Buffer{}}
			err := app.Run(context.Background(), args)
			var usageError UsageError
			if !errors.As(err, &usageError) || client.entryCalls != 0 || client.projectCalls != 0 {
				t.Fatalf("expected usage error before configuration/API access, got %v", err)
			}
		})
	}
}

func TestSummaryDaysIncludesToday(t *testing.T) {
	tests := []struct {
		name, now, timezone string
		days                int
		wantStart, wantEnd  string
	}{
		{"today only", "2026-05-02T12:00:00Z", "UTC", 1, "2026-05-02T00:00:00Z", "2026-05-02T00:00:00Z"},
		{"seven days", "2026-05-10T12:00:00Z", "UTC", 7, "2026-05-04T00:00:00Z", "2026-05-10T00:00:00Z"},
		{"month start", "2026-05-01T12:00:00Z", "UTC", 2, "2026-04-30T00:00:00Z", "2026-05-01T00:00:00Z"},
		{"month end", "2026-05-31T23:59:59Z", "UTC", 31, "2026-05-01T00:00:00Z", "2026-05-31T00:00:00Z"},
		{"leap day", "2024-03-01T12:00:00Z", "UTC", 2, "2024-02-29T00:00:00Z", "2024-03-01T00:00:00Z"},
		{"year start", "2026-01-01T12:00:00Z", "UTC", 2, "2025-12-31T00:00:00Z", "2026-01-01T00:00:00Z"},
		{"ahead of UTC", "2026-04-30T15:30:00Z", "Asia/Tokyo", 1, "2026-05-01T00:00:00+09:00", "2026-05-01T00:00:00+09:00"},
		{"behind UTC", "2026-06-01T01:00:00Z", "America/Los_Angeles", 1, "2026-05-31T00:00:00-07:00", "2026-05-31T00:00:00-07:00"},
		{"daylight saving change", "2026-03-09T12:00:00Z", "America/New_York", 3, "2026-03-07T00:00:00-05:00", "2026-03-09T00:00:00-04:00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeClient{}
			client := &dateRangeClient{fakeClient: fake}
			app, out := configuredApp(t, fake, "")
			app.NewClient = useClient(client)
			now, err := time.Parse(time.RFC3339, tt.now)
			if err != nil {
				t.Fatal(err)
			}
			app.Now = func() time.Time { return now }
			text, err := config.EncodeInitial("1", "secret", tt.timezone)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(config.Path(os.Getenv("HOME")), []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			if err := app.Run(context.Background(), []string{"summary", "--days", strconv.Itoa(tt.days), "--format", "tsv"}); err != nil {
				t.Fatal(err)
			}
			wantEnd, _ := time.ParseInLocation(time.DateOnly, tt.wantEnd[:10], client.from.Location())
			if gotStart, gotEnd := client.from.Format(time.RFC3339), client.to.Format(time.RFC3339); gotStart != tt.wantStart || gotEnd != wantEnd.AddDate(0, 0, 1).Format(time.RFC3339) {
				t.Fatalf("range = %s through %s, want %s through %s", gotStart, gotEnd, tt.wantStart, tt.wantEnd)
			}
			headers := strings.Split(strings.TrimSpace(out.String()), "\t")
			if len(headers) != tt.days+1 || headers[1] != tt.wantStart[:10] || headers[len(headers)-1] != tt.wantEnd[:10] {
				t.Fatalf("unexpected date columns: %q", out.String())
			}
		})
	}
}

func TestSummaryMonth(t *testing.T) {
	tests := []struct {
		name, month, zone, now, start, end string
		days                               int
	}{
		{"31 days", "2026-10", "UTC", "2026-10-01T00:00:00Z", "2026-10-01", "2026-10-31", 31},
		{"30 days", "2026-04", "UTC", "2026-05-01T00:00:00Z", "2026-04-01", "2026-04-30", 30},
		{"February", "2026-02", "UTC", "2026-02-28T23:59:59Z", "2026-02-01", "2026-02-28", 28},
		{"leap February", "2024-02", "UTC", "2026-05-01T00:00:00Z", "2024-02-01", "2024-02-29", 29},
		{"year end", "2026-12", "UTC", "2026-12-31T23:59:59Z", "2026-12-01", "2026-12-31", 31},
		{"Tokyo next month", "2026-05", "Asia/Tokyo", "2026-04-30T15:00:00Z", "2026-05-01", "2026-05-31", 31},
		{"New York previous month", "2026-03", "America/New_York", "2026-04-01T00:00:00Z", "2026-03-01", "2026-03-31", 31},
	}
	for _, tt := range tests {
		for _, month := range []string{tt.month, strings.ReplaceAll(tt.month, "-", ""), strings.ReplaceAll(tt.month, "-", "/")} {
			t.Run(tt.name+"/"+month, func(t *testing.T) {
				fake := &fakeClient{}
				client := &dateRangeClient{fakeClient: fake}
				app, out := configuredApp(t, fake, "")
				app.NewClient = useClient(client)
				now, err := time.Parse(time.RFC3339, tt.now)
				if err != nil {
					t.Fatal(err)
				}
				app.Now = func() time.Time { return now }
				if err := os.WriteFile(config.Path(os.Getenv("HOME")), []byte("workspace=\"1\"\ntoken=\"secret\"\ntimezone=\""+tt.zone+"\"\n"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := app.Run(context.Background(), []string{"summary", "--month", month, "--format", "tsv"}); err != nil {
					t.Fatal(err)
				}
				if fake.entryCalls != 1 || client.from.Format(time.DateOnly) != tt.start || client.to.AddDate(0, 0, -1).Format(time.DateOnly) != tt.end {
					t.Fatalf("calls=%d from=%s to=%s", fake.entryCalls, client.from, client.to)
				}
				header := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\t")
				if len(header) != tt.days+1 || header[1] != tt.start || header[len(header)-1] != tt.end {
					t.Fatalf("unexpected date header: %q", out.String())
				}
			})
		}
	}
}

func TestSummaryMonthEqualsForm(t *testing.T) {
	for _, month := range []string{"2026-10", "202610", "2026/10"} {
		t.Run(month, func(t *testing.T) {
			app, out := configuredApp(t, &fakeClient{}, "")
			if err := app.Run(context.Background(), []string{"summary", "--month=" + month}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "2026-10-01") || !strings.Contains(out.String(), "2026-10-31") {
				t.Fatal(out.String())
			}
		})
	}
}

func TestMonthUsageBeforeConfiguration(t *testing.T) {
	tests := [][]string{
		{"summary", "--month"},
		{"summary", "--month="},
		{"summary", "--month", "2026-1"},
		{"summary", "--month", "26-01"},
		{"summary", "--month", "2026-00"},
		{"summary", "--month", "2026-13"},
		{"summary", "--month", "202600"},
		{"summary", "--month", "202613"},
		{"summary", "--month", "2026/00"},
		{"summary", "--month", "2026/13"},
		{"summary", "--month", "20261"},
		{"summary", "--month", "2026/1"},
		{"summary", "--month", "2026//10"},
		{"summary", "--month", "2026.10"},
		{"summary", "--month", "2026/10/01"},
		{"summary", "--month", "20261001"},
		{"summary", "--month", "2026-10-01"},
		{"summary", "--month", "October"},
		{"summary", "--month", "2026-10", "--days", "0"},
		{"summary", "--month", "2026-10", "2026-10-01", "2026-10-31"},
		{"summary", "--month", "2026-10", "2026-10-01"},
		{"time-entry", "list", "--month", "2026-10"},
		{"project", "list", "--month", "2026-10"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			client := &fakeClient{}
			app := App{NewClient: useClient(client), Out: &bytes.Buffer{}}
			err := app.Run(context.Background(), args)
			var usageError UsageError
			if !errors.As(err, &usageError) || client.entryCalls != 0 || client.projectCalls != 0 {
				t.Fatalf("expected usage error before configuration/API access, got %v", err)
			}
		})
	}
}
