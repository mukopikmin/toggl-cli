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
)

type fakeClient struct {
	projects                 []model.Project
	entries                  []model.TimeEntry
	err                      error
	projectCalls, entryCalls int
}

func (f *fakeClient) Projects(context.Context, config.Config) ([]model.Project, error) {
	f.projectCalls++
	return f.projects, f.err
}
func (f *fakeClient) TimeEntries(context.Context, config.Config, time.Time, time.Time) ([]model.TimeEntry, error) {
	f.entryCalls++
	return f.entries, f.err
}
func configuredApp(t *testing.T, client *fakeClient, input string) (App, *bytes.Buffer) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := config.Path(home)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("workspace=\"1\"\ntoken=\"secret\"\ntimezone=\"UTC\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	return App{Client: client, Out: out, Err: &bytes.Buffer{}, In: strings.NewReader(input), Now: func() time.Time { return time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC) }, Version: "1.2.3"}, out
}
func TestHelpVersionAndUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
		fail bool
	}{{"help", nil, "Usage:", false}, {"version", []string{"--version"}, "1.2.3", false}, {"unknown", []string{"wat"}, "unknown command", true}, {"project needs verb", []string{"project"}, "requires a subcommand", true}, {"time entry needs verb", []string{"time-entry"}, "requires a subcommand", true}, {"invalid format", []string{"config", "--format", "xml"}, "format must", true}, {"foreign boolean", []string{"config", "--clipboard"}, "unknown option", true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			a := App{Client: &fakeClient{}, Out: &out, Version: "1.2.3"}
			err := a.Run(context.Background(), tt.args)
			if (err != nil) != tt.fail {
				t.Fatalf("%v", err)
			}
			text := out.String()
			if err != nil {
				text = err.Error()
				var usage UsageError
				if !errors.As(err, &usage) {
					t.Fatalf("expected UsageError, got %T", err)
				}
			}
			if !strings.Contains(text, tt.want) {
				t.Fatalf("%q", text)
			}
		})
	}
}
func TestConfigFormatsHideToken(t *testing.T) {
	for _, format := range []string{"csv", "json", "table"} {
		t.Run(format, func(t *testing.T) {
			app, out := configuredApp(t, &fakeClient{}, "")
			if err := app.Run(context.Background(), []string{"config", "--format", format}); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out.String(), "secret") || !strings.Contains(out.String(), "WORKSPACE") {
				t.Fatal(out.String())
			}
		})
	}
}
func TestProjectListAndSync(t *testing.T) {
	client := &fakeClient{projects: []model.Project{{ID: 2, Name: "Hidden", DisplayName: "Hidden", Hidden: true}, {ID: 1, Name: "Client", DisplayName: "Shown", Active: true}}}
	app, out := configuredApp(t, client, "")
	if err := app.Run(context.Background(), []string{"project", "list"}); err != nil {
		t.Fatal(err)
	}
	if out.String() != "Shown\n" {
		t.Fatal(out.String())
	}
	out.Reset()
	if err := app.Run(context.Background(), []string{"project", "sync"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(config.Path(os.Getenv("HOME")))
	if !strings.Contains(string(b), "[projects.\"1\"]") || !strings.Contains(string(b), "[projects.\"2\"]") {
		t.Fatal(string(b))
	}
}

func TestEmptyProjectListWritesNewline(t *testing.T) {
	app, out := configuredApp(t, &fakeClient{}, "")
	if err := app.Run(context.Background(), []string{"project", "list"}); err != nil {
		t.Fatal(err)
	}
	if out.String() != "\n" {
		t.Fatalf("%q", out.String())
	}
}
func TestSummaryFormatsAndBoundaries(t *testing.T) {
	pid := int64(1)
	client := &fakeClient{projects: []model.Project{{ID: 1, Name: "P", DisplayName: "P", Active: true}}, entries: []model.TimeEntry{{ID: 1, ProjectID: &pid, Start: "2026-05-01T23:30:00Z", DurationSeconds: 90}}}
	tests := []struct {
		name string
		args []string
		want string
	}{{"csv", []string{"summary", "2026-05-01", "2026-05-02"}, "Project\t2026-05-01\t2026-05-02\nP\t1.5\t"}, {"json", []string{"summary", "--days", "1", "--format", "json"}, `"2026-05-01"`}, {"no headers", []string{"summary", "2026-05-01", "2026-05-02", "--no-project", "--no-date"}, "1.5\t"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, out := configuredApp(t, client, "")
			if err := app.Run(context.Background(), tt.args); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), tt.want) {
				t.Fatalf("%q", out.String())
			}
		})
	}
	app, _ := configuredApp(t, client, "")
	if err := app.Run(context.Background(), []string{"summary", "2026-05-01", "2026-05-02", "--format", "table", "--separator", ","}); err == nil {
		t.Fatal("expected table/separator error")
	}
}

