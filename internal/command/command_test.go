package command

import (
	"bytes"
	"context"
	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
	"strings"
	"testing"
	"time"
)

type fake struct{}

func (fake) Projects(context.Context, config.Config) ([]model.Project, error) { return nil, nil }
func (fake) TimeEntries(context.Context, config.Config, time.Time, time.Time) ([]model.TimeEntry, error) {
	return nil, nil
}
func TestHelpVersionAndErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
		err  bool
	}{{"help", nil, "Usage:", false}, {"version", []string{"--version"}, "1.2.3", false}, {"unknown", []string{"wat"}, "", true}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			a := App{Client: fake{}, Out: &out, Version: "1.2.3"}
			e := a.Run(context.Background(), tt.args)
			if (e != nil) != tt.err {
				t.Fatal(e)
			}
			if !strings.Contains(out.String(), tt.want) {
				t.Fatalf("%q", out.String())
			}
		})
	}
}
