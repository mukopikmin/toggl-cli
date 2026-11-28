package command

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/mukopikmin/toggl-cli/internal/config"
)

func TestInitCreatesProtectedConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	out := &bytes.Buffer{}
	app := App{NewClient: useClient(&fakeClient{}), Out: out, In: strings.NewReader("workspace\ntoken\nUTC\n")}
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
	app := App{NewClient: useClient(&fakeClient{}), Out: &bytes.Buffer{}, In: strings.NewReader("workspace\n")}
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
	app := App{NewClient: useClient(&fakeClient{}), Out: &bytes.Buffer{}, In: strings.NewReader("\n")}
	err := app.Run(context.Background(), []string{"init"})
	if err == nil || err.Error() != "Workspace is required" {
		t.Fatalf("%v", err)
	}
}

func TestInitTimezoneInputCompletion(t *testing.T) {
	tests := []struct {
		name, input, timezone string
		fail                  bool
	}{
		{"EOF after workspace", "workspace\n", "", true},
		{"EOF after token", "workspace\ntest-token\n", "", true},
		{"unterminated token", "workspace\ntest-token", "", true},
		{"blank timezone", "workspace\ntest-token\n\n", "Asia/Tokyo", false},
		{"partial timezone", "workspace\ntest-token\nUTC", "UTC", false},
		{"whitespace timezone", "workspace\ntest-token\n  ", "Asia/Tokyo", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			app := App{Out: &bytes.Buffer{}, In: strings.NewReader(tt.input)}
			err := app.Run(context.Background(), []string{"init"})
			if tt.fail {
				if err == nil {
					t.Fatal("expected incomplete-input error")
				}
				if _, statErr := os.Stat(config.Path(os.Getenv("HOME"))); !os.IsNotExist(statErr) {
					t.Fatalf("configuration was created: %v", statErr)
				}
				return
			}
			doc, loadErr := config.Load()
			if err != nil || loadErr != nil || doc.Settings.Timezone != tt.timezone {
				t.Fatalf("init=%v load=%v timezone=%q", err, loadErr, doc.Settings.Timezone)
			}
		})
	}
}

type failingInitReader struct{}

func (failingInitReader) Read([]byte) (int, error) { return 0, errors.New("input failure") }

func TestInitDoesNotDefaultAfterReadFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	app := App{Out: &bytes.Buffer{}, In: io.MultiReader(strings.NewReader("workspace\ntest-token\n"), failingInitReader{})}
	if err := app.Run(context.Background(), []string{"init"}); err == nil {
		t.Fatal("expected read error")
	}
	if _, err := os.Stat(config.Path(os.Getenv("HOME"))); !os.IsNotExist(err) {
		t.Fatalf("configuration was created: %v", err)
	}
}
