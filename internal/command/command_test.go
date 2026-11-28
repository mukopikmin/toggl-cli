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
	clients                  []model.Client
	clientCalls              int
	projects                 []model.Project
	entries                  []model.TimeEntry
	err                      error
	projectCalls, entryCalls int
}

func (f *fakeClient) Clients(context.Context) ([]model.Client, error) {
	f.clientCalls++
	return f.clients, f.err
}

func (f *fakeClient) Projects(context.Context) ([]model.Project, error) {
	f.projectCalls++
	return f.projects, f.err
}

func (f *fakeClient) TimeEntries(context.Context, time.Time, time.Time) ([]model.TimeEntry, error) {
	f.entryCalls++
	return f.entries, f.err
}

func useClient(client TogglClient) func(string, string) TogglClient {
	return func(string, string) TogglClient { return client }
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
	return App{NewClient: useClient(client), Out: out, Err: &bytes.Buffer{}, In: strings.NewReader(input), Now: func() time.Time { return time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC) }, Version: "1.2.3"}, out
}

func TestHelpVersionAndUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
		fail bool
	}{{"help", nil, "Usage:", false}, {"help default format", nil, "(default: table)", false}, {"version", []string{"--version"}, "1.2.3", false}, {"unknown", []string{"wat"}, "unknown command", true}, {"project needs verb", []string{"project"}, "requires a subcommand", true}, {"time entry needs verb", []string{"time-entry"}, "requires a subcommand", true}, {"invalid format", []string{"config", "--format", "xml"}, "format must", true}, {"foreign boolean", []string{"config", "--clipboard"}, "unknown option", true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			a := App{NewClient: useClient(&fakeClient{}), Out: &out, Version: "1.2.3"}
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

func appendProjectConfig(t *testing.T, text string) {
	t.Helper()
	file, err := os.OpenFile(config.Path(os.Getenv("HOME")), os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = file.WriteString(text)
	closeErr := file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}

type dateRangeClient struct {
	*fakeClient
	from, to time.Time
}

func (c *dateRangeClient) TimeEntries(ctx context.Context, from, to time.Time) ([]model.TimeEntry, error) {
	c.from, c.to = from, to
	return c.fakeClient.TimeEntries(ctx, from, to)
}
