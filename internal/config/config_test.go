package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name, text string
		ok         bool
	}{{"valid", "workspace = \"1\"\ntoken = \"secret\"\ntimezone = \"UTC\"\n[projects.\"2\"]\nhidden = true\ndisplay_order = 4\n", true}, {"hash in strings", "workspace = \"team#1\"\ntoken = \"abc#def\"\n[projects.\"2\"]\ndisplay_name = \"Client #2\"\n", true}, {"missing", "workspace = \"1\"", false}, {"invalid project", "workspace = \"1\"\ntoken = \"x\"\n[projects.bad]\nhidden = true", false}, {"wrong type", "workspace = 1\ntoken = \"x\"", false}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, e := Parse(tt.text)
			if (e == nil) != tt.ok {
				t.Fatalf("error=%v", e)
			}
			if tt.name == "hash in strings" && (c.Token != "abc#def" || c.Projects[2].DisplayName != "Client #2") {
				t.Fatalf("%+v", c)
			}
		})
	}
}
func TestLoadErrorsAndDocument(t *testing.T) {
	old := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", old) })
	os.Unsetenv("HOME")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "HOME") {
		t.Fatal(err)
	}
	home := t.TempDir()
	os.Setenv("HOME", home)
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatal(err)
	}
	path := Path(home)
	os.MkdirAll(filepath.Dir(path), 0700)
	text := "workspace=\"1\"\ntoken=\"x\"\n"
	os.WriteFile(path, []byte(text), 0600)
	doc, err := Load()
	if err != nil || doc.Text != text || doc.Path != path {
		t.Fatalf("%+v %v", doc, err)
	}
}

func TestEncodeInitialRoundTripsSpecialCharacters(t *testing.T) {
	text, err := EncodeInitial(`work#"space`, `tok\en#value`, `Asia/Tokyo`)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(text)
	if err != nil {
		t.Fatal(err)
	}
	if got.Workspace != `work#"space` || got.Token != `tok\en#value` {
		t.Fatalf("%+v\n%s", got, text)
	}
}

func TestProjectsMustBeTable(t *testing.T) {
	tests := []struct {
		name, text string
		valid      bool
	}{
		{"omitted", "", true}, {"empty inline table", "projects = {}\n", true},
		{"empty table", "[projects]\n", true},
		{"project table", "[projects.1]\nhidden=false\n", true},
		{"empty array", "projects = []\n", false},
		{"array of tables", "[[projects]]\nhidden=false\n", false},
		{"string", "projects = \"invalid\"\n", false},
		{"project scalar", "[projects]\n1=\"invalid\"\n", false},
		{"project array", "[projects]\n1=[]\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse("workspace=\"1\"\ntoken=\"test-token\"\n" + tt.text)
			if (err == nil) != tt.valid {
				t.Fatalf("error=%v, valid=%v", err, tt.valid)
			}
		})
	}
}
