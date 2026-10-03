package command

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestReleaseTarget(t *testing.T) {
	tests := []struct {
		os, arch, want string
		failed         bool
	}{{"linux", "amd64", "linux-x64", false}, {"darwin", "arm64", "darwin-arm64", false}, {"windows", "amd64", "windows-x64", false}, {"linux", "arm64", "", true}}
	for _, tt := range tests {
		t.Run(tt.os+"-"+tt.arch, func(t *testing.T) {
			got, err := releaseTarget(tt.os, tt.arch)
			if (err != nil) != tt.failed || got != tt.want {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}
}
func TestVersionAndArchiveCompatibility(t *testing.T) {
	v, err := nightlyVersion(time.Date(2026, 8, 24, 23, 54, 0, 0, time.UTC), "c1648a3b01234567890123456789012345678901")
	if err != nil || v != "nightly-20260824-c1648a3" {
		t.Fatalf("%q %v", v, err)
	}
	tests := []struct {
		channel, current, target string
		want                     bool
	}{{"stable", "1.2.3", "1.2.4", true}, {"stable", "2.0.0", "1.9.9", false}, {"stable", "1.2.3", "1.2.3", false}, {"nightly", "nightly-20260823-aaaaaaa", "nightly-20260824-bbbbbbb", true}}
	for _, tt := range tests {
		if got := updateIsNewer(tt.channel, tt.current, tt.target); got != tt.want {
			t.Errorf("%+v got %v", tt, got)
		}
	}
	if got := archiveName("nightly", v, "windows-x64"); got != "toggl-cli-nightly-windows-x64.zip" {
		t.Fatal(got)
	}
}

func TestUpdateArgumentValidation(t *testing.T) {
	app := App{Version: "0.0.0-dev", Out: &bytes.Buffer{}, In: bytes.NewBufferString("n\n")}
	tests := []struct {
		args  []string
		usage bool
	}{{[]string{"--channel", "invalid"}, true}, {[]string{"--channel=invalid"}, true}, {[]string{"extra"}, true}, {nil, false}}
	for _, tt := range tests {
		err := app.update(tt.args)
		var got UsageError
		if tt.usage {
			if !errors.As(err, &got) {
				t.Fatalf("%v: expected usage error, got %v", tt.args, err)
			}
		} else if err == nil || !strings.Contains(err.Error(), "running from source") {
			t.Fatalf("expected source-build error, got %v", err)
		}
	}
}
func TestVerifyChecksum(t *testing.T) {
	data := []byte("archive")
	sum := sha256.Sum256(data)
	if err := verifyChecksum(data, hex.EncodeToString(sum[:])); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksum(data, "bad"); err == nil {
		t.Fatal("expected invalid checksum error")
	}
	wrong := sha256.Sum256([]byte("other"))
	if err := verifyChecksum(data, hex.EncodeToString(wrong[:])); err == nil {
		t.Fatal("expected mismatch")
	}
}
func TestExtractTarBinary(t *testing.T) {
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	body := []byte("binary")
	_ = tw.WriteHeader(&tar.Header{Name: "toggl-cli-v1.0.0-linux-x64/toggl", Mode: 0755, Size: int64(len(body))})
	_, _ = tw.Write(body)
	tw.Close()
	gz.Close()
	got, err := extractBinary(archive.Bytes(), "toggl-cli-v1.0.0-linux-x64.tar.gz", "linux-x64")
	if err != nil || !bytes.Equal(got, body) {
		t.Fatalf("%q %v", got, err)
	}
}

func TestExtractZipBinary(t *testing.T) {
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	w, err := zw.Create("toggl-cli-v1.0.0-windows-x64/toggl.exe")
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("binary"))
	zw.Close()
	got, err := extractBinary(archive.Bytes(), "toggl-cli-v1.0.0-windows-x64.zip", "windows-x64")
	if err != nil || string(got) != "binary" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestDesiredVersionUsesGitHubResponses(t *testing.T) {
	tests := []struct{ name, channel, want string }{{"stable", "stable", "1.2.3"}, {"nightly", "nightly", "nightly-20260824-c1648a3"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/releases/latest":
					w.Write([]byte(`{"tag_name":"v1.2.3"}`))
				case "/git/ref/tags/nightly":
					w.Write([]byte(`{"object":{"sha":"c1648a3b01234567890123456789012345678901"}}`))
				case "/commits/c1648a3b01234567890123456789012345678901":
					w.Write([]byte(`{"commit":{"committer":{"date":"2026-08-24T23:54:06Z"}}}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			old := updateAPI
			updateAPI = server.URL
			t.Cleanup(func() { updateAPI = old })
			got, _, err := desiredVersion(context.Background(), server.Client(), tt.channel)
			if err != nil || got != tt.want {
				t.Fatalf("%q %v", got, err)
			}
		})
	}
}

func TestInstallUpdateVerifiesAndReplacesExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows uses a deferred replacement helper")
	}
	root := "toggl-cli-v2.0.0-linux-x64"
	binary := []byte("#!/bin/sh\nprintf '2.0.0\\n'\n")
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: root + "/toggl", Mode: 0755, Size: int64(len(binary))})
	_, _ = tw.Write(binary)
	tw.Close()
	gz.Close()
	sum := sha256.Sum256(archive.Bytes())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".sha256") {
			w.Write([]byte(hex.EncodeToString(sum[:])))
			return
		}
		w.Write(archive.Bytes())
	}))
	defer server.Close()
	exe := t.TempDir() + "/toggl"
	if err := os.WriteFile(exe, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	plan := updatePlan{Channel: "stable", CurrentVersion: "1.0.0", TargetVersion: "2.0.0", Target: "linux-x64", Executable: exe, Archive: root + ".tar.gz", DownloadURL: server.URL + "/archive"}
	if err := installUpdate(context.Background(), plan, server.Client()); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(exe)
	if err != nil || !bytes.Equal(got, binary) {
		t.Fatalf("%q %v", got, err)
	}
}

func TestWindowsUpdateScriptCleansUpAndQuotesPaths(t *testing.T) {
	script := windowsUpdateScript(42, `C:\O'Brien\stage.exe`, `C:\O'Brien\toggl.exe`)
	for _, want := range []string{`try {`, `finally {`, `C:\O''Brien\stage.exe`, `C:\O''Brien\toggl.exe`, `Remove-Item -LiteralPath $PSCommandPath`} {
		if !strings.Contains(script, want) {
			t.Fatalf("script lacks %q:\n%s", want, script)
		}
	}
}
