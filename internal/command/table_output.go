package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"golang.org/x/term"
)

type tableTerminal interface {
	io.Writer
	Fd() uintptr
}

type tableOutputOptions struct {
	watch, clipboard bool
}

func (a App) validateTableWatch(watch bool, format string) error {
	if !watch {
		return nil
	}
	if format != "table" {
		return usage("--watch requires table format")
	}
	out, ok := a.Out.(tableTerminal)
	if !ok || !term.IsTerminal(int(out.Fd())) {
		return usage("--watch requires a terminal on stdout")
	}
	return nil
}

func (a App) writeTable(ctx context.Context, options tableOutputOptions, headers []string, rows [][]string, alternatives ...[]string) (err error) {
	out, terminal := a.Out.(tableTerminal)
	render := func(width int) string { return table(headers, rows, width, alternatives...) }
	var text string
	if options.watch {
		watchCtx, stop := signal.NotifyContext(ctx, os.Interrupt)
		defer stop()
		fd := out.Fd()
		restore, modeErr := enableTableTerminal(fd)
		if modeErr != nil {
			return modeErr
		}
		defer func() { err = errors.Join(err, restore()) }()
		text, err = watchTable(watchCtx, out, func() (int, int, error) {
			return term.GetSize(int(fd))
		}, render)
		if err != nil {
			return err
		}
	} else {
		var width int
		if terminal {
			width, _, _ = term.GetSize(int(out.Fd()))
		}
		text = render(width)
	}
	if _, err := fmt.Fprintln(a.Out, text); err != nil {
		return err
	}
	if options.clipboard {
		return clipboard(text)
	}
	return nil
}

const (
	enterTableScreen = "\x1b[?1049h"
	leaveTableScreen = "\x1b[?1049l"
	clearTableScreen = "\x1b[2J\x1b[H"
)

func watchTable(ctx context.Context, out io.Writer, getSize func() (int, int, error), render func(int) string) (text string, err error) {
	width, height, err := getSize()
	if err != nil {
		return "", fmt.Errorf("read terminal size: %w", err)
	}
	if width <= 0 || height <= 0 {
		return "", errors.New("--watch requires a positive terminal width and height")
	}

	// The alternate screen keeps resize redraws out of the terminal scrollback.
	defer func() {
		_, restoreErr := io.WriteString(out, leaveTableScreen)
		err = errors.Join(err, restoreErr)
	}()
	if _, err := io.WriteString(out, enterTableScreen); err != nil {
		return "", err
	}
	text = render(width)
	if err := writeTableFrame(out, text, width, height); err != nil {
		return "", err
	}

	// Polling works on consoles that do not send a window resize signal.
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return text, nil
		case <-ticker.C:
			cols, lines, err := getSize()
			if err != nil {
				return "", fmt.Errorf("read terminal size: %w", err)
			}
			if cols <= 0 || lines <= 0 || (cols == width && lines == height) {
				continue
			}
			if cols != width {
				text = render(cols)
			}
			width, height = cols, lines
			if err := writeTableFrame(out, text, width, height); err != nil {
				return "", err
			}
		}
	}
}

func writeTableFrame(out io.Writer, text string, width, height int) error {
	// Keep long lines from wrapping and pushing the heading out of view.
	lines := strings.SplitN(text, "\n", height+1)
	lines = lines[:min(len(lines), height)]
	for row, line := range lines {
		remaining := width
		for column, r := range line {
			remaining -= runeDisplayWidth(r)
			if remaining < 0 {
				lines[row] = line[:column]
				break
			}
		}
	}
	_, err := io.WriteString(out, clearTableScreen+strings.Join(lines, "\r\n"))
	return err
}
