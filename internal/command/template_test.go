package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
	"github.com/mukopikmin/toggl-cli/internal/template"
)

func TestTemplateLifecycleWithoutConfigurationOrAPI(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	client := &fakeClient{err: errors.New("must not call API")}
	var out bytes.Buffer
	app := App{Client: client, Out: &out}
	run := func(args ...string) string {
		t.Helper()
		out.Reset()
		if err := app.Run(context.Background(), args); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	if got := run("template", "list"); got != "" {
		t.Fatalf("empty list = %q", got)
	}
	run("template", "add", "weekly", "--days", "6", "--format", "table")
	run("template", "add", "spreadsheet", "--no-date", "--no-project")
	if got := run("template", "list"); got != "spreadsheet\nweekly\n" {
		t.Fatal(got)
	}
	shown := run("template", "show", "weekly")
	doc, err := template.Parse(shown)
	if err != nil || len(doc.Templates) != 1 || doc.Templates["weekly"].Format != "table" || *doc.Templates["weekly"].Days != 6 {
		t.Fatalf("shown = %q, error = %v", shown, err)
	}
	p := filepath.Join(home, ".config", "toggl-cli", "templates.toml")
	before, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Run(context.Background(), []string{"template", "add", "weekly", "--days", "1"}); err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("duplicate = %v", err)
	}
	after, _ := os.ReadFile(p)
	if !bytes.Equal(before, after) {
		t.Fatal("duplicate add changed file")
	}
	run("template", "add", "weekly", "--days", "1", "--force")
	doc, err = template.Load()
	if err != nil || *doc.Templates["weekly"].Days != 1 || doc.Templates["weekly"].Format != "csv" {
		t.Fatalf("force did not replace the template: %+v, %v", doc, err)
	}
	if !doc.Templates["spreadsheet"].NoDate {
		t.Fatal("force changed unrelated template")
	}
	run("template", "delete", "weekly")
	if got := run("template", "list"); got != "spreadsheet\n" {
		t.Fatal(got)
	}
	if client.entryCalls != 0 || client.projectCalls != 0 {
		t.Fatal("template management called API")
	}
	if _, err := os.Stat(config.Path(home)); !os.IsNotExist(err) {
		t.Fatalf("template management created config: %v", err)
	}
}

func TestTemplateUsageErrorsDoNotWrite(t *testing.T) {
	tests := [][]string{
		{"template"}, {"template", "save", "old"}, {"template", "add"},
		{"template", "add", "../escape"}, {"template", "add", ""},
		{"template", "add", "weekly", "2026-05-01"},
		{"template", "add", "weekly", "2026-02-30", "2026-03-01"},
		{"template", "add", "weekly", "2026-05-02", "2026-05-01"},
		{"template", "add", "weekly", "2026-05-01", "2026-05-02", "--days", "1"},
		{"template", "add", "weekly", "--days=-1"},
		{"template", "add", "weekly", "--days", "0"},
		{"summary", "--days", "0"},
		{"template", "add", "weekly", "--format", "table", "--separator", ","},
		{"template", "add", "weekly", "--format", "table", "--no-date"},
		{"template", "add", "weekly", "--template", "other"},
		{"template", "list", "extra"}, {"template", "list", "--force"},
		{"template", "show"}, {"template", "show", "a", "b"},
		{"template", "delete"}, {"template", "delete", "a", "--force"},
		{"summary", "--template="}, {"summary", "--template", "../escape"},
		{"summary", "--days", "1", "--force"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			app := App{Out: &bytes.Buffer{}}
			err := app.Run(context.Background(), args)
			var usageError UsageError
			if !errors.As(err, &usageError) {
				t.Fatalf("expected usage error, got %v", err)
			}
			if _, err := os.Stat(filepath.Join(home, ".config")); !os.IsNotExist(err) {
				t.Fatalf("invalid usage created files: %v", err)
			}
		})
	}
}

