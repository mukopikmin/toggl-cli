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
)

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
				app.Client = client
				now, err := time.Parse(time.RFC3339, tt.now)
				if err != nil {
					t.Fatal(err)
				}
				app.Now = func() time.Time { return now }
				if err := os.WriteFile(config.Path(os.Getenv("HOME")), []byte("workspace=\"1\"\ntoken=\"secret\"\ntimezone=\""+tt.zone+"\"\n"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := app.Run(context.Background(), []string{"summary", "--month", month}); err != nil {
					t.Fatal(err)
				}
				if fake.entryCalls != 1 || client.from.Format(time.DateOnly) != tt.start || client.to.Format(time.DateOnly) != tt.end {
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
			app := App{Client: client, Out: &bytes.Buffer{}}
			err := app.Run(context.Background(), args)
			var usageError UsageError
			if !errors.As(err, &usageError) || client.entryCalls != 0 || client.projectCalls != 0 {
				t.Fatalf("expected usage error before configuration/API access, got %v", err)
			}
		})
	}
}
