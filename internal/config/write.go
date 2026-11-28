package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

// PrepareCreate checks whether the home configuration can be created before prompting.
func PrepareCreate() (string, error) {
	path, err := homePath()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("%s already exists", DisplayPath)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return path, nil
}

func Create(path string, settings model.Settings) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	text, err := EncodeInitial(settings.Workspace, settings.Token, settings.Timezone)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if _, err := file.WriteString(text); err != nil {
		file.Close()
		os.Remove(path)
		return err
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return err
	}
	return nil
}
