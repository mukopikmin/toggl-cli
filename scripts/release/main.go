// Command release validates a tagged stable release and generates its notes.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const baselineTag = "release-notes-baseline"

var stableVersion = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

type runner func(string, ...string) (string, error)

func run(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s %v: %w: %s", command, args, err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(output)), nil
}

type release struct {
	TagName      string `json:"tagName"`
	IsDraft      bool   `json:"isDraft"`
	IsPrerelease bool   `json:"isPrerelease"`
}

type context struct {
	HeadSHA                   string    `json:"headSha"`
	PreviousStableTag         string    `json:"previousStableTag"`
	ApprovedPreviousStableTag string    `json:"approvedPreviousStableTag"`
	BaselineSelection         string    `json:"baselineSelection"`
	Releases                  []release `json:"releases"`
	StableTags                []string  `json:"stableTags"`
	repository                string
}

const approvedBaselineFormat = "%(objecttype)%0a%(contents:trailers:key=Previous-Stable-Tag,valueonly)"

func approvedBaseline(tagInfo string) (string, error) {
	lines := strings.Split(strings.TrimSpace(tagInfo), "\n")
	if len(lines) != 2 || lines[0] != "tag" || (lines[1] != baselineTag && !(strings.HasPrefix(lines[1], "v") && stableVersion.MatchString(lines[1][1:]))) {
		return "", errors.New("release requires an annotated tag with exactly one Previous-Stable-Tag trailer containing a stable tag or release-notes-baseline")
	}
	return lines[1], nil
}

func previousStableTag(releases []release) string {
	for _, r := range releases {
		if !r.IsDraft && !r.IsPrerelease && strings.HasPrefix(r.TagName, "v") && stableVersion.MatchString(r.TagName[1:]) {
			return r.TagName
		}
	}
	return baselineTag
}

func newerVersion(version, previousTag string) bool {
	if previousTag == baselineTag {
		return true
	}
	next, previous := strings.Split(version, "."), strings.Split(strings.TrimPrefix(previousTag, "v"), ".")
	for i := range next {
		if len(next[i]) != len(previous[i]) {
			return len(next[i]) > len(previous[i])
		}
		if next[i] != previous[i] {
			return next[i] > previous[i]
		}
	}
	return false
}

func releaseContext(version string, execute runner) (context, error) {
	c := context{Releases: []release{}, StableTags: []string{}}
	var err error
	if c.HeadSHA, err = execute("git", "rev-parse", "HEAD"); err != nil {
		return c, err
	}
	if c.repository, err = execute("gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner"); err != nil {
		return c, err
	}
	tagInfo, err := execute("git", "for-each-ref", "--format="+approvedBaselineFormat, "refs/tags/v"+version)
	if err != nil {
		return c, err
	}
	if c.ApprovedPreviousStableTag, err = approvedBaseline(tagInfo); err != nil {
		return c, err
	}
	// Include drafts and every page. API failures must never mean absence.
	output, err := execute("gh", "api", "--paginate", "repos/"+c.repository+"/releases?per_page=100", "--jq", ".[] | {tagName: .tag_name, isDraft: .draft, isPrerelease: .prerelease}")
	if err != nil {
		return c, err
	}
	decoder := json.NewDecoder(strings.NewReader(output))
	for {
		var r release
		if err := decoder.Decode(&r); err == io.EOF {
			break
		} else if err != nil {
			return c, fmt.Errorf("decode releases: %w", err)
		}
		c.Releases = append(c.Releases, r)
	}
	c.PreviousStableTag = previousStableTag(c.Releases)
	c.BaselineSelection = "first stable release in GitHub API order"
	tags, err := execute("git", "ls-remote", "--tags", "origin", "refs/tags/v*")
	if err != nil {
		return c, err
	}
	for _, line := range strings.Split(tags, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || !strings.HasPrefix(fields[1], "refs/tags/v") {
			continue
		}
		tag := strings.TrimPrefix(fields[1], "refs/tags/")
		if tag != "v"+version && stableVersion.MatchString(tag[1:]) {
			c.StableTags = append(c.StableTags, tag)
		}
	}
	if c.PreviousStableTag == baselineTag {
		c.BaselineSelection = "no stable release returned by GitHub"
	}
	diagnostics, _ := json.Marshal(c)
	if c.PreviousStableTag == baselineTag && len(c.StableTags) != 0 {
		return c, fmt.Errorf("cannot infer a first release: stable tags exist but no stable releases were returned: %s", diagnostics)
	}
	if c.PreviousStableTag != c.ApprovedPreviousStableTag {
		return c, fmt.Errorf("notes baseline differs from the approved tag trailer: %s", diagnostics)
	}
	return c, nil
}

