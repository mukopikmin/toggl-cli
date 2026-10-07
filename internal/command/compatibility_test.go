package command

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
)

func TestSummaryUsageBeforeConfiguration(t *testing.T) {
	tests := [][]string{
		{"summary"}, {"summary", "2026-05-01"},
		{"summary", "2026-05-01", "2026-05-02", "--days", "1"},
		{"summary", "2026-02-30", "2026-03-01"},
		{"summary", "2026-05-02", "2026-05-01"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			client := &fakeClient{}
			app := App{Client: client, Out: &bytes.Buffer{}}
			err := app.Run(context.Background(), args)
			var usageError UsageError
			if !errors.As(err, &usageError) || client.entryCalls != 0 || client.projectCalls != 0 {
				t.Fatalf("expected usage error before configuration/API access, got %v", err)
			}
		})
	}
}

func TestEmptyProjectJSONIsArray(t *testing.T) {
	tests := []struct {
		name     string
		projects []model.Project
	}{
		{"nil", nil}, {"empty", []model.Project{}},
		{"all hidden", []model.Project{{ID: 1, Hidden: true}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, out := configuredApp(t, &fakeClient{projects: tt.projects}, "")
			if err := app.Run(context.Background(), []string{"project", "list", "--format", "json"}); err != nil {
				t.Fatal(err)
			}
			if out.String() != "[]\n" {
				t.Fatalf("got %q, want an empty JSON array", out.String())
			}
		})
	}
}

type dateRangeClient struct {
	*fakeClient
	from, to time.Time
}

func (c *dateRangeClient) TimeEntries(ctx context.Context, cfg config.Config, from, to time.Time) ([]model.TimeEntry, error) {
	c.from, c.to = from, to
	return c.fakeClient.TimeEntries(ctx, cfg, from, to)
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
	}{
		{"month start", time.Date(2026, 5, 1, 0, 0, 0, 0, tokyo), "1", "1", "2026-05-01", "2026-05-01", false},
		{"month end", time.Date(2026, 5, 31, 23, 0, 0, 0, west), "1", "31", "2026-05-01", "2026-05-31", false},
		{"leap month end", time.Date(2024, 2, 1, 0, 0, 0, 0, tokyo), "28", "29", "2024-02-28", "2024-02-29", false},
		{"nonleap month end", time.Date(2026, 2, 28, 23, 0, 0, 0, west), "1", "28", "2026-02-01", "2026-02-28", false},
		{"nonleap overflow", time.Date(2026, 2, 1, 0, 0, 0, 0, tokyo), "29", "29", "", "", true},
		{"zero", time.Date(2026, 5, 1, 0, 0, 0, 0, tokyo), "0", "1", "", "", true},
		{"next month", time.Date(2026, 5, 1, 0, 0, 0, 0, tokyo), "31", "32", "", "", true},
		{"same month next year", time.Date(2026, 5, 1, 0, 0, 0, 0, tokyo), "366", "366", "", "", true},
		{"signed day", time.Date(2026, 5, 1, 0, 0, 0, 0, tokyo), "+1", "2", "", "", true},
		{"huge day", time.Date(2026, 5, 1, 0, 0, 0, 0, tokyo), "99999999999999999999", "2", "", "", true},
		{"reversed", time.Date(2026, 5, 1, 0, 0, 0, 0, tokyo), "2", "1", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeClient{}
			client := &dateRangeClient{fakeClient: fake}
			app, _ := configuredApp(t, fake, "")
			app.Client = client
			app.Now = func() time.Time { return tt.now }
			err := app.Run(context.Background(), []string{"time-entry", "list", tt.start, tt.end})
			if tt.fail {
				var usageError UsageError
				if !errors.As(err, &usageError) || fake.entryCalls != 0 {
					t.Fatalf("expected usage error without fetching, got %v, calls=%d", err, fake.entryCalls)
				}
				return
			}
			if err != nil || client.from.Format(time.DateOnly) != tt.wantStart || client.to.Format(time.DateOnly) != tt.wantEnd {
				t.Fatalf("from=%s to=%s error=%v", client.from, client.to, err)
			}
		})
	}
}

func TestInitTimezoneInputCompletion(t *testing.T) {
	tests := []struct {
		name, input, timezone string
		fail                  bool
	}{
		{"EOF after workspace", "workspace\n", "", true},
		{"EOF after token", "workspace\ntest-token\n", "", true},
		{"unterminated token", "workspace\ntest-token", "", true},
		{"blank timezone", "workspace\ntest-token\n\n", "Asia/Tokyo", false},
		{"partial timezone", "workspace\ntest-token\nUTC", "UTC", false},
		{"whitespace timezone", "workspace\ntest-token\n  ", "Asia/Tokyo", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			app := App{Out: &bytes.Buffer{}, In: strings.NewReader(tt.input)}
			err := app.Run(context.Background(), []string{"init"})
			if tt.fail {
				if err == nil {
					t.Fatal("expected incomplete-input error")
				}
				if _, statErr := os.Stat(config.Path(os.Getenv("HOME"))); !os.IsNotExist(statErr) {
					t.Fatalf("configuration was created: %v", statErr)
				}
				return
			}
			doc, loadErr := config.Load()
			if err != nil || loadErr != nil || doc.Config.Timezone != tt.timezone {
				t.Fatalf("init=%v load=%v timezone=%q", err, loadErr, doc.Config.Timezone)
			}
		})
	}
}

type failingInitReader struct{}

func (failingInitReader) Read([]byte) (int, error) { return 0, errors.New("input failure") }

func TestInitDoesNotDefaultAfterReadFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := App{Out: &bytes.Buffer{}, In: io.MultiReader(strings.NewReader("workspace\ntest-token\n"), failingInitReader{})}
	if err := app.Run(context.Background(), []string{"init"}); err == nil {
		t.Fatal("expected read error")
	}
	if _, err := os.Stat(config.Path(os.Getenv("HOME"))); !os.IsNotExist(err) {
		t.Fatalf("configuration was created: %v", err)
	}
}
