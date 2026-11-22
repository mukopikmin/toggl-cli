package command

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
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
			if err := app.Run(context.Background(), []string{"summary", "--days", strconv.Itoa(tt.days)}); err != nil {
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

func TestSummaryTableDateHeadings(t *testing.T) {
	tests := []struct {
		name, start, end string
		terminalWidth    int
		want             string
	}{
		{"full dates fit", "2026-05-09", "2026-05-10", 37, "│ Project │ 2026-05-09 │ 2026-05-10 │"},
		{"month and day fit", "2026-05-09", "2026-05-10", 25, "│ Proj. │ 05-09 │ 05-10 │"},
		{"day only", "2026-05-09", "2026-05-10", 24, "│ Proj. │ 9 │ 10 │"},
		{"below minimum", "2026-05-09", "2026-05-10", 17, "│ Proj. │ 9 │ 10 │"},
		{"month boundary", "2026-04-30", "2026-05-02", 29, "│ Proj. │ 04-30 │ 05-01 │ 2 │"},
		{"year boundary", "2025-12-31", "2026-01-02", 39, "│ Proj. │ 2025-12-31 │ 2026-01-01 │ 2 │"},
		{"unknown width", "2026-05-09", "2026-05-10", 0, "│ Project │ 2026-05-09 │ 2026-05-10 │"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, err := time.Parse(time.DateOnly, tt.start)
			if err != nil {
				t.Fatal(err)
			}
			to, err := time.Parse(time.DateOnly, tt.end)
			if err != nil {
				t.Fatal(err)
			}
			days := model.DateRange(from, to)
			headers := append([]string{"Project"}, days...)
			output := table(headers, nil, tableLayout{maxWidth: tt.terminalWidth}, summaryHeaderVariants(days, from, to)...)
			if heading := strings.SplitN(output, "\n", 3)[1]; heading != tt.want {
				t.Fatalf("heading = %q, want %q", heading, tt.want)
			}
		})
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
