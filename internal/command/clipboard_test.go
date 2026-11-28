package command

import (
	"errors"
	"strings"
	"testing"
)

func TestClipboardUnavailable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if err := clipboard("text"); err == nil || err.Error() != "Could not copy output to the clipboard." {
		t.Fatal(err)
	}
}

func TestClipboardCommandParityAndFallback(t *testing.T) {
	windows := clipboardCommands("windows")
	if len(windows) != 3 || windows[0].name != "clip" || windows[1].name != "powershell.exe" || windows[2].name != "powershell" {
		t.Fatalf("%+v", windows)
	}
	linux := clipboardCommands("linux")
	attempts := []string{}
	err := writeClipboard("summary output", linux, func(name string) (string, error) { return name, nil }, func(command clipboardCommand, text string) error {
		attempts = append(attempts, command.name+":"+text)
		if command.name == "xclip" {
			return nil
		}
		return errors.New("failed")
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(attempts, ",") != "wl-copy:summary output,xclip:summary output" {
		t.Fatalf("%v", attempts)
	}
}