func TestSummaryAllowsEmptySeparator(t *testing.T) {
	client := &fakeClient{projects: []model.Project{{ID: 1, DisplayName: "P"}}}
	app, out := configuredApp(t, client, "")
	if err := app.Run(context.Background(), []string{"summary", "2026-05-01", "2026-05-01", "--separator", ""}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Project2026-05-01") {
		t.Fatal(out.String())
	}
}
func TestTimeEntryFormats(t *testing.T) {
	pid := int64(8)
	stop := "2026-05-01T01:00:00Z"
	client := &fakeClient{entries: []model.TimeEntry{{ID: 2, ProjectID: &pid, Description: "a,b", Start: "2026-05-01T00:00:00Z", Stop: &stop, DurationSeconds: 3600}}}
	for _, format := range []string{"csv", "json", "table"} {
		t.Run(format, func(t *testing.T) {
			app, out := configuredApp(t, client, "")
			if err := app.Run(context.Background(), []string{"time-entry", "list", "1", "2", "--format", format, "--separator", func() string {
				if format == "table" {
					return ""
				}
				return ","
			}()}); format == "table" {
				if err == nil {
					t.Fatal("expected separator error")
				}
				return
			} else if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "a,b") {
				t.Fatal(out.String())
			}
		})
	}
}

func TestTimeEntryRejectsEmptySeparator(t *testing.T) {
	app, _ := configuredApp(t, &fakeClient{}, "")
	err := app.Run(context.Background(), []string{"time-entry", "list", "1", "2", "--separator", ""})
	if err == nil || err.Error() != "separator must not be empty" {
		t.Fatalf("%v", err)
	}
}
func TestInitCreatesProtectedConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	out := &bytes.Buffer{}
	app := App{Client: &fakeClient{}, Out: out, In: strings.NewReader("workspace\ntoken\nUTC\n")}
	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(config.Path(home))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	before, _ := os.ReadFile(config.Path(home))
	app.In = strings.NewReader("replacement\nreplacement\nUTC\n")
	if err := app.Run(context.Background(), []string{"init"}); err == nil {
		t.Fatal("expected existing config error")
	}
	after, _ := os.ReadFile(config.Path(home))
	if !bytes.Equal(before, after) {
		t.Fatal("existing config was overwritten")
	}
}

func TestInitRejectsIncompleteInputWithoutCreatingFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	app := App{Client: &fakeClient{}, Out: &bytes.Buffer{}, In: strings.NewReader("workspace\n")}
	if err := app.Run(context.Background(), []string{"init"}); err == nil || !strings.Contains(err.Error(), "API token") {
		t.Fatal(err)
	}
	if _, err := os.Stat(config.Path(home)); !os.IsNotExist(err) {
		t.Fatalf("config should not exist: %v", err)
	}
}

func TestInitRejectsEmptyNonInteractiveRequiredValue(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	app := App{Client: &fakeClient{}, Out: &bytes.Buffer{}, In: strings.NewReader("\n")}
	err := app.Run(context.Background(), []string{"init"})
	if err == nil || err.Error() != "Workspace is required" {
		t.Fatalf("%v", err)
	}
}

func TestClipboardUnavailable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if err := clipboard("text"); err == nil || err.Error() != "Could not copy output to the clipboard." {
		t.Fatal(err)
	}
}

func TestClipboardCommandParityAndFallback(t *testing.T) {
	windows := clipboardCommands("windows")
	if len(windows) != 3 || windows[0].name != "clip" || windows[1].name != "powershell.exe" || windows[2].name != "powershell" {
		t.Fatalf("%+v", windows)
	}
	linux := clipboardCommands("linux")
	attempts := []string{}
	err := writeClipboard("summary output", linux, func(name string) (string, error) { return name, nil }, func(command clipboardCommand, text string) error {
		attempts = append(attempts, command.name+":"+text)
		if command.name == "xclip" {
			return nil
		}
		return errors.New("failed")
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(attempts, ",") != "wl-copy:summary output,xclip:summary output" {
		t.Fatalf("%v", attempts)
	}
}
