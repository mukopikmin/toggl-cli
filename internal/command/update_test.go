package command

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mukopikmin/toggl-cli/internal/selfupdate"
)

func TestUpdateArgumentValidation(t *testing.T) {
	app := App{Updater: selfupdate.Service{}, Version: "0.0.0-dev", Out: &bytes.Buffer{}, In: bytes.NewBufferString("n\n")}
	tests := []struct {
		args  []string
		usage bool
	}{{[]string{"--channel", "invalid"}, true}, {[]string{"--channel=invalid"}, true}, {[]string{"--channel="}, true}, {[]string{"--channel", ""}, true}, {[]string{"--channel"}, true}, {[]string{"extra"}, true}, {[]string{"--"}, false}, {[]string{"--channel", "stable", "--"}, false}, {nil, false}}
	for _, tt := range tests {
		err := app.update(context.Background(), tt.args)
		var got UsageError
		if tt.usage {
			if !errors.As(err, &got) {
				t.Fatalf("%v: expected usage error, got %v", tt.args, err)
			}
		} else if err == nil || !strings.Contains(err.Error(), "running from source") {
			t.Fatalf("expected source-build error, got %v", err)
		}
	}
}
func TestUpdateUsesRunContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	app := App{Updater: selfupdate.Service{}, Out: &bytes.Buffer{}}
	if err := app.Run(ctx, []string{"update"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want cancellation", err)
	}
}
