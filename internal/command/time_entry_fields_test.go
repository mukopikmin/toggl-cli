package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

func TestTimeEntryListSelectedFields(t *testing.T) {
	projectID := int64(8)
	stop := "2026-05-01T01:00:00Z"
	start := "2026-05-01T00:00:00Z"
	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	entries := []model.TimeEntry{
		{ID: 2, ProjectID: &projectID, Description: "Finished, entry", Start: start, Stop: &stop, DurationSeconds: 3600},
		{ID: 1, Description: "Running", Start: start, DurationSeconds: 90 - now.Unix()},
	}
	for _, tt := range []struct {
		format, fields, want string
	}{
		{"table", "duration_minutes,id", table([]string{"duration_minutes", "id"}, [][]string{{"1.5", "1"}, {"60", "2"}}, tableLayout{}) + "\n"},
		{"csv", "stop", "stop\n\"\"\n" + stop + "\n"},
		{"tsv", "description, project_id", "description\tproject_id\nRunning\t\nFinished, entry\t8\n"},
		{"json", "stop,project_id,duration_minutes", `[
			{"stop": null, "project_id": null, "duration_minutes": 1.5},
			{"stop": "2026-05-01T01:00:00Z", "project_id": 8, "duration_minutes": 60}
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
			app := App{Client: client, Out: &bytes.Buffer{}}
			err := app.Run(context.Background(), []string{"time-entry", "list", "1", "2", "--fields", tt.field})
			var usageErr UsageError
			if !errors.As(err, &usageErr) || !strings.Contains(err.Error(), tt.want) || client.entryCalls != 0 {
				t.Fatalf("expected usage error containing %q before configuration and API access, got %v", tt.want, err)
			}
		})
	}
}
