package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompletionCandidates(t *testing.T) {
	tests := []struct {
		name  string
		words []string
		want  string
	}{
		{"command", []string{"pro"}, "project"},
		{"project verbs", []string{"project", ""}, "list reorder sync"},
		{"client verb", []string{"client", "l"}, "list"},
		{"time entry verb", []string{"time-entry", ""}, "list"},
		{"shells", []string{"completion", ""}, "bash zsh fish"},
		{"install removed", []string{"completion", "install", ""}, ""},
		{"installed shell has no arguments", []string{"completion", "install", "bash", ""}, ""},
		{"summary options", []string{"summary", "--"}, "--format --watch --days --month --no-project --no-date --clipboard"},
		{"project options", []string{"project", "list", "--"}, "--format --watch"},
		{"client options", []string{"client", "list", "--"}, "--format --watch"},
		{"entry options", []string{"time-entry", "list", "1", "2", "--"}, "--format --watch"},
		{"config options", []string{"config", "--"}, "--format --watch"},
		{"update options", []string{"update", "--"}, "--channel"},
		{"format values", []string{"summary", "--format", ""}, "csv tsv json table"},
		{"short format", []string{"project", "list", "-f", "j"}, "json"},
		{"attached format", []string{"config", "-fj"}, "-fjson"},
		{"assignment format", []string{"config", "--format=j"}, "--format=json"},
		{"channel values", []string{"update", "--channel", ""}, "stable nightly"},
		{"assignment channel", []string{"update", "--channel=n"}, "--channel=nightly"},
		{"consumed format", []string{"summary", "--format", "json", "--d"}, "--days"},
		{"removed separator", []string{"summary", "--separator", ""}, ""},
		{"arbitrary days", []string{"summary", "--days", ""}, ""},
		{"month option", []string{"summary", "--m"}, "--month"},
		{"arbitrary month", []string{"summary", "--month", ""}, ""},
		{"consumed month", []string{"summary", "--month", "2026/10", "--f"}, "--format"},
		{"consumed month assignment", []string{"summary", "--month=202610", "--w"}, "--watch"},
		{"end options", []string{"summary", "--", "--f"}, ""},
		{"invalid command", []string{"unknown", ""}, ""},
		{"invalid verb", []string{"project", "unknown", ""}, ""},
		{"sync has no options", []string{"project", "sync", ""}, ""},
		{"reorder has no options", []string{"project", "reorder", ""}, ""},
		{"init has no options", []string{"init", ""}, ""},
		{"shell has no arguments", []string{"completion", "bash", ""}, ""},
		{"foreign option", []string{"update", "--format=j"}, ""},
		{"literal input", []string{"$(touch /tmp/toggl-unexpected)"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := completionCandidates(tt.words); strings.Join(got, " ") != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCompletionWithoutConfiguration(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, shell := range []string{"bash", "zsh", "fish"} {
		var out bytes.Buffer
		app := App{Out: &out} // No client: completion must never call the API.
		if err := app.Run(context.Background(), []string{"completion", shell}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "__complete") {
			t.Fatal("missing completion protocol")
		}
	}
	for _, args := range [][]string{{"completion"}, {"completion", "powershell"}, {"completion", "bash", "extra"}} {
		var out bytes.Buffer
		err := (App{Out: &out}).Run(context.Background(), args)
		var usage UsageError
		if !errors.As(err, &usage) || out.Len() != 0 {
			t.Fatalf("args=%q err=%v output=%q", args, err, out.String())
		}
	}
}

func TestCompletionHelperProcess(t *testing.T) {
	if os.Getenv("TOGGL_COMPLETION_TEST_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			if err := (App{}).Run(context.Background(), os.Args[i+1:]); err != nil {
				os.Exit(1)
			}
			os.Exit(0)
		}
	}
	os.Exit(1)
}

func completionShellEnv(t *testing.T) []string {
	t.Helper()
	dir := t.TempDir()
	// Use a path with spaces to verify shell quoting and executable dispatch.
	bin := filepath.Join(dir, "test bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(bin, "toggl")
	if err := os.WriteFile(fixture, []byte("#!/bin/sh\nexec \"$TOGGL_TEST_EXECUTABLE\" -test.run=TestCompletionHelperProcess -- \"$@\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	testExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return append(os.Environ(), "TOGGL_COMPLETION_TEST_HELPER=1", "TOGGL_TEST_EXECUTABLE="+testExecutable, "TOGGL_FIXTURE="+fixture,
		"HOME="+dir, "XDG_CONFIG_HOME="+filepath.Join(dir, "config"), "XDG_DATA_HOME="+filepath.Join(dir, "data"),
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestBashCompletion(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not installed")
	}
	env := completionShellEnv(t)
	script, _ := completionScripts.ReadFile("completions/bash")
	tests := []struct{ words, want string }{
		{`"pro"`, "project"},
		{`"project" ""`, "list\nreorder\nsync"},
		{`"summary" "--format" "j"`, "json"},
		{`"summary" "--format" "=" "j"`, "json"},
		{`"summary" "--format" "=" ""`, "csv\ntsv\njson\ntable"},
		{`"summary" "--format" "=" "json" "--d"`, "--days"},
		{`"summary" "-fj"`, "-fjson"},
		{`"update" "--channel" "=" "n"`, "nightly"},
		{`"summary" "--" "--f"`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.words, func(t *testing.T) {
			code := string(script) + "\nCOMP_WORDS=(\"$TOGGL_FIXTURE\" " + tt.words + ")\nCOMP_CWORD=$((${#COMP_WORDS[@]}-1))\n_toggl\nprintf '%s\\n' \"${COMPREPLY[@]}\"\n"
			cmd := exec.Command(bash, "--noprofile", "--norc", "-c", code)
			cmd.Env = env
			out, err := cmd.CombinedOutput()
			if err != nil || strings.TrimSpace(string(out)) != tt.want {
				t.Fatalf("err=%v output=%q want=%q", err, out, tt.want)
			}
		})
	}
}

func TestZshCompletion(t *testing.T) {
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh is not installed")
	}
	dir := t.TempDir()
	script, _ := completionScripts.ReadFile("completions/zsh")
	path := filepath.Join(dir, "_toggl")
	if err := os.WriteFile(path, script, 0600); err != nil {
		t.Fatal(err)
	}
	env := append(completionShellEnv(t), "TOGGL_COMPLETION_SCRIPT="+path, "TOGGL_COMPLETION_DIR="+dir)
	for _, mode := range []string{"source", "autoload"} {
		for _, tt := range []struct{ words, want string }{
			{`"pro"`, "project"},
			{`"project" ""`, "list\nreorder\nsync"},
			{`"summary" "--format" "j"`, "json"},
			{`"summary" "--format=j"`, "--format=json"},
			{`"update" "--channel" "n"`, "nightly"},
		} {
			t.Run(mode+"/"+tt.words, func(t *testing.T) {
				setup := `autoload -Uz compinit; compinit -D; source "$TOGGL_COMPLETION_SCRIPT"`
				if mode == "autoload" {
					setup = `fpath=("$TOGGL_COMPLETION_DIR" $fpath); autoload -Uz compinit; compinit -D`
				}
				code := setup + `
compadd() { shift; printf '%s\n' "$@"; }
words=("$TOGGL_FIXTURE" ` + tt.words + `)
CURRENT=${#words}
PREFIX=${words[CURRENT]}
_toggl
`
				cmd := exec.Command(zsh, "-fc", code)
				cmd.Env = env
				out, err := cmd.CombinedOutput()
				if err != nil || strings.TrimSpace(string(out)) != tt.want {
					t.Fatalf("err=%v output=%q want=%q", err, out, tt.want)
				}
			})
		}
	}
}

func TestFishCompletion(t *testing.T) {
	fish, err := exec.LookPath("fish")
	if err != nil {
		t.Skip("fish is not installed")
	}
	script, _ := completionScripts.ReadFile("completions/fish")
	env := completionShellEnv(t)
	for _, tt := range []struct{ line, want string }{
		{"toggl pro", "project"},
		{"toggl project ", "list\nreorder\nsync"},
		{"toggl summary --format j", "json"},
		{"toggl summary --format=j", "--format=json"},
		{"toggl summary --f", "--format"},
		{"toggl update --channel=n", "--channel=nightly"},
		{"toggl summary -- --f", ""},
	} {
		t.Run(tt.line, func(t *testing.T) {
			cmd := exec.Command(fish, "--no-config", "-c", string(script)+"\ncomplete -C '"+tt.line+"'")
			cmd.Env = env
			out, err := cmd.CombinedOutput()
			if err != nil || strings.TrimSpace(string(out)) != tt.want {
				t.Fatalf("err=%v output=%q want=%q", err, out, tt.want)
			}
		})
	}
}
