package config

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/mukopikmin/toggl-cli/internal/model"
)

const (
	settingsFilename = "config.toml"
	DisplayPath      = "~/.config/toggl-cli/" + settingsFilename
)

type HomeNotSetError struct{}

func (HomeNotSetError) Error() string { return "HOME environment variable not set" }

type FileNotFoundError struct{}

func (FileNotFoundError) Error() string { return DisplayPath + " file not found" }

type Document struct {
	Path, Text string
	Settings   model.Settings
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

func Path(home string) string { return configPath(home, settingsFilename) }

func configPath(home, filename string) string {
	return filepath.Join(home, ".config", "toggl-cli", filename)
}

func homePath(filename string) (string, error) {
	home := os.Getenv("HOME")
	if home == "" {
		return "", HomeNotSetError{}
	}
	return configPath(home, filename), nil
}

func Load() (Document, error) {
	p, err := homePath(settingsFilename)
	if err != nil {
		return Document{}, err
	}
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return Document{}, FileNotFoundError{}
	}
	if err != nil {
		return Document{}, fmt.Errorf("Unable to read %s: %w", DisplayPath, err)
	}
	settings, err := Parse(string(b))
	return Document{Path: p, Text: string(b), Settings: settings}, err
}

func Parse(text string) (model.Settings, error) {
	var raw rawConfig
	metadata, err := toml.Decode(text, &raw)
	if err != nil {
		var diagnostic toml.ParseError
		if errors.As(err, &diagnostic) && diagnostic.Position.Line > 0 {
			return model.Settings{}, fmt.Errorf("Invalid configuration at line %d", diagnostic.Position.Line)
		}
		return model.Settings{}, errors.New("Invalid configuration")
	}
	// Implicit parent tables have no metadata type; explicit values must be tables.
	if kind := metadata.Type("projects"); kind != "" && kind != "Hash" {
		return model.Settings{}, fmt.Errorf("Invalid project configuration: projects")
	}
	var missing []string
	if raw.Workspace == nil || *raw.Workspace == "" {
		missing = append(missing, "workspace")
	}
	if raw.Token == nil || *raw.Token == "" {
		missing = append(missing, "token")
	}
	if len(missing) > 0 {
		return model.Settings{}, fmt.Errorf("Missing required configuration: %s", strings.Join(missing, ", "))
	}
	cfg := model.Settings{Workspace: *raw.Workspace, Token: *raw.Token, Projects: map[int64]model.ProjectSettings{}}
	if raw.Timezone != nil {
		cfg.Timezone = *raw.Timezone
	}
	var invalid []string
	for key, p := range raw.Projects {
		id, err := strconv.ParseInt(key, 10, 64)
		kind := metadata.Type("projects", key)
		if err != nil || !regexp.MustCompile(`^\d+$`).MatchString(key) || (kind != "" && kind != "Hash") {
			invalid = append(invalid, "projects."+key)
			continue
		}
		if _, exists := cfg.Projects[id]; exists {
			invalid = append(invalid, fmt.Sprintf("projects.%d (duplicate project ID)", id))
			continue
		}
		project := model.ProjectSettings{DisplayOrder: p.DisplayOrder}
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
		return model.Settings{}, fmt.Errorf("Invalid project configuration: %s", strings.Join(invalid, ", "))
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
