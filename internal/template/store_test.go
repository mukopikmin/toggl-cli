package template

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

func TestDocumentRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	one, empty, separator := 1, "", "\"\\\t\n#"
	d := Document{Templates: map[string]model.SummaryOptions{
		"today":           {Days: &one, Format: "json", Clipboard: true},
		"fixed":           {StartDate: "2026-05-01", EndDate: "2026-05-31", Separator: &separator, NoDate: true},
		"empty-separator": {Separator: &empty, NoProject: true},
		"output-only":     {Format: "table"},
	}}
	if err := d.Write(); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil || !reflect.DeepEqual(d, got) {
		t.Fatalf("got %+v, %v; want %+v", got, err, d)
	}
	p, _ := path()
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("file permissions = %o", info.Mode().Perm())
	}
	if err := (Document{Templates: map[string]model.SummaryOptions{}}).Write(); err != nil {
		t.Fatal(err)
	}
	got, err = Load()
	if err != nil || len(got.Templates) != 0 {
		t.Fatalf("empty document = %+v, %v", got, err)
	}
	leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(p), ".templates-*"))
	if len(leftovers) != 0 {
		t.Fatalf("temporary files remain: %v", leftovers)
	}
}

func TestParseRejectsInvalidTemplates(t *testing.T) {
	for _, text := range []string{
		"[", "templates = []", "templates = 1", "[templates.bad]\ndays = '1'",
		"[templates.bad]\ndays = -1", "[templates.bad]\ndays = 0", "[templates.bad]\ndays = 9007199254740992",
		"[templates.bad]\nformat = 'xml'", "[templates.bad]\nformat = 'table'\nno_date = true",
		"[templates.bad]\nformat = 'table'\nseparator = ''",
		"[templates.bad]\nstart_date = '2026-05-01'",
		"[templates.bad]\nstart_date = '2026-02-30'\nend_date = '2026-03-01'",
		"[templates.bad]\nstart_date = '2026-05-02'\nend_date = '2026-05-01'",
		"[templates.bad]\ndays = 0\nstart_date = '2026-05-01'\nend_date = '2026-05-02'",
		"[templates.bad]\nunknown = true", "[other]", "[templates.'../escape']",
		"[templates.bad]\nclipboard = 'false'", "[templates.bad]\nformat = 'csv'\nformat = 'json'",
	} {
		t.Run(text, func(t *testing.T) {
			if _, err := Parse(text); err == nil {
				t.Fatal("accepted invalid document")
			}
		})
	}
}

func TestInvalidWritePreservesExistingFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	d := Document{Templates: map[string]model.SummaryOptions{"good": {Format: "table"}}}
	if err := d.Write(); err != nil {
		t.Fatal(err)
	}
	p, _ := path()
	before, _ := os.ReadFile(p)
	d.Templates["bad"] = model.SummaryOptions{Format: "invalid"}
	if err := d.Write(); err == nil {
		t.Fatal("invalid write succeeded")
	}
	after, _ := os.ReadFile(p)
	if !bytes.Equal(before, after) {
		t.Fatal("invalid write changed file")
	}
}

func TestLoadMissingHomeAndIOErrors(t *testing.T) {
	t.Setenv("HOME", "")
	if _, err := Load(); err == nil {
		t.Fatal("accepted missing HOME")
	}
	if err := (Document{}).Write(); err == nil {
		t.Fatal("write accepted missing HOME")
	}
	t.Setenv("HOME", t.TempDir())
	d, err := Load()
	if err != nil || len(d.Templates) != 0 {
		t.Fatalf("missing file: %+v, %v", d, err)
	}
	p, _ := path()
	if err := os.MkdirAll(p, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("directory read succeeded")
	}
	if err := (Document{}).Write(); err == nil {
		t.Fatal("replaced directory with file")
	}
	leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(p), ".templates-*"))
	if len(leftovers) != 0 {
		t.Fatal("failed write left temporary files")
	}
}
