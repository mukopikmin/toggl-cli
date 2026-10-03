package command

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var updateRepository = "https://github.com/mukopikmin/toggl-cli"
var updateAPI = "https://api.github.com/repos/mukopikmin/toggl-cli"

type updatePlan struct {
	Channel, CurrentVersion, TargetVersion, Target, Executable, Archive, DownloadURL string
	Available                                                                        bool
}

var stableVersion = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:[-+][0-9A-Za-z.-]+)?$`)
var nightlyVersionPattern = regexp.MustCompile(`^nightly-(\d{8})-([0-9a-fA-F]{7})$`)

func isNightly(v string) bool { return v == "nightly" || nightlyVersionPattern.MatchString(v) }
func defaultChannel(v string) string {
	if isNightly(v) {
		return "nightly"
	}
	return "stable"
}
func releaseTarget(goos, goarch string) (string, error) {
	switch goos + "/" + goarch {
	case "linux/amd64":
		return "linux-x64", nil
	case "darwin/arm64":
		return "darwin-arm64", nil
	case "windows/amd64":
		return "windows-x64", nil
	}
	return "", fmt.Errorf("Self-update is not supported on %s/%s.", goos, goarch)
}
func archiveName(channel, version, target string) string {
	ext := "tar.gz"
	if target == "windows-x64" {
		ext = "zip"
	}
	if channel == "nightly" {
		return "toggl-cli-nightly-" + target + "." + ext
	}
	return "toggl-cli-v" + version + "-" + target + "." + ext
}
func updateIsNewer(channel, current, target string) bool {
	if current == target {
		return false
	}
	if channel == "nightly" {
		a, b := nightlyVersionPattern.FindStringSubmatch(current), nightlyVersionPattern.FindStringSubmatch(target)
		if len(a) > 0 && len(b) > 0 && a[1] != b[1] {
			return b[1] > a[1]
		}
		return true
	}
	a, b := stableVersion.FindStringSubmatch(current), stableVersion.FindStringSubmatch(target)
	if len(a) == 0 || len(b) == 0 {
		return true
	}
	for i := 1; i <= 3; i++ {
		x, _ := strconv.Atoi(a[i])
		y, _ := strconv.Atoi(b[i])
		if x != y {
			return y > x
		}
	}
	return strings.Contains(current, "-") && !strings.Contains(target, "-")
}
func nightlyVersion(timestamp time.Time, sha string) (string, error) {
	if len(sha) < 7 || !regexp.MustCompile(`^[0-9a-fA-F]{7,40}$`).MatchString(sha) {
		return "", errors.New("Invalid commit SHA.")
	}
	return "nightly-" + timestamp.UTC().Format("20060102") + "-" + sha[:7], nil
}

func getJSON(ctx context.Context, client *http.Client, url string, out any) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "toggl-cli-update")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("GitHub request failed (%d)", res.StatusCode)
	}
	if err = json.NewDecoder(res.Body).Decode(out); err != nil {
		return errors.New("GitHub returned invalid JSON")
	}
	return nil
}
func desiredVersion(ctx context.Context, client *http.Client, channel string) (version, tag string, err error) {
	if channel == "stable" {
		var x struct {
			Tag string `json:"tag_name"`
		}
		if err = getJSON(ctx, client, updateAPI+"/releases/latest", &x); err != nil {
			return
		}
		if !strings.HasPrefix(x.Tag, "v") || !stableVersion.MatchString(x.Tag[1:]) {
			err = errors.New("GitHub returned an invalid latest release tag")
			return
		}
		return x.Tag[1:], x.Tag, nil
	}
	var ref struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err = getJSON(ctx, client, updateAPI+"/git/ref/tags/nightly", &ref); err != nil {
		return
	}
	var commit struct {
		Commit struct {
			Committer struct {
				Date time.Time `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}
	if err = getJSON(ctx, client, updateAPI+"/commits/"+ref.Object.SHA, &commit); err != nil {
		return
	}
	version, err = nightlyVersion(commit.Commit.Committer.Date, ref.Object.SHA)
	return version, "nightly", err
}
func checkUpdate(ctx context.Context, current, channel string, client *http.Client) (updatePlan, error) {
	if channel == "" {
		channel = defaultChannel(current)
	}
	target, err := releaseTarget(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return updatePlan{}, err
	}
	validVersion := stableVersion.MatchString(current) || current == "nightly" || regexp.MustCompile(`^nightly-\d{8}-[0-9a-fA-F]{7,40}$`).MatchString(current)
	if current == "0.0.0-dev" {
		return updatePlan{}, errors.New("Self-update is unavailable when running from source. Install a compiled toggl binary first.")
	}
	if !validVersion {
		return updatePlan{}, errors.New("The running executable does not report a valid Toggl CLI version.")
	}
	exe, err := os.Executable()
	if err != nil {
		return updatePlan{}, err
	}
	out, err := exec.Command(exe, "--version").Output()
	if err != nil || strings.TrimSpace(string(out)) != current {
		return updatePlan{}, fmt.Errorf("The running executable is not the expected compiled Toggl CLI binary: %s", exe)
	}
	version, tag, err := desiredVersion(ctx, client, channel)
	if err != nil {
		return updatePlan{}, err
	}
	archive := archiveName(channel, version, target)
	return updatePlan{channel, current, version, target, exe, archive, updateRepository + "/releases/download/" + tag + "/" + archive, updateIsNewer(channel, current, version)}, nil
}
func download(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return nil, fmt.Errorf("download failed (%d)", res.StatusCode)
	}
	return io.ReadAll(res.Body)
}
func verifyChecksum(data []byte, text string) error {
	want := strings.TrimSpace(text)
	if len(want) != 64 {
		return errors.New("Release checksum is not exactly one 64-digit hexadecimal SHA-256 value.")
	}
	if _, e := hex.DecodeString(want); e != nil {
		return errors.New("Release checksum is not exactly one 64-digit hexadecimal SHA-256 value.")
	}
	got := sha256.Sum256(data)
	if hex.EncodeToString(got[:]) != strings.ToLower(want) {
		return errors.New("Update checksum mismatch; the existing binary was not changed.")
	}
	return nil
}
func extractBinary(archive []byte, name, target string) ([]byte, error) {
	root := strings.TrimSuffix(name, ".zip")
	root = strings.TrimSuffix(strings.TrimSuffix(root, ".gz"), ".tar")
	binary := root + "/toggl"
	if target == "windows-x64" {
		binary += ".exe"
		zr, e := zip.NewReader(strings.NewReader(string(archive)), int64(len(archive)))
		if e != nil {
			return nil, e
		}
		for _, f := range zr.File {
			if filepath.ToSlash(f.Name) == binary {
				r, e := f.Open()
				if e != nil {
					return nil, e
				}
				defer r.Close()
				return io.ReadAll(r)
			}
		}
	} else {
		gz, e := gzip.NewReader(strings.NewReader(string(archive)))
		if e != nil {
			return nil, e
		}
		defer gz.Close()
		tr := tar.NewReader(gz)
		for {
			h, e := tr.Next()
			if e == io.EOF {
				break
			}
			if e != nil {
				return nil, e
			}
			if filepath.ToSlash(h.Name) == binary {
				return io.ReadAll(tr)
			}
		}
	}
	return nil, errors.New("Release archive does not contain the expected Toggl CLI binary.")
}

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

