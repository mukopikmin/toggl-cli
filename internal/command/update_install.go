package command

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func windowsUpdateScript(pid int, staged, executable string) string {
	quotedStage, quotedExecutable := strings.ReplaceAll(staged, "'", "''"), strings.ReplaceAll(executable, "'", "''")
	return fmt.Sprintf(`$ErrorActionPreference = "Stop"
$updated = $false
try {
  Wait-Process -Id %d -ErrorAction SilentlyContinue
  Move-Item -LiteralPath '%s' -Destination '%s' -Force
  $updated = $true
} finally {
  if (-not $updated) { Remove-Item -LiteralPath '%s' -Force -ErrorAction SilentlyContinue }
  Remove-Item -LiteralPath $PSCommandPath -Force -ErrorAction SilentlyContinue
}
`, pid, quotedStage, quotedExecutable, quotedStage)
}

type updateInstallRuntime struct {
	goos          string
	binaryVersion func(string) (string, error)
	startHelper   func(string) error
}

func installUpdate(ctx context.Context, p updatePlan, client *http.Client) error {
	return installUpdateWithRuntime(ctx, p, client, updateInstallRuntime{
		goos: runtime.GOOS,
		binaryVersion: func(path string) (string, error) {
			out, err := exec.Command(path, "--version").Output()
			return string(out), err
		},
		startHelper: func(path string) error {
			cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", path)
			if err := cmd.Start(); err != nil {
				return err
			}
			_ = cmd.Process.Release()
			return nil
		},
	})
}

func installUpdateWithRuntime(ctx context.Context, p updatePlan, client *http.Client, platform updateInstallRuntime) error {
	type result struct {
		data []byte
		err  error
	}
	archiveResult, checksumResult := make(chan result, 1), make(chan result, 1)
	go func() { b, e := download(ctx, client, p.DownloadURL); archiveResult <- result{b, e} }()
	go func() { b, e := download(ctx, client, p.DownloadURL+".sha256"); checksumResult <- result{b, e} }()
	a, c := <-archiveResult, <-checksumResult
	if a.err != nil || c.err != nil {
		return errors.New("Failed to download the update archive or checksum.")
	}
	archive, checksum := a.data, c.data
	var err error
	if err = verifyChecksum(archive, string(checksum)); err != nil {
		return err
	}
	binary, err := extractBinary(archive, p.Archive, p.Target)
	if err != nil {
		return err
	}
	dir := filepath.Dir(p.Executable)
	pattern := ".toggl-update-*"
	if platform.goos == "windows" {
		pattern += ".exe"
	}
	stage, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return err
	}
	stageName := stage.Name()
	defer func() {
		if stageName != "" {
			os.Remove(stageName)
		}
	}()
	if _, err = stage.Write(binary); err != nil {
		stage.Close()
		return err
	}
	if err = stage.Close(); err != nil {
		return err
	}
	if err = os.Chmod(stageName, 0755); err != nil {
		return err
	}
	out, err := platform.binaryVersion(stageName)
	if err != nil || strings.TrimSpace(out) != p.TargetVersion {
		return fmt.Errorf("Downloaded binary version mismatch (expected %s); the existing binary was not changed.", p.TargetVersion)
	}
	if platform.goos == "windows" {
		script := windowsUpdateScript(os.Getpid(), stageName, p.Executable)
		helper, err := os.CreateTemp(dir, ".toggl-update-*.ps1")
		if err != nil {
			return err
		}
		if _, err = helper.WriteString(script); err != nil {
			helper.Close()
			os.Remove(helper.Name())
			return err
		}
		if err = helper.Close(); err != nil {
			os.Remove(helper.Name())
			return err
		}
		if err = platform.startHelper(helper.Name()); err != nil {
			os.Remove(helper.Name())
			return err
		}
		stageName = ""
		return nil
	}
	return os.Rename(stageName, p.Executable)
}
