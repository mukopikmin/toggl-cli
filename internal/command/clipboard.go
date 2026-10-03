package command

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

type clipboardCommand struct {
	name string
	args []string
}

func clipboardCommands(goos string) []clipboardCommand {
	switch goos {
	case "darwin":
		return []clipboardCommand{{name: "pbcopy"}}
	case "windows":
		return []clipboardCommand{{name: "clip"}, {name: "powershell.exe", args: []string{"-NoProfile", "-Command", "Set-Clipboard"}}, {name: "powershell", args: []string{"-NoProfile", "-Command", "Set-Clipboard"}}}
	case "linux":
		return []clipboardCommand{{name: "wl-copy"}, {name: "xclip", args: []string{"-selection", "clipboard"}}, {name: "xsel", args: []string{"--clipboard", "--input"}}}
	default:
		return nil
	}
}

func writeClipboard(text string, commands []clipboardCommand, lookPath func(string) (string, error), run func(clipboardCommand, string) error) error {
	for _, command := range commands {
		if _, err := lookPath(command.name); err != nil {
			continue
		}
		if run(command, text) == nil {
			return nil
		}
	}
	return errors.New("Could not copy output to the clipboard.")
}

func clipboard(text string) error {
	return writeClipboard(text, clipboardCommands(runtime.GOOS), exec.LookPath, func(command clipboardCommand, text string) error {
		child := exec.Command(command.name, command.args...)
		child.Stdin = strings.NewReader(text)
		return child.Run()
	})
}
