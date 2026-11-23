package selfupdate

import (
	"archive/tar"
	"archive/zip"
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

const updateRepository = "https://github.com/mukopikmin/toggl-cli"
const updateAPI = "https://api.github.com/repos/mukopikmin/toggl-cli"

// Service checks GitHub releases and installs verified updates.
type Service struct {
	Client *http.Client
}

func (s Service) httpClient() *http.Client {
	if s.Client != nil {
		return s.Client
	}
	return http.DefaultClient
}

type Plan struct {
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
func desiredVersion(ctx context.Context, client *http.Client, api, channel string) (version, tag string, err error) {
	if channel == "stable" {
		var x struct {
			Tag string `json:"tag_name"`
		}
		if err = getJSON(ctx, client, api+"/releases/latest", &x); err != nil {
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
	if err = getJSON(ctx, client, api+"/git/ref/tags/nightly", &ref); err != nil {
		return
	}
	var commit struct {
		Commit struct {
			Committer struct {
				Date time.Time `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}
	if err = getJSON(ctx, client, api+"/commits/"+ref.Object.SHA, &commit); err != nil {
		return
	}
	version, err = nightlyVersion(commit.Commit.Committer.Date, ref.Object.SHA)
	return version, "nightly", err
}
func (s Service) Check(ctx context.Context, current, channel string) (Plan, error) {
	if err := ctx.Err(); err != nil {
		return Plan{}, err
	}
	if channel == "" {
		channel = defaultChannel(current)
	}
	target, err := releaseTarget(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return Plan{}, err
	}
	validVersion := stableVersion.MatchString(current) || current == "nightly" || regexp.MustCompile(`^nightly-\d{8}-[0-9a-fA-F]{7,40}$`).MatchString(current)
	if current == "0.0.0-dev" {
		return Plan{}, errors.New("Self-update is unavailable when running from source. Install a compiled toggl binary first.")
	}
	if !validVersion {
		return Plan{}, errors.New("The running executable does not report a valid Toggl CLI version.")
	}
	exe, err := os.Executable()
	if err != nil {
		return Plan{}, err
	}
	out, err := exec.CommandContext(ctx, exe, "--version").Output()
	if ctx.Err() != nil {
		return Plan{}, ctx.Err()
	}
	if err != nil || strings.TrimSpace(string(out)) != current {
		return Plan{}, fmt.Errorf("The running executable is not the expected compiled Toggl CLI binary: %s", exe)
	}
	version, tag, err := desiredVersion(ctx, s.httpClient(), updateAPI, channel)
	if err != nil {
		return Plan{}, err
	}
	archive := archiveName(channel, version, target)
	return Plan{channel, current, version, target, exe, archive, updateRepository + "/releases/download/" + tag + "/" + archive, updateIsNewer(channel, current, version)}, nil
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
