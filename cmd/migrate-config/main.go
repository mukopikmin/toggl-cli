package main

import (
	"fmt"
	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/migrate"
	"os"
	"path/filepath"
)

func run() error {
	home := os.Getenv("HOME")
	if home == "" {
		return fmt.Errorf("HOME environment variable not set")
	}
	target := config.Path(home)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("%s already exists; remove it first if you want to overwrite it", config.DisplayPath)
	} else if !os.IsNotExist(err) {
		return err
	}
	b, err := os.ReadFile(filepath.Join(home, ".toggl_config"))
	if os.IsNotExist(err) {
		return migrate.MissingLegacyError()
	}
	if err != nil {
		return err
	}
	text, err := migrate.Convert(string(b))
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	if err = os.WriteFile(target, []byte(text), 0600); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n", target)
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
