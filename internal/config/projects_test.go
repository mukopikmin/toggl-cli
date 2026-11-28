package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

func TestUpdateProjectOrders(t *testing.T) {
	tests := []struct {
		name, input string
		ids         []int64
		contains    []string
	}{{"updates and preserves comments", "[projects.\"1\"]\n# x\ndisplay_order = 9 # keep\n[projects.2]\nhidden = false\n", []int64{2, 1}, []string{"display_order = 2 # keep", "display_order = 1"}}, {"appends missing with CRLF", "workspace = \"x\"\r\n", []int64{3}, []string{"[projects.\"3\"]\r\ndisplay_order = 1\r\n"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateProjectOrders(tt.input, tt.ids)
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Fatalf("%q lacks %q", got, want)
				}
			}
		})
	}
}

func TestSaveProjectOrderAcrossTOMLLayouts(t *testing.T) {
	for _, layout := range []string{
		"[projects . \"1\"]\n\"display_order\" = 9 # preserve\n",
		"projects = { \"1\" = { display_order = 9 } }\n",
		"projects.\"1\".display_order = 9\n",
	} {
		t.Run(layout, func(t *testing.T) {
			text := "workspace = \"1\"\ntoken = \"test-token\"\n[extra]\nkeep = true\n"
			// Root dotted and inline keys come before the extra table.
			text = strings.Replace(text, "[extra]", layout+"[extra]", 1)
			path := filepath.Join(t.TempDir(), "config.toml")
			if err := os.WriteFile(path, []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err := Parse(text)
			if err != nil {
				t.Fatal(err)
			}
			doc := Document{Path: path, Text: text, Settings: cfg}
			if err := doc.SaveProjectOrder([]int64{1}); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Parse(string(data))
			if err != nil || got.Projects[1].DisplayOrder == nil || *got.Projects[1].DisplayOrder != 1 {
				t.Fatalf("got %+v, error %v", got, err)
			}
			if !strings.Contains(string(data), "keep = true") {
				t.Fatal("unrelated key was lost")
			}
			if strings.Contains(layout, "# preserve") && !strings.Contains(string(data), "# preserve") {
				t.Fatal("inline comment was lost")
			}
		})
	}
}

func TestAddProjectsRejectsStaleDocument(t *testing.T) {
	original := "workspace = \"1\"\ntoken = \"test-token\"\ntimezone = \"UTC\"\n"
	edited := strings.Replace(original, "UTC", "Asia/Tokyo", 1)
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(edited), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Parse(original)
	if err != nil {
		t.Fatal(err)
	}
	doc := Document{Path: path, Text: original, Settings: cfg}
	if err := doc.AddProjects([]model.Project{{ID: 1, Name: "Project"}}); err == nil {
		t.Fatal("saved a stale document")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != edited {
		t.Fatalf("latest file changed, read error %v", err)
	}
}
