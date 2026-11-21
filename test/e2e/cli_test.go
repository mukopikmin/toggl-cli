//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTogglCLI(t *testing.T) {
	addrs, err := net.DefaultResolver.LookupHost(t.Context(), apiHost)
	if err != nil || len(addrs) != 1 || addrs[0] != "127.0.0.1" {
		t.Fatalf("run make test-e2e to map %s to loopback (got %v, %v)", apiHost, addrs, err)
	}
	caFile := startMockAPI(t)

	tests := []struct {
		name        string
		args        []string
		token       string
		caFile      string
		exitCode    int
		stdout      string
		stderrMatch string
	}{
		{
			name:   "client list",
			args:   []string{"client", "list", "--format", "json"},
			token:  fixtureToken,
			caFile: caFile,
			stdout: "[\n  {\n    \"id\": 2,\n    \"name\": \"Example Client\"\n  }\n]\n",
		},
		{
			name:   "summary across a month boundary in Japan",
			args:   []string{"summary", "2026-10-31", "2026-11-01", "--format", "csv"},
			token:  fixtureToken,
			caFile: caFile,
			stdout: "Project,2026-10-31,2026-11-01\nExample Project,30,60\n",
		},
		{
			name:        "unauthorized",
			args:        []string{"client", "list"},
			token:       "rejected-test-token",
			caFile:      caFile,
			exitCode:    1,
			stderrMatch: "Error: Failed to fetch clients: HTTP 401 Unauthorized",
		},
		{
			name:        "untrusted certificate",
			args:        []string{"client", "list"},
			token:       fixtureToken,
			caFile:      filepath.Join(t.TempDir(), "missing-ca.pem"),
			exitCode:    1,
			stderrMatch: "x509: certificate signed by unknown authority",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixtureHome := t.TempDir()
			configDir := filepath.Join(fixtureHome, ".config", "toggl-cli")
			if err := os.MkdirAll(configDir, 0700); err != nil {
				t.Fatal(err)
			}
			config := fmt.Sprintf("workspace = \"123\"\ntoken = %q\ntimezone = \"Asia/Tokyo\"\n", test.token)
			if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(config), 0600); err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "/toggl", test.args...)
			cmd.Env = []string{"HOME=" + fixtureHome, "SSL_CERT_FILE=" + test.caFile, "SSL_CERT_DIR=" + fixtureHome}
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			exitCode := 0
			if err := cmd.Run(); err != nil {
				var exitErr *exec.ExitError
				if ctx.Err() != nil || !errors.As(err, &exitErr) {
					t.Fatalf("run toggl %v: %v (context: %v, stderr: %s)", test.args, err, ctx.Err(), &stderr)
				}
				exitCode = exitErr.ExitCode()
			}
			if exitCode != test.exitCode {
				t.Errorf("exit code = %d, want %d (stderr: %s)", exitCode, test.exitCode, &stderr)
			}
			if got := stdout.String(); got != test.stdout {
				t.Errorf("stdout = %q, want %q", got, test.stdout)
			}
			if got := stderr.String(); (test.stderrMatch == "" && got != "") || !strings.Contains(got, test.stderrMatch) {
				t.Errorf("stderr = %q, want %q", got, test.stderrMatch)
			}
		})
	}
}
