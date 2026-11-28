package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
)

func TestTimeEntryListSelectedFields(t *testing.T) {
	projectID := int64(8)
	stopTime := time.Date(2026, 5, 1, 1, 0, 0, 0, time.UTC)
	stop := stopTime.Format(time.RFC3339Nano)
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	entries := []model.TimeEntry{
		{ID: 2, ProjectID: &projectID, Description: "Finished, entry", Start: start, Stop: &stopTime, DurationSeconds: 3600},
		{ID: 1, Description: "Running", Start: now.Add(-90 * time.Second), Running: true},
	}
	for _, tt := range []struct {
		format, fields, want string
	}{
		{"table", "duration_minutes,id", table([]string{"duration_minutes", "id"}, [][]string{{"60", "2"}, {"1.5", "1"}}, tableLayout{}) + "\n"},
		{"csv", "stop", "stop\n" + stop + "\n\"\"\n"},
		{"tsv", "description, project_id", "description\tproject_id\nFinished, entry\t8\nRunning\t\n"},
		{"json", "stop,project_id,duration_minutes", `[
			{"stop": "2026-05-01T01:00:00Z", "project_id": 8, "duration_minutes": 60},
			{"stop": null, "project_id": null, "duration_minutes": 1.5}
		]`},
	} {
		t.Run(tt.format, func(t *testing.T) {
			client := &fakeClient{entries: entries}
			app, out := configuredApp(t, client, "")
			app.Now = func() time.Time { return now }
			args := []string{"time-entry", "list", "1", "2", "--format", tt.format, "--fields", tt.fields}
			if err := app.Run(context.Background(), args); err != nil {
				t.Fatal(err)
			}
			if tt.format == "json" {
				var got, want []map[string]json.RawMessage
				if err := json.Unmarshal(out.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal([]byte(tt.want), &want); err != nil || !reflect.DeepEqual(got, want) {
					t.Fatalf("got %q, want %s (error %v)", out.String(), tt.want, err)
				}
			} else if got := out.String(); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
			if client.entryCalls != 1 || client.clientCalls != 0 || client.projectCalls != 0 {
				t.Fatalf("unexpected API calls: entries=%d projects=%d clients=%d", client.entryCalls, client.projectCalls, client.clientCalls)
			}
		})
	}
}

func TestTimeEntryListInvalidFieldsBeforeConfiguration(t *testing.T) {
	for _, tt := range []struct {
		field, want string
	}{
		{"", "comma-separated list"},
		{"name", "unknown time-entry field"},
		{"id,id", "duplicate time-entry field"},
	} {
		t.Run(tt.field, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			client := &fakeClient{}
			app := App{NewClient: useClient(client), Out: &bytes.Buffer{}}
			err := app.Run(context.Background(), []string{"time-entry", "list", "1", "2", "--fields", tt.field})
			var usageErr UsageError
			if !errors.As(err, &usageErr) || !strings.Contains(err.Error(), tt.want) || client.entryCalls != 0 {
				t.Fatalf("expected usage error containing %q before configuration and API access, got %v", tt.want, err)
			}
		})
	}
}

func TestTimeEntryFormats(t *testing.T) {
	pid := int64(8)
	stop := time.Date(2026, 5, 1, 1, 0, 0, 0, time.UTC)
	client := &fakeClient{entries: []model.TimeEntry{{ID: 2, ProjectID: &pid, Description: "a,b", Start: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), Stop: &stop, DurationSeconds: 3600}}}
	for _, format := range []string{"csv", "tsv", "json", "table"} {
		t.Run(format, func(t *testing.T) {
			app, out := configuredApp(t, client, "")
			if err := app.Run(context.Background(), []string{"time-entry", "list", "1", "2", "--format", format}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "a,b") {
				t.Fatal(out.String())
			}
		})
	}
}

