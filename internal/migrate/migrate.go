package migrate

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

type legacyProject struct {
	DisplayName string `toml:"display_name"`
}
type legacyConfig struct {
	Workspace string                   `toml:"workspace"`
	Token     string                   `toml:"token"`
	Projects  map[string]legacyProject `toml:"projects,omitempty"`
}

func Convert(text string) (string, error) {
	cfg := legacyConfig{Projects: map[string]legacyProject{}}
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		switch {
		case key == "WORKSPACE":
			cfg.Workspace = value
		case key == "TOKEN":
			cfg.Token = value
		case strings.HasPrefix(key, "PROJECT_NAME_") && len(key) > len("PROJECT_NAME_"):
			cfg.Projects[strings.TrimPrefix(key, "PROJECT_NAME_")] = legacyProject{value}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := toml.NewEncoder(&out).Encode(cfg); err != nil {
		return "", err
	}
	return out.String(), nil
}
func MissingLegacyError() error { return fmt.Errorf("~/.toggl_config file not found") }