type check struct {
	Name     string `json:"name"`
	OK       bool   `json:"ok"`
	Actual   string `json:"actual"`
	Expected string `json:"expected"`
}

type inspection struct {
	context
	OK        bool    `json:"ok"`
	Version   string  `json:"version"`
	Tag       string  `json:"tag"`
	TagObject string  `json:"tagObject"`
	Checks    []check `json:"checks"`
	CIRuns    []ciRun `json:"ciRuns"`
}

type ciRun struct {
	DatabaseID   int64  `json:"databaseId"`
	WorkflowName string `json:"workflowName"`
	HeadBranch   string `json:"headBranch"`
	Event        string `json:"event"`
	Status       string `json:"status"`
	Conclusion   string `json:"conclusion"`
	HeadSHA      string `json:"headSha"`
	URL          string `json:"url"`
}

func inspectRelease(version string, execute runner) (inspection, error) {
	var result inspection
	if !stableVersion.MatchString(version) {
		return result, fmt.Errorf("invalid semantic version: %s", version)
	}
	c, err := releaseContext(version, execute)
	if err != nil {
		return result, err
	}
	result.context, result.Version, result.Tag = c, version, "v"+version
	if _, err = execute("git", "fetch", "--no-tags", "origin", "main"); err != nil {
		return result, err
	}
	// Read all required refs with hard failures rather than treating them as absent.
	refs := []string{"origin/main", "refs/tags/" + result.Tag, "refs/tags/" + result.Tag + "^{commit}", "refs/tags/" + c.PreviousStableTag + "^{commit}"}
	values := make([]string, len(refs))
	for i, ref := range refs {
		values[i], err = execute("git", "rev-parse", "--verify", ref)
		if err != nil {
			return result, err
		}
	}
	mainSHA, tagObject, tagCommit, previousCommit := values[0], values[1], values[2], values[3]
	result.TagObject = tagObject
	tagRef := "refs/tags/" + result.Tag
	remoteTag, err := execute("git", "ls-remote", "--exit-code", "--tags", "origin", tagRef)
	if err != nil {
		return result, err
	}
	remoteTagObject := ""
	for _, line := range strings.Split(remoteTag, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == tagRef {
			remoteTagObject = fields[0]
		}
	}
	_, ancestorErr := execute("git", "merge-base", "--is-ancestor", previousCommit, c.HeadSHA)
	baselineOK := ancestorErr == nil && previousCommit != c.HeadSHA
	if c.PreviousStableTag == baselineTag {
		root, err := execute("git", "rev-list", "--max-parents=0", "HEAD")
		if err != nil {
			return result, err
		}
		baselineOK = baselineOK && root == previousCommit
	}
	ciOutput, err := execute("gh", "run", "list", "--repo", c.repository, "--workflow", "test.yml", "--branch", "main", "--event", "push", "--commit", c.HeadSHA, "--limit", "1", "--json", "databaseId,workflowName,headBranch,event,status,conclusion,headSha,url")
	if err != nil {
		return result, err
	}
	result.CIRuns = []ciRun{}
	if err := json.Unmarshal([]byte(ciOutput), &result.CIRuns); err != nil {
		return result, fmt.Errorf("decode Test runs: %w", err)
	}
	ciURL := ""
	for _, r := range result.CIRuns {
		if r.DatabaseID > 0 && r.WorkflowName == "Test" && r.HeadBranch == "main" && r.Event == "push" && r.Status == "completed" && r.Conclusion == "success" && r.HeadSHA == c.HeadSHA {
			ciURL = r.URL
		}
	}
	absent := true
	for _, r := range c.Releases {
		absent = absent && r.TagName != result.Tag
	}
	result.Checks = []check{
		{"remoteMain", c.HeadSHA == mainSHA, c.HeadSHA, mainSHA},
		{"releaseTag", tagCommit == c.HeadSHA && remoteTagObject == tagObject, tagCommit + " / " + tagObject + " locally / " + remoteTagObject + " on origin", "unchanged " + result.Tag + " at " + c.HeadSHA},
		{"releaseAbsent", absent, fmt.Sprintf("absent: %t", absent), "absent, including drafts"},
		{"notesBaseline", baselineOK, c.PreviousStableTag + " at " + previousCommit, "an earlier ancestor; the root commit for the first release"},
		{"newerVersion", newerVersion(version, c.PreviousStableTag), version, "newer than " + c.PreviousStableTag},
		{"ci", ciURL != "", "selected URL: " + ciURL + "; returned runs: " + ciOutput, "successful completed Test push run on main for " + c.HeadSHA},
	}
	result.OK = true
	for _, check := range result.Checks {
		result.OK = result.OK && check.OK
	}
	return result, nil
}

