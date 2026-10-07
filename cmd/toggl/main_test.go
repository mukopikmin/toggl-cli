package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestExitCodesAndStreams(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		code           int
		stdout, stderr string
	}{{"help", nil, 0, "Usage:", ""}, {"version", []string{"--version"}, 0, "0.0.0-dev", ""}, {"usage", []string{"unknown"}, 1, "", "unknown command: unknown"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tt.args, &stdout, &stderr)
			if code != tt.code || !strings.Contains(stdout.String(), tt.stdout) || !strings.Contains(stderr.String(), tt.stderr) {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestMissingConfigErrorDoesNotPrintUsage(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var stdout, stderr bytes.Buffer
	if code := run([]string{"config"}, &stdout, &stderr); code != 1 {
		t.Fatal(code)
	}
	if !strings.Contains(stderr.String(), "Please create") || strings.Contains(stderr.String(), "Usage:") {
		t.Fatal(stderr.String())
	}
}

func TestSummaryUsageErrorWithoutConfigPrintsHelp(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var stdout, stderr bytes.Buffer
	if code := run([]string{"summary"}, &stdout, &stderr); code != 1 {
		t.Fatal(code)
	}
	if !strings.Contains(stderr.String(), "summary requires") || !strings.Contains(stderr.String(), "Usage:") || strings.Contains(stderr.String(), "Please create") {
		t.Fatal(stderr.String())
	}
}
