package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const DisplayPath = "~/.config/toggl-cli/config.toml"

type Project struct {
	DisplayName  string
	Hidden       bool
	DisplayOrder *float64
}
type Config struct {
	Workspace, Token, Timezone string
	Projects                   map[int64]Project
}
type Document struct {
	Path, Text string
	Config     Config
}

func Path(home string) string { return filepath.Join(home, ".config", "toggl-cli", "config.toml") }
func Load() (Document, error) {
	home := os.Getenv("HOME")
	if home == "" {
		return Document{}, errors.New("HOME environment variable not set")
	}
	p := Path(home)
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return Document{}, fmt.Errorf("%s file not found", DisplayPath)
	}
	if err != nil {
		return Document{}, fmt.Errorf("Unable to read %s: %w", DisplayPath, err)
	}
	c, err := Parse(string(b))
	return Document{p, string(b), c}, err
}

func Parse(text string) (Config, error) {
	c := Config{Projects: map[int64]Project{}}
	var current *int64
	s := bufio.NewScanner(strings.NewReader(text))
	for s.Scan() {
		line := strings.TrimSpace(strings.SplitN(s.Text(), "#", 2)[0])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") {
			raw := strings.Trim(line, "[] ")
			if !strings.HasPrefix(raw, "projects.") {
				return c, fmt.Errorf("Invalid configuration")
			}
			idtxt := strings.Trim(strings.TrimPrefix(raw, "projects."), "\"'")
			id, err := strconv.ParseInt(idtxt, 10, 64)
			if err != nil {
				return c, fmt.Errorf("Invalid project configuration: projects.%s", idtxt)
			}
			current = &id
			if _, ok := c.Projects[id]; !ok {
				c.Projects[id] = Project{}
			}
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return c, fmt.Errorf("Invalid configuration")
		}
		key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		unquote := func(v string) (string, error) { q, err := strconv.Unquote(v); return q, err }
		if current == nil {
			v, err := unquote(val)
			if err != nil {
				return c, fmt.Errorf("Invalid configuration")
			}
			switch key {
			case "workspace":
				c.Workspace = v
			case "token":
				c.Token = v
			case "timezone":
				c.Timezone = v
			}
		} else {
			p := c.Projects[*current]
			switch key {
			case "display_name":
				v, e := unquote(val)
				if e != nil {
					return c, fmt.Errorf("Invalid project configuration: projects.%d", *current)
				}
				p.DisplayName = v
			case "hidden":
				v, e := strconv.ParseBool(val)
				if e != nil {
					return c, fmt.Errorf("Invalid project configuration: projects.%d", *current)
				}
				p.Hidden = v
			case "display_order":
				v, e := strconv.ParseFloat(val, 64)
				if e != nil {
					return c, fmt.Errorf("Invalid project configuration: projects.%d", *current)
				}
				p.DisplayOrder = &v
			}
			c.Projects[*current] = p
		}
	}
	if c.Workspace == "" || c.Token == "" {
		var m []string
		if c.Workspace == "" {
			m = append(m, "workspace")
		}
		if c.Token == "" {
			m = append(m, "token")
		}
		return c, fmt.Errorf("Missing required configuration: %s", strings.Join(m, ", "))
	}
	return c, s.Err()
}