func TestTimeEntryListSortsByInstant(t *testing.T) {
	tokyo := time.FixedZone("Tokyo", 9*60*60)
	tokyoStart := time.Date(2026, 5, 2, 9, 0, 0, 0, tokyo)
	utcStart := tokyoStart.UTC()
	client := &fakeClient{entries: []model.TimeEntry{
		{ID: 3, Start: utcStart.Add(time.Hour)},
		{ID: 2, Start: utcStart},
		{ID: 1, Start: tokyoStart},
	}}
	app, out := configuredApp(t, client, "")
	if err := app.Run(context.Background(), []string{"time-entry", "list", "2", "2", "--format", "tsv"}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(out.String(), "\n")
	if !strings.HasPrefix(lines[1], "1\t") || !strings.HasPrefix(lines[2], "2\t") || !strings.HasPrefix(lines[3], "3\t") {
		t.Fatal(out.String())
	}
}

func TestTimeEntryCalendarDays(t *testing.T) {
	tokyo := time.FixedZone("Tokyo", 9*60*60)
	west := time.FixedZone("West", -8*60*60)
	tests := []struct {
		name               string
		now                time.Time
		start, end         string
		wantStart, wantEnd string
		fail               bool
		zone               string
	}{
		{"month start", time.Date(2026, 5, 1, 12, 0, 0, 0, tokyo), "1", "1", "2026-05-01", "2026-05-01", false, "UTC"},
		{"month end", time.Date(2026, 5, 31, 12, 0, 0, 0, west), "1", "31", "2026-05-01", "2026-05-31", false, "UTC"},
		{"leap month end", time.Date(2024, 2, 1, 12, 0, 0, 0, tokyo), "28", "29", "2024-02-28", "2024-02-29", false, "UTC"},
		{"nonleap month end", time.Date(2026, 2, 28, 12, 0, 0, 0, west), "1", "28", "2026-02-01", "2026-02-28", false, "UTC"},
		{"nonleap overflow", time.Date(2026, 2, 1, 12, 0, 0, 0, tokyo), "29", "29", "", "", true, "UTC"},
		{"zero", time.Date(2026, 5, 1, 12, 0, 0, 0, tokyo), "0", "1", "", "", true, "UTC"},
		{"next month", time.Date(2026, 5, 1, 12, 0, 0, 0, tokyo), "31", "32", "", "", true, "UTC"},
		{"same month next year", time.Date(2026, 5, 1, 12, 0, 0, 0, tokyo), "366", "366", "", "", true, "UTC"},
		{"signed day", time.Date(2026, 5, 1, 12, 0, 0, 0, tokyo), "+1", "2", "", "", true, "UTC"},
		{"huge day", time.Date(2026, 5, 1, 12, 0, 0, 0, tokyo), "99999999999999999999", "2", "", "", true, "UTC"},
		{"reversed", time.Date(2026, 5, 1, 12, 0, 0, 0, tokyo), "2", "1", "", "", true, "UTC"},
		{"Tokyo already in May", time.Date(2026, 4, 30, 16, 0, 0, 0, time.UTC), "1", "1", "2026-05-01", "2026-05-01", false, "Asia/Tokyo"},
		{"New York still in March", time.Date(2026, 4, 1, 2, 0, 0, 0, time.UTC), "8", "8", "2026-03-08", "2026-03-08", false, "America/New_York"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeClient{}
			client := &dateRangeClient{fakeClient: fake}
			app, _ := configuredApp(t, fake, "")
			app.NewClient = useClient(client)
			loc, err := time.LoadLocation(tt.zone)
			if err != nil {
				t.Fatal(err)
			}
			text, err := config.EncodeInitial("1", "test-token", tt.zone)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(config.Path(os.Getenv("HOME")), []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			app.Now = func() time.Time { return tt.now }
			err = app.Run(context.Background(), []string{"time-entry", "list", tt.start, tt.end})
			if tt.fail {
				var usageError UsageError
				if !errors.As(err, &usageError) || fake.entryCalls != 0 {
					t.Fatalf("expected usage error without fetching, got %v, calls=%d", err, fake.entryCalls)
				}
				return
			}
			wantStart, _ := time.ParseInLocation(time.DateOnly, tt.wantStart, loc)
			wantEnd, _ := time.ParseInLocation(time.DateOnly, tt.wantEnd, loc)
			if err != nil || !client.from.Equal(wantStart) || !client.to.Equal(wantEnd.AddDate(0, 0, 1)) {
				t.Fatalf("from=%s to=%s error=%v", client.from, client.to, err)
			}
		})
	}
}
