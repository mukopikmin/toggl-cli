package command

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (a App) installCompletion(shell string) error {
	switch shell {
	case "bash", "zsh", "fish":
	default:
		return usagef("unsupported completion shell: %s (expected bash, zsh, or fish)", shell)
	}
	home := os.Getenv("HOME")
	if home == "" {
		return fmt.Errorf("HOME environment variable not set")
	}
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		dataDir = filepath.Join(home, ".local", "share")
	}
	var destination, notice string
	switch shell {
	case "bash":
		dir := os.Getenv("BASH_COMPLETION_USER_DIR")
		if dir == "" {
			dir = filepath.Join(dataDir, "bash-completion")
		}
		destination = filepath.Join(dir, "completions", "toggl")
		notice = "Open a new bash session with bash-completion enabled to load completions."
	case "fish":
		dir := os.Getenv("XDG_CONFIG_HOME")
		if dir == "" {
			dir = filepath.Join(home, ".config")
		}
		destination = filepath.Join(dir, "fish", "completions", "toggl.fish")
		notice = "Open a new fish session to load completions."
	case "zsh":
		// Match the installer's default fpath discovery without loading user rc files.
		if paths, err := exec.Command("zsh", "-fc", `printf '%s\n' "${fpath[@]}"`).Output(); err == nil {
			for _, dir := range strings.Split(strings.TrimSpace(string(paths)), "\n") {
				if !filepath.IsAbs(dir) || filepath.Base(dir) != "site-functions" {
					continue
				}
				if info, err := os.Stat(dir); err != nil || !info.IsDir() {
					continue
				}
				path := filepath.Join(dir, "_toggl")
				if err := writeCompletion(shell, path); err == nil {
					_, err := fmt.Fprintf(a.Out, "Installed zsh completion to %s\nOpen a new zsh session with compinit enabled to load completions.\n", path)
					return err
				}
			}
		}
		destination = filepath.Join(dataDir, "zsh", "site-functions", "_toggl")
		notice = "No writable default site-functions directory was found. Place _toggl in a directory on your fpath, or add its directory to fpath before running compinit. Open a new zsh session afterward."
	}
	if err := writeCompletion(shell, destination); err != nil {
		return fmt.Errorf("install %s completion to %s: %w", shell, destination, err)
	}
	_, err := fmt.Fprintf(a.Out, "Installed %s completion to %s\n%s\n", shell, destination, notice)
	return err
}

func writeCompletion(shell, destination string) error {
	content, err := completionScripts.ReadFile("completions/" + shell)
	if err != nil {
		return err
	}
	dir := filepath.Dir(destination)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".toggl-completion-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(content); err != nil {
		f.Close()
		return err
	}
	if err := f.Chmod(0644); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), destination)
}
