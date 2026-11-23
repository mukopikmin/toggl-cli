package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/mukopikmin/toggl-cli/internal/model"
)

// SyncProjects adds missing project settings to the snapshot loaded before the fetch.
func (d Document) SyncProjects(projects []model.Project) (int, error) {
	projects = slices.Clone(projects)
	sort.Slice(projects, func(i, j int) bool { return projects[i].ID < projects[j].ID })
	var text strings.Builder
	text.WriteString(d.Text)
	if !strings.HasSuffix(d.Text, "\n") {
		text.WriteByte('\n')
	}
	count := 0
	added := make(map[int64]bool)
	for _, project := range projects {
		if _, exists := d.Config.Projects[project.ID]; exists || added[project.ID] {
			continue
		}
		name := strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(project.Name)
		fmt.Fprintf(&text, "\n# %s\n[projects.\"%d\"]\nhidden = false\n", strings.ReplaceAll(name, "\n", "\n# "), project.ID)
		added[project.ID] = true
		count++
	}
	if count == 0 {
		return 0, nil
	}
	if err := d.save(text.String()); err != nil {
		return 0, err
	}
	return count, nil
}

// SaveProjectOrder updates visible projects in the order returned by the terminal.
func (d Document) SaveProjectOrder(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	// Compare decoded values to detect layouts the text editor cannot safely
	// modify, including inline tables, dotted keys and multiline values.
	var expected map[string]any
	if _, err := toml.Decode(d.Text, &expected); err != nil {
		return err
	}
	projects, _ := expected["projects"].(map[string]any)
	if projects == nil {
		projects = make(map[string]any)
		expected["projects"] = projects
	}
	keys := make(map[int64]string, len(projects))
	for key := range projects {
		id, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			return err
		}
		if other, ok := keys[id]; ok {
			return fmt.Errorf("ambiguous project ID %d in keys %q and %q", id, other, key)
		}
		keys[id] = key
	}
	for i, id := range ids {
		key, exists := keys[id]
		if !exists {
			key = strconv.FormatInt(id, 10)
			projects[key] = make(map[string]any)
			keys[id] = key
		}
		project, ok := projects[key].(map[string]any)
		if !ok {
			return fmt.Errorf("invalid project table %q", key)
		}
		project["display_order"] = int64(i + 1)
	}
	text := updateProjectOrders(d.Text, ids)
	var actual map[string]any
	if _, err := toml.Decode(text, &actual); err != nil || !reflect.DeepEqual(expected, actual) {
		var encoded strings.Builder
		if err := toml.NewEncoder(&encoded).Encode(expected); err != nil {
			return err
		}
		text = encoded.String()
	}
	return d.save(text)
}

func (d Document) save(text string) error {
	if _, err := Parse(text); err != nil {
		return err
	}
	path, err := filepath.EvalSymlinks(d.Path)
	if err != nil {
		return err
	}
	stage, err := os.CreateTemp(filepath.Dir(path), ".toggl-config-*")
	if err != nil {
		return err
	}
	defer os.Remove(stage.Name())
	_, writeErr := stage.WriteString(text)
	closeErr := stage.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	current, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if string(current) != d.Text {
		return errors.New("configuration changed while the command was running; retry with the latest file")
	}
	return os.Rename(stage.Name(), path)
}

var projectHeader = regexp.MustCompile(`^\s*\[\s*(?:projects|"projects"|'projects')[ \t]*\.[ \t]*(?:"(\d+)"|'(\d+)'|(\d+))\s*\]\s*(?:#.*)?$`)
var orderLine = regexp.MustCompile(`^\s*(?:display_order|"display_order"|'display_order')\s*=`)

func updateProjectOrders(text string, ids []int64) string {
	if len(ids) == 0 {
		return text
	}
	newline := "\n"
	if strings.Contains(text, "\r\n") {
		newline = "\r\n"
	}
	trailing := strings.HasSuffix(text, "\n")
	lines := regexp.MustCompile(`\r?\n`).Split(text, -1)
	if trailing {
		lines = lines[:len(lines)-1]
	}
	remaining := map[int64]int{}
	for i, id := range ids {
		remaining[id] = i + 1
	}
	out := []string{}
	for i := 0; i < len(lines); {
		m := projectHeader.FindStringSubmatch(lines[i])
		if m == nil {
			out = append(out, lines[i])
			i++
			continue
		}
		idText := m[1] + m[2] + m[3]
		id, _ := strconv.ParseInt(idText, 10, 64)
		order, ok := remaining[id]
		out = append(out, lines[i])
		i++
		found := false
		for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "[") {
			line := lines[i]
			if ok && orderLine.MatchString(line) {
				indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
				comment := ""
				if at := strings.Index(line, " #"); at >= 0 {
					comment = line[at:]
				}
				line = fmt.Sprintf("%sdisplay_order = %d%s", indent, order, comment)
				found = true
			}
			out = append(out, line)
			i++
		}
		if ok && !found {
			out = append(out, fmt.Sprintf("display_order = %d", order))
		}
		delete(remaining, id)
	}
	for _, id := range ids {
		order, ok := remaining[id]
		if !ok {
			continue
		}
		if len(out) > 0 && out[len(out)-1] != "" {
			out = append(out, "")
		}
		out = append(out, fmt.Sprintf("[projects.\"%d\"]", id), fmt.Sprintf("display_order = %d", order))
	}
	result := strings.Join(out, newline)
	if trailing {
		result += newline
	}
	return result
}
