package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture() map[string]string {
	return map[string]string{
		"git rev-parse HEAD": "head",
		"gh repo view --json nameWithOwner --jq .nameWithOwner": "owner/repo",
		"gh api --paginate repos/owner/repo/releases?per_page=100 --jq .[] | {tagName: .tag_name, isDraft: .draft, isPrerelease: .prerelease}": `{"tagName":"nightly","isPrerelease":true}
{"tagName":"v1.0.0"}`,
		"git fetch --no-tags origin main":                          "",
		"git rev-parse --verify origin/main":                       "head",
		"git rev-parse --verify refs/tags/v1.1.0":                  "tag-object",
		"git rev-parse --verify refs/tags/v1.1.0^{commit}":         "head",
		"git rev-parse --verify refs/tags/v1.0.0^{commit}":         "previous",
		"git ls-remote --exit-code --tags origin refs/tags/v1.1.0": "tag-object\trefs/tags/v1.1.0",
		"git merge-base --is-ancestor previous head":               "",
		"gh run list --repo owner/repo --workflow test.yml --branch main --event push --commit head --limit 1 --json conclusion,headSha,url":    `[{"conclusion":"success","headSha":"head","url":"https://github.com/owner/repo/actions/runs/1"}]`,
		"gh api --method POST repos/owner/repo/releases/generate-notes -f tag_name=v1.1.0 -f target_commitish=head -f previous_tag_name=v1.0.0": `{"name":"Release 1.1.0","body":"Changes since v1.0.0"}`,
	}
}

func fakeRunner(t *testing.T, responses map[string]string) runner {
	t.Helper()
	return func(command string, args ...string) (string, error) {
		key := command + " " + strings.Join(args, " ")
		value, ok := responses[key]
		if !ok {
			return "", errors.New("command failed: " + key)
		}
		return value, nil
	}
}

func TestVersionAndBaselineSelection(t *testing.T) {
	for _, version := range []string{"0.0.0", "1.2.3", "999999999999999999999999999999.0.0"} {
		if !stableVersion.MatchString(version) {
			t.Errorf("rejected stable version %q", version)
		}
	}
	for _, version := range []string{"", "v1.0.0", "01.0.0", "1.01.0", "1.0.01", "1.0", "1.0.0-rc.1", "1.0.0+meta", "1.0.0\n"} {
		if stableVersion.MatchString(version) {
			t.Errorf("accepted invalid version %q", version)
		}
	}
	for _, tc := range []struct {
		next, previous string
		want           bool
	}{
		{"0.1.0", baselineTag, true}, {"1.0.0", "v1.0.0", false},
		{"1.0.0", "v1.0.1", false}, {"1.0.9", "v1.0.10", false},
		{"1.0.10", "v1.0.9", true}, {"2.0.0", "v1.99.99", true},
		{"99999999999999999999999.0.0", "v99999999999999999999998.9.9", true},
	} {
		if got := newerVersion(tc.next, tc.previous); got != tc.want {
			t.Errorf("newerVersion(%q, %q) = %t", tc.next, tc.previous, got)
		}
	}
	releases := []release{{TagName: "nightly"}, {TagName: "v0.0.0-nightly"}, {TagName: "v3.0.0", IsDraft: true}, {TagName: "v2.0.0", IsPrerelease: true}, {TagName: "v01.0.0"}}
	if got := previousStableTag(releases); got != baselineTag {
		t.Fatalf("baseline = %q", got)
	}
	releases = append(releases, release{TagName: "v1.0.0"})
	if got := previousStableTag(releases); got != "v1.0.0" {
		t.Fatalf("stable baseline = %q", got)
	}
}

