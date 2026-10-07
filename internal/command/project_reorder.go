package command

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
	"golang.org/x/term"
)

type reorderState struct {
	Projects []model.Project
	Selected int
	Moving   bool
}

func updateReorderState(s reorderState, action string) reorderState {
	if len(s.Projects) == 0 {
		return s
	}
	destination := s.Selected
	switch action {
	case "select-up":
		destination--
	case "select-down":
		destination++
	case "move-up":
		destination--
	case "move-down":
		destination++
	}
	if destination < 0 || destination >= len(s.Projects) {
		return s
	}
	if strings.HasPrefix(action, "move-") {
		s.Projects = append([]model.Project(nil), s.Projects...)
		s.Projects[s.Selected], s.Projects[destination] = s.Projects[destination], s.Projects[s.Selected]
	}
	s.Selected = destination
	return s
}

func renderReorder(s reorderState) string {
	lines := []string{"\x1b[2J\x1b[HReorder projects", "j/k: select down/up  Shift+j/Shift+k: move down/up  Space: pick/drop", "While picked, j/k or Up/Down: move  Enter: save  q/Esc: cancel", ""}
	for i, p := range s.Projects {
		marker := " "
		if i == s.Selected {
			marker = ">"
			if s.Moving {
				marker = "*"
			}
		}
		lines = append(lines, fmt.Sprintf("%s %s", marker, p.DisplayName))
	}
	// Raw mode disables the terminal's automatic carriage return on newline.
	return strings.Join(lines, "\r\n")
}

func selectOrder(projects []model.Project, in, out *os.File) ([]model.Project, bool, error) {
	if len(projects) == 0 {
		return projects, true, nil
	}
	if !term.IsTerminal(int(in.Fd())) || !term.IsTerminal(int(out.Fd())) {
		return nil, false, errors.New("project reorder requires an interactive terminal")
	}
	old, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return nil, false, err
	}
	defer term.Restore(int(in.Fd()), old)
	fmt.Fprint(out, "\x1b[?25l")
	defer fmt.Fprint(out, "\x1b[2J\x1b[H\x1b[?25h")
	return selectOrderLoop(projects, in, out)
}

func selectOrderLoop(projects []model.Project, in io.Reader, out io.Writer) ([]model.Project, bool, error) {
	s := reorderState{Projects: append([]model.Project(nil), projects...)}
	buf := make([]byte, 16)
	for {
		fmt.Fprint(out, renderReorder(s))
		n, e := in.Read(buf)
		if errors.Is(e, io.EOF) {
			return nil, false, nil
		}
		if e != nil {
			return nil, false, e
		}
		key := string(buf[:n])
		if key == "\r" || key == "\n" {
			return s.Projects, true, nil
		}
		if key == "q" || key == "\x1b" {
			return nil, false, nil
		}
		if key == " " {
			s.Moving = !s.Moving
			continue
		}
		action := ""
		switch key {
		case "k", "\x1b[A":
			if s.Moving {
				action = "move-up"
			} else {
				action = "select-up"
			}
		case "j", "\x1b[B":
			if s.Moving {
				action = "move-down"
			} else {
				action = "select-down"
			}
		case "K", "\x1b[1;5A":
			action = "move-up"
		case "J", "\x1b[1;5B":
			action = "move-down"
		}
		if action != "" {
			s = updateReorderState(s, action)
		}
	}
}

var projectHeader = regexp.MustCompile(`^\s*\[\s*projects\.(?:"(\d+)"|'(\d+)'|(\d+))\s*\]\s*(?:#.*)?$`)
var orderLine = regexp.MustCompile(`^\s*display_order\s*=`)

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

func (a App) reorderProjects(doc config.Document, projects []model.Project) error {
	if len(projects) == 0 {
		fmt.Fprintln(a.Out, "No visible projects to reorder")
		return nil
	}
	in, okIn := a.In.(*os.File)
	out, okOut := a.Out.(*os.File)
	if !okIn || !okOut {
		return errors.New("project reorder requires an interactive terminal")
	}
	ordered, saved, err := selectOrder(projects, in, out)
	if err != nil {
		return err
	}
	if !saved {
		fmt.Fprintln(a.Out, "Project order was not changed")
		return nil
	}
	ids := make([]int64, len(ordered))
	for i, p := range ordered {
		ids[i] = p.ID
	}
	text := updateProjectOrders(doc.Text, ids)
	if _, err = config.Parse(text); err != nil {
		return err
	}
	if err = os.WriteFile(doc.Path, []byte(text), 0600); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "Saved the order of %d project(s)\n", len(ordered))
	return nil
}
