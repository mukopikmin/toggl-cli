package command

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func completionUpdateArchive(t *testing.T, script string, extra []*tar.Header) []byte {
	t.Helper()
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	files := map[string]string{"toggl": "new binary"}
	if script != "" {
		files["install-completions.sh"] = script
		for _, shell := range []string{"bash", "zsh", "fish"} {
			files["completions/"+shell] = "# " + shell
		}
	}
	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{Name: "release/" + name, Typeflag: tar.TypeReg, Mode: 0600, Size: int64(len(content))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	for _, h := range extra {
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestUpdateRefreshesCompletions(t *testing.T) {
	for _, tc := range []struct {
		name, script, version     string
		badChecksum, warning, ran bool
	}{
		{"success", `printf '%s' updated > "$TEST_COMPLETION_MARKER"`, "2.0.0", false, false, true},
		{"script failure preserves update", `exit 1`, "2.0.0", false, true, false},
		{"legacy archive", "", "2.0.0", false, false, false},
		{"checksum failure never runs script", `printf '%s' unsafe > "$TEST_COMPLETION_MARKER"`, "2.0.0", true, false, false},
		{"version failure never runs script", `printf '%s' unsafe > "$TEST_COMPLETION_MARKER"`, "1.0.0", false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			exe := filepath.Join(dir, "toggl")
			marker := filepath.Join(dir, "marker")
			t.Setenv("TEST_COMPLETION_MARKER", marker)
			if err := os.WriteFile(exe, []byte("old binary"), 0755); err != nil {
				t.Fatal(err)
			}
			archive := completionUpdateArchive(t, tc.script, nil)
			sum := fmt.Sprintf("%x", sha256.Sum256(archive))
			if tc.badChecksum {
				sum = strings.Repeat("0", 64)
			}
			client := &http.Client{Transport: updateTestTransport(func(r *http.Request) (*http.Response, error) {
				b := archive
				if strings.HasSuffix(r.URL.Path, ".sha256") {
					b = []byte(sum)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(b))}, nil
			})}
			var out, warnings bytes.Buffer
			plan := updatePlan{Archive: "release.tar.gz", Target: "linux-x64", TargetVersion: "2.0.0", Executable: exe, DownloadURL: "https://update.test/release"}
			err := installUpdateWithRuntime(context.Background(), plan, client, updateInstallRuntime{goos: "linux", binaryVersion: func(string) (string, error) { return tc.version, nil }, completionOut: &out, completionErr: &warnings})
			success := !tc.badChecksum && tc.version == "2.0.0"
			if (err == nil) != success {
				t.Fatalf("err=%v success=%v", err, success)
			}
			want := "old binary"
			if success {
				want = "new binary"
			}
			if b, _ := os.ReadFile(exe); string(b) != want {
				t.Fatalf("binary=%q", b)
			}
			_, statErr := os.Stat(marker)
			if (statErr == nil) != tc.ran {
				t.Fatalf("script ran=%v want=%v", statErr == nil, tc.ran)
			}
			if strings.Contains(warnings.String(), "completion installation failed") != tc.warning {
				t.Fatal(warnings.String())
			}
		})
	}
}

func TestUpdateRejectsUnsafeCompletionEntries(t *testing.T) {
	for _, h := range []*tar.Header{
		{Name: "release/install-completions.sh", Typeflag: tar.TypeSymlink, Linkname: "/tmp/other"},
		{Name: "release/completions/bash", Typeflag: tar.TypeReg, Size: 0},
	} {
		archive := completionUpdateArchive(t, "exit 0", []*tar.Header{h})
		if err := installUpdateCompletions(context.Background(), archive, "release.tar.gz", nil, nil); err == nil {
			t.Fatalf("accepted unsafe entry: %+v", h)
		}
	}
}