type notes struct {
	context
	Version string `json:"version"`
	Tag     string `json:"tag"`
	Title   string `json:"title"`
	Body    string `json:"body"`
}

func generateNotes(version string, execute runner) (notes, error) {
	var result notes
	if !stableVersion.MatchString(version) {
		return result, fmt.Errorf("invalid semantic version: %s", version)
	}
	c, err := releaseContext(version, execute)
	if err != nil {
		return result, err
	}
	if _, err = execute("git", "rev-parse", "--verify", "refs/tags/"+c.PreviousStableTag+"^{commit}"); err != nil {
		return result, err
	}
	output, err := execute("gh", "api", "--method", "POST", "repos/"+c.repository+"/releases/generate-notes", "-f", "tag_name=v"+version, "-f", "target_commitish="+c.HeadSHA, "-f", "previous_tag_name="+c.PreviousStableTag)
	if err != nil {
		return result, err
	}
	var generated struct{ Name, Body string }
	if err = json.Unmarshal([]byte(output), &generated); err != nil {
		return result, fmt.Errorf("decode release notes: %w", err)
	}
	if strings.TrimSpace(generated.Body) == "" {
		return result, errors.New("generated release notes are empty")
	}
	result = notes{context: c, Version: version, Tag: "v" + version, Title: generated.Name, Body: generated.Body}
	if result.Title == "" {
		result.Title = result.Tag
	}
	return result, nil
}

func main() {
	if err := executeCLI(os.Args[1:], run, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func executeCLI(args []string, execute runner, stdout io.Writer) error {
	if len(args) == 0 || (args[0] != "check" && args[0] != "notes") {
		return errors.New("expected command 'check' or 'notes'")
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	version := flags.String("version", "", "stable semantic version without v prefix")
	output := flags.String("output", "", "notes JSON output path")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || !stableVersion.MatchString(*version) {
		return errors.New("expected --version MAJOR.MINOR.PATCH without leading zeroes and no positional arguments")
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if args[0] == "check" {
		result, err := inspectRelease(*version, execute)
		if err != nil {
			return err
		}
		if err := encoder.Encode(result); err != nil {
			return err
		}
		if !result.OK {
			return errors.New("release validation failed")
		}
		return nil
	}
	if *output == "" {
		return errors.New("notes requires --output <path>")
	}
	result, err := generateNotes(*version, execute)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(*output, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return encoder.Encode(result)
}
