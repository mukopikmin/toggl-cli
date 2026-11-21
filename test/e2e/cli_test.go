//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestTogglCLI(t *testing.T) {
	addrs, err := net.DefaultResolver.LookupHost(t.Context(), apiHost)
	if err != nil || len(addrs) != 1 || addrs[0] != "127.0.0.1" {
		t.Fatalf("run make test-e2e to map %s to loopback (got %v, %v)", apiHost, addrs, err)
	}
	caFile, traffic := startMockAPI(t)
	listClients := []string{"client", "list", "--format", "json"}

	tests := []struct {
		name         string
		args         []string
		token        string
		caFile       string
		exitCode     int
		wantJSON     []map[string]any
		wantCSV      [][]string
		unauthorized bool
		rejectTLS    bool
	}{
		{
			name:     "client list",
			args:     listClients,
			token:    fixtureToken,
			caFile:   caFile,
			wantJSON: []map[string]any{{"id": float64(2), "name": "Example Client"}},
		},
		{
			name:   "summary across a month boundary in Japan",
			args:   []string{"summary", "2026-10-31", "2026-11-01", "--format", "csv"},
			token:  fixtureToken,
			caFile: caFile,
			wantCSV: [][]string{
				{"Project", "2026-10-31", "2026-11-01"},
				{"Example Project", "30", "60"},
			},
		},
		{
			name:         "unauthorized",
			args:         listClients,
			token:        "rejected-test-token",
			caFile:       caFile,
			exitCode:     1,
			unauthorized: true,
		},
		{
			name:      "untrusted certificate",
			args:      listClients,
			token:     fixtureToken,
			caFile:    filepath.Join(t.TempDir(), "missing-ca.pem"),
			exitCode:  1,
			rejectTLS: true,
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
			before := traffic.snapshot()
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
			switch {
			case test.wantJSON != nil:
				var got []map[string]any
				if err := json.Unmarshal(stdout.Bytes(), &got); err != nil || !reflect.DeepEqual(got, test.wantJSON) {
					t.Errorf("stdout = %q, want JSON %v (parse error: %v)", &stdout, test.wantJSON, err)
				}
			case test.wantCSV != nil:
				got, err := csv.NewReader(&stdout).ReadAll()
				if err != nil || !reflect.DeepEqual(got, test.wantCSV) {
					t.Errorf("CSV = %q, want %q (parse error: %v)", got, test.wantCSV, err)
				}
			default:
				if stdout.Len() != 0 {
					t.Errorf("stdout = %q, want empty on error", &stdout)
				}
			}
			if test.exitCode == 0 && stderr.Len() != 0 {
				t.Errorf("stderr = %q, want empty on success", &stderr)
			}
			if test.exitCode != 0 && strings.TrimSpace(stderr.String()) == "" {
				t.Error("expected a diagnostic on stderr")
			}

			after := traffic.snapshot()
			if test.unauthorized && after.unauthorized == before.unauthorized {
				t.Error("mock did not return HTTP 401 for a client list request")
			}
			if test.rejectTLS {
				if after.clientHellos == before.clientHellos {
					t.Error("mock did not observe a TLS connection attempt")
				}
				if after.requests != before.requests {
					t.Error("mock received an HTTP request with an untrusted certificate")
				}
			}
		})
	}
}
