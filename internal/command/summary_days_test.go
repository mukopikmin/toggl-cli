package command

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/config"
)

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
			app.Client = client
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
			if err := app.Run(context.Background(), []string{"summary", "--days", strconv.Itoa(tt.days), "--format", "csv"}); err != nil {
				t.Fatal(err)
			}
			if gotStart, gotEnd := client.from.Format(time.RFC3339), client.to.Format(time.RFC3339); gotStart != tt.wantStart || gotEnd != tt.wantEnd {
				t.Fatalf("range = %s through %s, want %s through %s", gotStart, gotEnd, tt.wantStart, tt.wantEnd)
			}
			headers := strings.Split(strings.TrimSpace(out.String()), "\t")
			if len(headers) != tt.days+1 || headers[1] != tt.wantStart[:10] || headers[len(headers)-1] != tt.wantEnd[:10] {
				t.Fatalf("unexpected date columns: %q", out.String())
			}
		})
	}
}
