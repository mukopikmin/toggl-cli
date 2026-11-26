package command

import (
	"strings"
	"testing"
)

func TestTableUnicodeAndMultiline(t *testing.T) {
	got := table([]string{"Name", "Value"}, [][]string{{"日本", "one\ntwo"}}, tableLayout{})
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
			lines := strings.Split(table(headers, rows, tableLayout{maxWidth: tt.terminal}, compact), "\n")
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
	got := table([]string{"Project"}, [][]string{{"開発作業"}}, tableLayout{maxWidth: 11}, []string{"Proj."})
	if !strings.Contains(got, "Project") || !strings.Contains(got, "開発作業") {
		t.Fatal(got)
	}
}

func TestTableTruncatesProjectNamesAfterShorteningDates(t *testing.T) {
	headers := []string{"Project", "2026-05-09"}
	rows := [][]string{{"開発作業", "1.25"}, {"P", "2"}}
	compact := []string{"Proj.", "05-09"}
	minimal := []string{"Proj.", "9"}
	for _, tt := range []struct {
		name                  string
		terminal, width       int
		wantProject, wantDate string
	}{
		{"unknown width", 0, 25, "開発作業", "2026-05-09"},
		{"wide terminal", 25, 25, "開発作業", "2026-05-09"},
		{"compact headings suffice", 20, 20, "開発作業", "05-09"},
		{"minimal headings suffice", 19, 19, "開発作業", "9"},
		{"truncate project name", 18, 18, "開発作…", "9"},
		{"preserve other columns below minimum width", 11, 12, "…", "9"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			layout := tableLayout{maxWidth: tt.terminal, truncateFirstColumn: true}
			lines := strings.Split(table(headers, rows, layout, compact, minimal), "\n")
			for _, line := range lines {
				if width := displayWidth(line); width != tt.width {
					t.Fatalf("line width = %d, want %d: %q", width, tt.width, line)
				}
			}
			if date := strings.TrimSpace(strings.Split(lines[1], "│")[2]); date != tt.wantDate {
				t.Errorf("date = %q, want %q", date, tt.wantDate)
			}
			data := strings.Split(lines[3], "│")
			if name := strings.TrimSpace(data[1]); name != tt.wantProject || strings.TrimSpace(data[2]) != "1.25" {
				t.Errorf("data row = %q, want %q and 1.25", lines[3], tt.wantProject)
			}
			if name := strings.TrimSpace(strings.Split(lines[4], "│")[1]); name != "P" {
				t.Errorf("short project name = %q, want P", name)
			}
		})
	}
}

func TestProjectTableShortensHeadingAndNameTogether(t *testing.T) {
	layout := tableLayout{maxWidth: 9, truncateFirstColumn: true}
	got := table([]string{"Project"}, [][]string{{"開発作業\ne\u0301quipe"}}, layout, []string{"Proj."})
	want := "┌───────┐\n│ Proj. │\n├───────┤\n│ 開発… │\n│ e\u0301qui… │\n└───────┘"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestTerminalTextEscapesControls(t *testing.T) {
	if got := terminalText("日本\x1b\u202e"); got != "日本\\x1b\\u202e" {
		t.Fatalf("%q", got)
	}
}
