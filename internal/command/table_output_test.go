package command

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/synctest"
)

func TestWatchTableRedrawsAfterResize(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		var out bytes.Buffer
		sizes := [][2]int{{14, 8}, {14, 8}, {8, 2}, {8, 8}, {14, 8}}
		calls := 0
		getSize := func() (int, int, error) {
			if calls == len(sizes) {
				cancel()
				return 14, 8, nil
			}
			size := sizes[calls]
			calls++
			return size[0], size[1], nil
		}
		renders := 0
		render := func(width int) string {
			renders++
			return table([]string{"2026-05-09"}, [][]string{{"1.25"}}, tableLayout{maxWidth: width}, []string{"9"})
		}

		width, err := watchTable(ctx, &out, getSize, render)
		if err != nil {
			t.Fatal(err)
		}
		frames := strings.Split(out.String(), clearTableScreen)
		if renders != 3 || len(frames) != 5 || frames[0] != enterTableScreen {
			t.Fatalf("rendered %d times, output = %q", renders, out.String())
		}
		if !strings.Contains(frames[1], "2026-05-09") || !strings.Contains(frames[2], "│ 9    │") || strings.Contains(frames[2], "1.25") || !strings.Contains(frames[3], "1.25") {
			t.Fatalf("unexpected frames: %q", frames)
		}
		if !strings.HasSuffix(frames[4], leaveTableScreen) || !strings.Contains(frames[4], "2026-05-09") || width != 14 {
			t.Fatalf("final frame = %q, width = %d", frames[4], width)
		}
	})
}

func TestWatchTableRestoresScreenAfterSizeError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var out bytes.Buffer
		initial := true
		_, err := watchTable(t.Context(), &out, func() (int, int, error) {
			if initial {
				initial = false
				return 80, 24, nil
			}
			return 0, 0, io.ErrClosedPipe
		}, func(int) string { return "table" })
		if !errors.Is(err, io.ErrClosedPipe) || !strings.HasSuffix(out.String(), leaveTableScreen) {
			t.Fatalf("err = %v, output = %q", err, out.String())
		}
	})
}

func TestTableFrameClipsToTerminalDimensions(t *testing.T) {
	var out bytes.Buffer
	if err := writeTableFrame(&out, "日本語\nabcde\nlast", 3, 2); err != nil {
		t.Fatal(err)
	}
	if want := clearTableScreen + "日\r\nabc"; out.String() != want {
		t.Fatalf("frame = %q, want %q", out.String(), want)
	}
}

func TestTableWatchRequiresTableFormatAndTerminal(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"config", "--format", "csv", "--watch"}, "requires table format"},
		{[]string{"client", "list", "--format", "json", "--watch"}, "requires table format"},
		{[]string{"config", "--format", "table", "--watch"}, "requires a terminal on stdout"},
		{[]string{"project", "list", "--format", "table", "--watch"}, "requires a terminal on stdout"},
		{[]string{"client", "list", "--format", "table", "--watch"}, "requires a terminal on stdout"},
		{[]string{"time-entry", "list", "1", "2", "--format", "table", "--watch"}, "requires a terminal on stdout"},
		{[]string{"summary", "--days", "2", "--format", "table", "--watch"}, "requires a terminal on stdout"},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			var out bytes.Buffer
			app := App{Out: &out}
			err := app.Run(t.Context(), tt.args)
			var usage UsageError
			if !errors.As(err, &usage) || !strings.Contains(err.Error(), tt.want) || out.Len() != 0 {
				t.Fatalf("err = %v, stdout = %q", err, out.String())
			}
		})
	}
}
