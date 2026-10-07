package command

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type updateTestTransport func(*http.Request) (*http.Response, error)

func (f updateTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestWindowsUpdateHandoffPreservesStagedBinary(t *testing.T) {
	root := "toggl-cli-v2.0.0-windows-x64"
	binary := []byte("new binary")
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	w, err := zw.Create(root + "/toggl.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(binary); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archive.Bytes())
	client := &http.Client{Transport: updateTestTransport(func(r *http.Request) (*http.Response, error) {
		body := archive.Bytes()
		if strings.HasSuffix(r.URL.Path, ".sha256") {
			body = []byte(hex.EncodeToString(sum[:]))
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body))}, nil
	})}
	tests := []struct {
		name, version         string
		versionErr, helperErr error
	}{
		{"successful handoff", "2.0.0", nil, nil},
		{"helper cannot start", "2.0.0", nil, errors.New("cannot start helper")},
		{"wrong binary version", "1.0.0", nil, nil},
		{"binary cannot run", "", errors.New("cannot probe binary"), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			executable := filepath.Join(dir, "toggl.exe")
			if err := os.WriteFile(executable, []byte("old binary"), 0755); err != nil {
				t.Fatal(err)
			}
			plan := updatePlan{TargetVersion: "2.0.0", Target: "windows-x64", Executable: executable, Archive: root + ".zip", DownloadURL: "https://update.test/archive"}
			var staged, helper string
			platform := updateInstallRuntime{
				goos: "windows",
				binaryVersion: func(path string) (string, error) {
					staged = path
					got, err := os.ReadFile(path)
					if err != nil || !bytes.Equal(got, binary) || filepath.Ext(path) != ".exe" {
						t.Fatalf("staged binary=%q error=%v path=%q", got, err, path)
					}
					return tt.version, tt.versionErr
				},
				startHelper: func(path string) error {
					helper = path
					text, err := os.ReadFile(path)
					if err != nil || !strings.Contains(string(text), "Wait-Process") || !strings.Contains(string(text), strings.ReplaceAll(staged, "'", "''")) {
						t.Fatalf("helper=%q error=%v", text, err)
					}
					return tt.helperErr
				},
			}
			err := installUpdateWithRuntime(context.Background(), plan, client, platform)
			success := tt.version == plan.TargetVersion && tt.versionErr == nil && tt.helperErr == nil
			if (err == nil) != success {
				t.Fatalf("error=%v, success=%v", err, success)
			}
			got, readErr := os.ReadFile(executable)
			if readErr != nil || string(got) != "old binary" {
				t.Fatalf("installed executable changed before process exit: %q, %v", got, readErr)
			}
			if success {
				got, readErr := os.ReadFile(staged)
				if readErr != nil || !bytes.Equal(got, binary) {
					t.Fatalf("staged binary was not preserved: %q, %v", got, readErr)
				}
				if _, err := os.Stat(helper); err != nil {
					t.Fatalf("helper was not preserved: %v", err)
				}
			} else {
				stages, _ := filepath.Glob(filepath.Join(dir, ".toggl-update-*"))
				if len(stages) != 0 {
					t.Fatalf("failed update left files behind: %v", stages)
				}
			}
		})
	}
}