func TestInspectRelease(t *testing.T) {
	for _, tc := range []struct {
		name, command, response, failedCheck string
	}{
		{name: "valid"},
		{"main advanced", "git rev-parse --verify origin/main", "new-head", "remoteMain"},
		{"tag targets another commit", "git rev-parse --verify refs/tags/v1.1.0^{commit}", "other", "releaseTag"},
		{"tag moved remotely", "git ls-remote --exit-code --tags origin refs/tags/v1.1.0", "new-object\trefs/tags/v1.1.0", "releaseTag"},
		{"draft already exists", "gh api --paginate repos/owner/repo/releases?per_page=100 --jq .[] | {tagName: .tag_name, isDraft: .draft, isPrerelease: .prerelease}", "{\"tagName\":\"v1.1.0\",\"isDraft\":true}\n{\"tagName\":\"v1.0.0\"}", "releaseAbsent"},
		{"no intervening changes", "git rev-parse --verify refs/tags/v1.0.0^{commit}", "head", "notesBaseline"},
		{"missing CI", "gh run list --repo owner/repo --workflow test.yml --branch main --event push --commit head --limit 1 --json conclusion,headSha,url", "[]", "ci"},
		{"failed CI", "gh run list --repo owner/repo --workflow test.yml --branch main --event push --commit head --limit 1 --json conclusion,headSha,url", `[{"conclusion":"failure","headSha":"head","url":"run"}]`, "ci"},
		{"wrong CI commit", "gh run list --repo owner/repo --workflow test.yml --branch main --event push --commit head --limit 1 --json conclusion,headSha,url", `[{"conclusion":"success","headSha":"other","url":"run"}]`, "ci"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			responses := fixture()
			if tc.command != "" {
				responses[tc.command] = tc.response
			}
			result, err := inspectRelease("1.1.0", fakeRunner(t, responses))
			if err != nil {
				t.Fatal(err)
			}
			if result.OK != (tc.failedCheck == "") {
				t.Fatalf("unexpected validation: %+v", result)
			}
			for _, check := range result.Checks {
				if check.Name == tc.failedCheck && check.OK {
					t.Errorf("%s should fail", check.Name)
				}
			}
			data, _ := json.Marshal(result)
			if !bytes.Contains(data, []byte(`"headSha":"head"`)) || !bytes.Contains(data, []byte(`"previousStableTag":"v1.0.0"`)) {
				t.Fatalf("missing workflow outputs: %s", data)
			}
		})
	}
}

func TestFirstReleaseBaseline(t *testing.T) {
	for _, root := range []string{"previous", "wrong-root"} {
		responses := fixture()
		responses["gh api --paginate repos/owner/repo/releases?per_page=100 --jq .[] | {tagName: .tag_name, isDraft: .draft, isPrerelease: .prerelease}"] = ""
		responses["git rev-parse --verify refs/tags/"+baselineTag+"^{commit}"] = "previous"
		responses["git rev-list --max-parents=0 HEAD"] = root
		result, err := inspectRelease("1.1.0", fakeRunner(t, responses))
		if err != nil || result.OK != (root == "previous") {
			t.Fatalf("root %s: %+v, %v", root, result, err)
		}
	}
}

func TestInspectionFailsClosed(t *testing.T) {
	for command := range fixture() {
		if strings.Contains(command, "generate-notes") {
			continue
		}
		t.Run(command, func(t *testing.T) {
			responses := fixture()
			delete(responses, command)
			result, err := inspectRelease("1.1.0", fakeRunner(t, responses))
			if err == nil && result.OK {
				t.Fatal("command failure allowed release")
			}
		})
	}
	responses := fixture()
	responses["gh api --paginate repos/owner/repo/releases?per_page=100 --jq .[] | {tagName: .tag_name, isDraft: .draft, isPrerelease: .prerelease}"] = "invalid json"
	if _, err := inspectRelease("1.1.0", fakeRunner(t, responses)); err == nil {
		t.Fatal("accepted malformed API response")
	}
}

func TestNotesAndCLI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.json")
	var stdout bytes.Buffer
	if err := executeCLI([]string{"notes", "--version", "1.1.0", "--output", path}, fakeRunner(t, fixture()), &stdout); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var result notes
	if err := json.Unmarshal(data, &result); err != nil || result.Title != "Release 1.1.0" || result.Body != "Changes since v1.0.0" || result.PreviousStableTag != "v1.0.0" || result.HeadSHA != "head" {
		t.Fatalf("unexpected notes %s: %v", data, err)
	}
	for command := range fixture() {
		if !strings.Contains(command, "generate-notes") {
			continue
		}
		for _, response := range []string{`{"body":" "}`, `invalid json`} {
			responses := fixture()
			responses[command] = response
			if _, err := generateNotes("1.1.0", fakeRunner(t, responses)); err == nil {
				t.Fatalf("accepted invalid notes %q", response)
			}
		}
	}
	for _, args := range [][]string{nil, {"publish"}, {"check"}, {"check", "--version", "01.0.0"}, {"check", "--version", "1.0.0", "extra"}, {"notes", "--version", "1.1.0"}} {
		if err := executeCLI(args, fakeRunner(t, nil), &stdout); err == nil {
			t.Errorf("accepted arguments %v", args)
		}
	}
	responses := fixture()
	responses["git rev-parse --verify origin/main"] = "new-head"
	stdout.Reset()
	if err := executeCLI([]string{"check", "--version", "1.1.0"}, fakeRunner(t, responses), &stdout); err == nil || !strings.Contains(stdout.String(), `"ok": false`) {
		t.Fatalf("failed checks must emit JSON and return an error: %s, %v", &stdout, err)
	}
}
