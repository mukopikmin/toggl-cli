package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/mukopikmin/toggl-cli/internal/command"
	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/toggl"
	"io"
	"os"
	_ "time/tzdata"
)

var version = "0.0.0-dev"

func run(args []string, stdout, stderr io.Writer) int {
	app := command.App{Client: toggl.HTTPClient{}, Version: version, Out: stdout, Err: stderr}
	if err := app.Run(context.Background(), args); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		var notFound config.FileNotFoundError
		if errors.As(err, &notFound) {
			fmt.Fprintln(stderr, "Please create ~/.config/toggl-cli/config.toml with the following format:")
			fmt.Fprintln(stderr, "workspace = \"your_workspace_id\"")
			fmt.Fprintln(stderr, "token = \"your_api_token\"")
		}
		var usage command.UsageError
		if errors.As(err, &usage) {
			fmt.Fprintf(stderr, "\n%s\n", command.Help)
		}
		return 1
	}
	return 0
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
