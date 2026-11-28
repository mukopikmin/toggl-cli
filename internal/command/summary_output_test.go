package command

import (
	"strings"
	"testing"
	"time"
)

func TestSummaryDaysMonthEnds(t *testing.T) {
	a, _ := time.Parse(time.DateOnly, "2024-02-28")
	b, _ := time.Parse(time.DateOnly, "2024-03-01")
	got := summaryDays(a, b)
	if len(got) != 3 || got[1] != "2024-02-29" {
		t.Fatal(got)
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
			days := summaryDays(from, to)
			headers := append([]string{"Project"}, days...)
			output := table(headers, nil, tableLayout{maxWidth: tt.terminalWidth}, summaryHeaderVariants(days, from, to)...)
			if heading := strings.SplitN(output, "\n", 3)[1]; heading != tt.want {
				t.Fatalf("heading = %q, want %q", heading, tt.want)
			}
		})
	}
}