func installUpdate(ctx context.Context, p updatePlan, client *http.Client) error {
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
	if runtime.GOOS == "windows" {
		pattern += ".exe"
	}
	stage, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return err
	}
	stageName := stage.Name()
	defer os.Remove(stageName)
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
	out, err := exec.Command(stageName, "--version").Output()
	if err != nil || strings.TrimSpace(string(out)) != p.TargetVersion {
		return fmt.Errorf("Downloaded binary version mismatch (expected %s); the existing binary was not changed.", p.TargetVersion)
	}
	if runtime.GOOS == "windows" {
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
		cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", helper.Name())
		if err = cmd.Start(); err != nil {
			os.Remove(helper.Name())
			return err
		}
		_ = cmd.Process.Release()
		stageName = ""
		return nil
	}
	return os.Rename(stageName, p.Executable)
}

func (a App) update(args []string) error {
	channel := ""
	if len(args) == 1 && strings.HasPrefix(args[0], "--channel=") {
		channel = strings.TrimPrefix(args[0], "--channel=")
	} else if len(args) == 2 && args[0] == "--channel" {
		channel = args[1]
	} else if len(args) == 1 && args[0] == "--channel" {
		return usage("option --channel requires a value")
	} else if len(args) > 0 {
		if strings.HasPrefix(args[0], "-") {
			return usagef("unknown option: %s", args[0])
		}
		return usage("update does not accept positional arguments")
	}
	if channel != "" && channel != "stable" && channel != "nightly" {
		return usage("channel must be stable or nightly")
	}
	client := http.DefaultClient
	plan, err := checkUpdate(context.Background(), a.Version, channel, client)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "Current version: %s\nUpdate channel: %s\nAvailable version: %s\n", plan.CurrentVersion, plan.Channel, plan.TargetVersion)
	if !plan.Available {
		fmt.Fprintln(a.Out, "Already up to date; no update was installed.")
		return nil
	}
	fmt.Fprint(a.Out, "Download and install this update? [y/N] ")
	line, _ := bufio.NewReader(a.In).ReadString('\n')
	if strings.ToLower(strings.TrimSpace(line)) != "y" {
		fmt.Fprintln(a.Out, "Update cancelled.")
		return nil
	}
	if err = installUpdate(context.Background(), plan, client); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "Installed version %s.\n", plan.TargetVersion)
	return nil
}
