package config

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

const DisplayPath = "~/.config/toggl-cli/config.toml"

type HomeNotSetError struct{}

func (HomeNotSetError) Error() string { return "HOME environment variable not set" }

type FileNotFoundError struct{}

func (FileNotFoundError) Error() string { return DisplayPath + " file not found" }

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
type rawProject struct {
	DisplayName  *string  `toml:"display_name"`
	Hidden       *bool    `toml:"hidden"`
	DisplayOrder *float64 `toml:"display_order"`
}
type rawConfig struct {
	Workspace *string               `toml:"workspace"`
	Token     *string               `toml:"token"`
	Timezone  *string               `toml:"timezone"`
	Projects  map[string]rawProject `toml:"projects"`
}

func Path(home string) string { return filepath.Join(home, ".config", "toggl-cli", "config.toml") }
func Load() (Document, error) {
	home := os.Getenv("HOME")
	if home == "" {
		return Document{}, HomeNotSetError{}
	}
	p := Path(home)
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return Document{}, FileNotFoundError{}
	}
	if err != nil {
		return Document{}, fmt.Errorf("Unable to read %s: %w", DisplayPath, err)
	}
	c, err := Parse(string(b))
	return Document{Path: p, Text: string(b), Config: c}, err
}

func Parse(text string) (Config, error) {
	var raw rawConfig
	if _, err := toml.Decode(text, &raw); err != nil {
		return Config{}, fmt.Errorf("Invalid configuration: %w", err)
	}
	var missing []string
	if raw.Workspace == nil || *raw.Workspace == "" {
		missing = append(missing, "workspace")
	}
	if raw.Token == nil || *raw.Token == "" {
		missing = append(missing, "token")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("Missing required configuration: %s", strings.Join(missing, ", "))
	}
	cfg := Config{Workspace: *raw.Workspace, Token: *raw.Token, Projects: map[int64]Project{}}
	if raw.Timezone != nil {
		cfg.Timezone = *raw.Timezone
	}
	var invalid []string
	for key, p := range raw.Projects {
		id, err := strconv.ParseInt(key, 10, 64)
		if err != nil || !regexp.MustCompile(`^\d+$`).MatchString(key) {
			invalid = append(invalid, "projects."+key)
			continue
		}
		project := Project{DisplayOrder: p.DisplayOrder}
		if p.DisplayOrder != nil && (math.IsNaN(*p.DisplayOrder) || math.IsInf(*p.DisplayOrder, 0)) {
			invalid = append(invalid, "projects."+key)
			continue
		}
		if p.DisplayName != nil {
			if *p.DisplayName == "" {
				invalid = append(invalid, "projects."+key)
				continue
			}
			project.DisplayName = *p.DisplayName
		}
		if p.Hidden != nil {
			project.Hidden = *p.Hidden
		}
		cfg.Projects[id] = project
	}
	if len(invalid) > 0 {
		return Config{}, fmt.Errorf("Invalid project configuration: %s", strings.Join(invalid, ", "))
	}
	return cfg, nil
}

func EncodeInitial(workspace, token, timezone string) (string, error) {
	value := struct {
		Workspace string `toml:"workspace"`
		Token     string `toml:"token"`
		Timezone  string `toml:"timezone"`
	}{workspace, token, timezone}
	var output bytes.Buffer
	if err := toml.NewEncoder(&output).Encode(value); err != nil {
		return "", err
	}
	return output.String(), nil
}
