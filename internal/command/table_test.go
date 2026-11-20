package command

import (
	"strings"
	"testing"
)

func TestTableUnicodeAndMultiline(t *testing.T) {
	got := table([]string{"Name", "Value"}, [][]string{{"日本", "one\ntwo"}}, 0)
	want := "┌──────┬───────┐\n│ Name │ Value │\n├──────┼───────┤\n│ 日本 │ one   │\n│      │ two   │\n└──────┴───────┘"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if strings.Contains(got, "+") {
		t.Fatal(got)
	}
}

func TestTableCompactHeaders(t *testing.T) {
	headers := []string{"Project", "2026-04-30", "2026-05-01"}
	compact := []string{"Proj.", "04-30", "05-01"}
	rows := [][]string{{"開発作業", "1.25", "2\n3"}}
	for _, tt := range []struct {
		name         string
		terminal     int
		wantWidth    int
		wantHeadings []string
	}{
		{"unknown width", 0, 38, headers},
		{"exact full width", 38, 38, headers},
		{"below full width", 37, 28, compact},
		{"exact compact width", 28, 28, compact},
		{"below compact width", 27, 28, compact},
	} {
		t.Run(tt.name, func(t *testing.T) {
			lines := strings.Split(table(headers, rows, tt.terminal, compact), "\n")
			if got := len(lines); got != 6 {
				t.Fatalf("got %d lines, want 6: %q", got, lines)
			}
			for _, line := range lines {
				if got := displayWidth(line); got != tt.wantWidth {
					t.Fatalf("line width = %d, want %d: %q", got, tt.wantWidth, line)
				}
			}
			for i, want := range tt.wantHeadings {
				if got := strings.TrimSpace(strings.Split(lines[1], "│")[i+1]); got != want {
					t.Errorf("heading %d = %q, want %q", i, got, want)
				}
			}
			for _, want := range []string{"開発作業", "1.25", "2", "3"} {
				if !strings.Contains(strings.Join(lines[3:5], "\n"), want) {
					t.Errorf("data cells are missing %q: %q", want, lines[3:5])
				}
			}
		})
	}
}

func TestTableUsesFullHeadersWhenAbbreviationCannotReduceWidth(t *testing.T) {
	got := table([]string{"Project"}, [][]string{{"開発作業"}}, 11, []string{"Proj."})
	if !strings.Contains(got, "Project") || !strings.Contains(got, "開発作業") {
		t.Fatal(got)
	}
}