func TestUnknownTemplateDoesNotUseAPI(t *testing.T) {
	for _, args := range [][]string{
		{"summary", "--template", "missing"},
		{"template", "show", "missing"}, {"template", "delete", "missing"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			app := App{Out: &bytes.Buffer{}}
			if err := app.Run(context.Background(), args); err == nil || !strings.Contains(err.Error(), "unknown template") {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestSummaryTemplateMatchesDirectExecution(t *testing.T) {
	tests := []struct {
		name                        string
		settings, overrides, direct []string
	}{
		{"table", []string{"--days", "1", "-ftable"}, nil, []string{"--days", "1", "-ftable"}},
		{"format override", []string{"--days", "6", "-ftable"}, []string{"--days", "1", "-fcsv"}, []string{"--days", "1", "-fcsv"}},
		{"fixed replaces relative", []string{"--days", "6"}, []string{"2026-05-01", "2026-05-02"}, []string{"2026-05-01", "2026-05-02"}},
		{"relative replaces fixed", []string{"2026-04-01", "2026-04-30"}, []string{"--days", "1"}, []string{"--days", "1"}},
		{"output only", []string{"--no-project", "--no-date"}, []string{"2026-05-01", "2026-05-02"}, []string{"2026-05-01", "2026-05-02", "--no-project", "--no-date"}},
		{"clear booleans", []string{"--days", "1", "--no-project", "--no-date", "--clipboard"}, []string{"--project", "--date", "--no-clipboard"}, []string{"--days", "1"}},
		{"empty separator", []string{"--days", "1", "--separator", ""}, nil, []string{"--days", "1", "--separator", ""}},
		{"override separator", []string{"--days", "1", "--separator", ","}, []string{"--separator", "|"}, []string{"--days", "1", "--separator", "|"}},
		{"json", []string{"--days", "1", "--format", "json"}, nil, []string{"--days", "1", "--format", "json"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectID := int64(1)
			fake := &fakeClient{
				projects: []model.Project{{ID: projectID, DisplayName: "Client A", Active: true}},
				entries:  []model.TimeEntry{{ID: 1, ProjectID: &projectID, Start: "2026-05-02T01:00:00Z", DurationSeconds: 5400}},
			}
			app, out := configuredApp(t, fake, "")
			client := &dateRangeClient{fakeClient: fake}
			app.Client = client
			configBefore, _ := os.ReadFile(config.Path(os.Getenv("HOME")))
			if err := app.Run(context.Background(), append([]string{"template", "add", "test"}, tt.settings...)); err != nil {
				t.Fatal(err)
			}
			templatePath := filepath.Join(os.Getenv("HOME"), ".config", "toggl-cli", "templates.toml")
			before, _ := os.ReadFile(templatePath)
			out.Reset()
			if err := app.Run(context.Background(), append([]string{"summary", "--template", "test"}, tt.overrides...)); err != nil {
				t.Fatal(err)
			}
			got, from, to := out.String(), client.from, client.to
			out.Reset()
			if err := app.Run(context.Background(), append([]string{"summary"}, tt.direct...)); err != nil {
				t.Fatal(err)
			}
			if got != out.String() || !from.Equal(client.from) || !to.Equal(client.to) {
				t.Fatalf("template = %q [%s, %s], direct = %q [%s, %s]", got, from, to, out.String(), client.from, client.to)
			}
			after, _ := os.ReadFile(templatePath)
			configAfter, _ := os.ReadFile(config.Path(os.Getenv("HOME")))
			if !bytes.Equal(before, after) || !bytes.Equal(configBefore, configAfter) {
				t.Fatal("execution changed saved templates or configuration")
			}
		})
	}
}

func TestCorruptTemplatesAreNotOverwritten(t *testing.T) {
	for _, args := range [][]string{
		{"template", "add", "weekly", "--days", "1", "--force"},
		{"template", "list"}, {"template", "show", "weekly"},
		{"template", "delete", "weekly"}, {"summary", "--template", "weekly"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			fake := &fakeClient{}
			app, _ := configuredApp(t, fake, "")
			p := filepath.Join(os.Getenv("HOME"), ".config", "toggl-cli", "templates.toml")
			before := []byte("[templates.weekly]\ndays = 'invalid'\n")
			if err := os.WriteFile(p, before, 0600); err != nil {
				t.Fatal(err)
			}
			if err := app.Run(context.Background(), args); err == nil {
				t.Fatal("accepted corrupt templates")
			}
			after, err := os.ReadFile(p)
			if err != nil || !bytes.Equal(before, after) || fake.entryCalls != 0 || fake.projectCalls != 0 {
				t.Fatalf("corrupt file was changed or API called: %v", err)
			}
			if err := app.Run(context.Background(), []string{"summary", "--days", "1"}); err != nil {
				t.Fatalf("ordinary summary should not load templates: %v", err)
			}
		})
	}
}

func TestSummaryValidatesMergedTemplateBeforeAPI(t *testing.T) {
	tests := []struct{ settings, overrides []string }{
		{[]string{"--days", "1", "--no-date"}, []string{"--format", "table"}},
		{[]string{"--days", "1", "--separator", ""}, []string{"--format", "table"}},
		{[]string{"--format", "table"}, nil},
		{[]string{"--days", "1"}, []string{"2026-05-01"}},
		{[]string{"--days", "1"}, []string{"2026-05-01", "2026-05-02", "--days", "1"}},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.overrides, " "), func(t *testing.T) {
			fake := &fakeClient{}
			app, _ := configuredApp(t, fake, "")
			if err := app.Run(context.Background(), append([]string{"template", "add", "test"}, tt.settings...)); err != nil {
				t.Fatal(err)
			}
			err := app.Run(context.Background(), append([]string{"summary", "--template", "test"}, tt.overrides...))
			var usageError UsageError
			if !errors.As(err, &usageError) || fake.entryCalls != 0 || fake.projectCalls != 0 {
				t.Fatalf("expected validation before API, got %v", err)
			}
		})
	}
}

func TestSummaryTemplateResolvesDaysAtExecution(t *testing.T) {
	fake := &fakeClient{}
	app, out := configuredApp(t, fake, "")
	client := &dateRangeClient{fakeClient: fake}
	app.Client = client
	p := config.Path(os.Getenv("HOME"))
	if err := os.WriteFile(p, []byte("workspace='1'\ntoken='test'\ntimezone='Asia/Tokyo'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := app.Run(context.Background(), []string{"template", "add", "rolling", "--days", "2", "--format", "json"}); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct{ now, from, to string }{
		{"2026-04-30T15:30:00Z", "2026-04-30", "2026-05-01"},
		{"2026-05-31T14:30:00Z", "2026-05-30", "2026-05-31"},
		{"2024-02-29T15:30:00Z", "2024-02-29", "2024-03-01"},
	} {
		now, _ := time.Parse(time.RFC3339, tt.now)
		app.Now = func() time.Time { return now }
		out.Reset()
		if err := app.Run(context.Background(), []string{"summary", "--template", "rolling"}); err != nil {
			t.Fatal(err)
		}
		if client.from.Format(time.DateOnly) != tt.from || client.to.Format(time.DateOnly) != tt.to {
			t.Fatalf("at %s: got %s to %s", tt.now, client.from, client.to)
		}
		_, offset := client.to.Zone()
		if offset != 9*60*60 {
			t.Fatal("did not use configured timezone")
		}
	}
}
