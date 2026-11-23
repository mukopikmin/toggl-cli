package command

import (
	"bytes"
	"github.com/mukopikmin/toggl-cli/internal/model"
	"strings"
	"testing"
)

func TestRenderReorderLineStarts(t *testing.T) {
	projects := []model.Project{
		{ID: 1, DisplayName: "First project"},
		{ID: 2, DisplayName: "Second project"},
	}
	tests := []struct {
		name  string
		state reorderState
		rows  []string
	}{
		{"empty", reorderState{}, nil},
		{"selected first", reorderState{Projects: projects}, []string{"> First project", "  Second project"}},
		{"picked last", reorderState{Projects: projects, Selected: 1, Moving: true}, []string{"  First project", "* Second project"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := renderReorder(tt.state)
			for i := range output {
				if output[i] == '\n' && (i == 0 || output[i-1] != '\r') {
					t.Fatalf("raw terminal newline at byte %d does not return to the line start: %q", i, output)
				}
			}
			lines := strings.Split(output, "\r\n")
			if len(lines) != 4+len(tt.rows) {
				t.Fatalf("got %d lines, want %d", len(lines), 4+len(tt.rows))
			}
			for i, want := range tt.rows {
				if got := lines[4+i]; got != want {
					t.Errorf("project row %d: got %q, want %q", i, got, want)
				}
			}
		})
	}
}

func TestUpdateReorderState(t *testing.T) {
	p := []model.Project{{ID: 1}, {ID: 2}, {ID: 3}}
	s := updateReorderState(reorderState{Projects: p, Selected: 1}, "move-up")
	if s.Selected != 0 || s.Projects[0].ID != 2 || p[0].ID != 1 {
		t.Fatalf("%+v", s)
	}
}

func TestSelectOrderEOFCancels(t *testing.T) {
	projects := []model.Project{{ID: 1, DisplayName: "one"}}
	got, saved, err := selectOrderLoop(projects, bytes.NewReader(nil), &bytes.Buffer{})
	if err != nil || saved || got != nil {
		t.Fatalf("got=%v saved=%v err=%v", got, saved, err)
	}
}
