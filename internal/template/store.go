// Package template persists named summary settings separately from credentials.
package template

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/BurntSushi/toml"
	"github.com/mukopikmin/toggl-cli/internal/model"
)

const DisplayPath = "~/.config/toggl-cli/templates.toml"

type Document struct {
	Templates map[string]SavedSummary
}

type SavedSummary struct {
	Options         model.SummaryOptions
	legacySeparator *string
}

type tomlDocument struct {
	Templates map[string]tomlSummary `toml:"templates"`
}

type tomlSummary struct {
	StartDate string `toml:"start_date,omitempty"`
	EndDate   string `toml:"end_date,omitempty"`
	Days      *int   `toml:"days,omitempty"`
	Format    string `toml:"format,omitempty"`
	// Preserve the legacy field when showing or rewriting a saved template.
	Separator *string `toml:"separator,omitempty"`
	NoProject bool    `toml:"no_project"`
	NoDate    bool    `toml:"no_date"`
	Clipboard bool    `toml:"clipboard"`
}

func ValidateName(name string) error {
	if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`).MatchString(name) {
		return fmt.Errorf("template name must start with a letter or digit and contain only letters, digits, underscores, or hyphens")
	}
	return nil
}

func path() (string, error) {
	home := os.Getenv("HOME")
	if home == "" {
		return "", fmt.Errorf("HOME environment variable not set")
	}
	return filepath.Join(home, ".config", "toggl-cli", "templates.toml"), nil
}

func Load() (Document, error) {
	p, err := path()
	if err != nil {
		return Document{}, err
	}
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return Document{Templates: map[string]SavedSummary{}}, nil
	}
	if err != nil {
		return Document{}, fmt.Errorf("unable to read %s: %w", DisplayPath, err)
	}
	return Parse(string(b))
}

func Parse(text string) (Document, error) {
	var stored tomlDocument
	metadata, err := toml.Decode(text, &stored)
	if err != nil {
		return Document{}, fmt.Errorf("invalid templates: %w", err)
	}
	if keys := metadata.Undecoded(); len(keys) != 0 {
		return Document{}, fmt.Errorf("unknown template setting: %s", keys[0])
	}
	if kind := metadata.Type("templates"); kind != "" && kind != "Hash" {
		return Document{}, fmt.Errorf("templates must be a TOML table")
	}
	d := Document{Templates: make(map[string]SavedSummary, len(stored.Templates))}
	for name, s := range stored.Templates {
		d.Templates[name] = SavedSummary{
			Options: model.SummaryOptions{
				Period: model.SummaryPeriod{StartDate: s.StartDate, EndDate: s.EndDate, Days: s.Days},
				Output: model.SummaryOutput{Format: s.Format, NoProject: s.NoProject, NoDate: s.NoDate, Clipboard: s.Clipboard},
			},
			legacySeparator: s.Separator,
		}
	}
	if err := d.validate(); err != nil {
		return Document{}, err
	}
	return d, nil
}

func (d Document) Names() []string {
	names := make([]string, 0, len(d.Templates))
	for name := range d.Templates {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (d Document) validate() error {
	for _, name := range d.Names() {
		if err := ValidateName(name); err != nil {
			return fmt.Errorf("invalid template %q: %w", name, err)
		}
		if err := d.Templates[name].Options.Validate(false); err != nil {
			return fmt.Errorf("invalid template %q: %w", name, err)
		}
	}
	return nil
}

func (d Document) Encode() (string, error) {
	if err := d.validate(); err != nil {
		return "", err
	}
	stored := tomlDocument{}
	if d.Templates != nil {
		stored.Templates = make(map[string]tomlSummary, len(d.Templates))
	}
	for name, saved := range d.Templates {
		period, output := saved.Options.Period, saved.Options.Output
		stored.Templates[name] = tomlSummary{
			StartDate: period.StartDate,
			EndDate:   period.EndDate,
			Days:      period.Days,
			Format:    output.Format,
			Separator: saved.legacySeparator,
			NoProject: output.NoProject,
			NoDate:    output.NoDate,
			Clipboard: output.Clipboard,
		}
	}
	var out bytes.Buffer
	if err := toml.NewEncoder(&out).Encode(stored); err != nil {
		return "", err
	}
	return out.String(), nil
}

// Write replaces the file only after the full document is validated and written.
func (d Document) Write() error {
	text, err := d.Encode()
	if err != nil {
		return err
	}
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(p), ".templates-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.WriteString(text); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), p)
}
