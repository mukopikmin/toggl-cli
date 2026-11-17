package command

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCompletion(t *testing.T) {
	for _, tc := range []struct{ name, shell, override, relative, notice string }{
		{"bash default", "bash", "", ".local/share/bash-completion/completions/toggl", "bash-completion"},
		{"bash XDG", "bash", "XDG_DATA_HOME", "custom/bash-completion/completions/toggl", "bash-completion"},
		{"bash override", "bash", "BASH_COMPLETION_USER_DIR", "custom/completions/toggl", "bash-completion"},
		{"fish default", "fish", "", ".config/fish/completions/toggl.fish", "new fish session"},
		{"fish XDG", "fish", "XDG_CONFIG_HOME", "custom/fish/completions/toggl.fish", "new fish session"},
		{"zsh fallback", "zsh", "", ".local/share/zsh/site-functions/_toggl", "No writable default"},
		{"zsh XDG", "zsh", "XDG_DATA_HOME", "custom/zsh/site-functions/_toggl", "No writable default"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := completionInstallHome(t)
			if tc.override != "" {
				t.Setenv(tc.override, filepath.Join(home, "custom"))
			}
			var out bytes.Buffer
			app := App{Out: &out}    // No config or API client is needed.
			for i := 0; i < 2; i++ { // Reinstalling the same completion is safe.
				out.Reset()
				if err := app.Run(context.Background(), []string{"completion", "install", tc.shell}); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(out.String(), tc.notice) {
					t.Fatal(out.String())
				}
			}
			path := filepath.Join(home, filepath.FromSlash(tc.relative))
			got, err := os.ReadFile(path)
			want, _ := completionScripts.ReadFile("completions/" + tc.shell)
			if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("file=%s err=%v", path, err)
			}
			if !strings.Contains(out.String(), path) {
				t.Fatal(out.String())
			}
			stages, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".toggl-completion-*"))
			if len(stages) != 0 {
				t.Fatalf("temporary files left: %v", stages)
			}
			for _, name := range []string{".bashrc", ".zshrc", ".config/fish/config.fish"} {
				b, _ := os.ReadFile(filepath.Join(home, name))
				if string(b) != "# user settings\n" {
					t.Fatalf("modified %s", name)
				}
			}
		})
	}
}

func completionInstallHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	for _, key := range []string{"XDG_DATA_HOME", "XDG_CONFIG_HOME", "BASH_COMPLETION_USER_DIR"} {
		t.Setenv(key, "")
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir()) // Never discover the machine's real zsh directories.
	for _, name := range []string{".bashrc", ".zshrc", ".config/fish/config.fish"} {
		path := filepath.Join(home, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("# user settings\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

func TestInstallZshDefaultPath(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("requires POSIX sh")
	}
	home := completionInstallHome(t)
	dir := filepath.Join(home, "site-functions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	// Include missing and non-site directories before the usable directory.
	t.Setenv("TEST_ZSH_PATHS", filepath.Join(home, "missing/site-functions")+"\n"+home+"\n"+dir)
	if err := os.WriteFile(filepath.Join(os.Getenv("PATH"), "zsh"), []byte("#!/bin/sh\nprintf '%s\\n' \"$TEST_ZSH_PATHS\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := (App{Out: &out}).Run(context.Background(), []string{"completion", "install", "zsh"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "_toggl")); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "No writable") || !strings.Contains(out.String(), "compinit") {
		t.Fatal(out.String())
	}
}

func TestInstallCompletionErrors(t *testing.T) {
	for _, args := range [][]string{{"install"}, {"install", "bash", "extra"}, {"install", "powershell"}, {"install", "../bash"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			completionInstallHome(t)
			var out bytes.Buffer
			if err := (App{Out: &out}).Run(context.Background(), append([]string{"completion"}, args...)); err == nil || out.Len() != 0 {
				t.Fatalf("err=%v output=%s", err, &out)
			}
		})
	}
	t.Run("missing HOME", func(t *testing.T) {
		completionInstallHome(t)
		t.Setenv("HOME", "")
		if err := (App{}).Run(context.Background(), []string{"completion", "install", "bash"}); err == nil {
			t.Fatal("expected HOME error")
		}
	})
	t.Run("blocked destination", func(t *testing.T) {
		home := completionInstallHome(t)
		blocked := filepath.Join(home, "blocked")
		if err := os.WriteFile(blocked, []byte("existing"), 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("BASH_COMPLETION_USER_DIR", blocked)
		var out bytes.Buffer
		if err := (App{Out: &out}).Run(context.Background(), []string{"completion", "install", "bash"}); err == nil || out.Len() != 0 {
			t.Fatalf("err=%v output=%s", err, &out)
		}
		if b, _ := os.ReadFile(blocked); string(b) != "existing" {
			t.Fatal("existing file changed")
		}
	})
}
