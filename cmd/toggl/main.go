package main

import (
	"context"
	"fmt"
	"github.com/mukopikmin/toggl-cli/internal/command"
	"github.com/mukopikmin/toggl-cli/internal/toggl"
	"os"
)

var version = "0.0.0-dev"

func main() {
	app := command.App{Client: toggl.HTTPClient{}, Version: version}
	if err := app.Run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n%s\n", err, command.Help)
		os.Exit(1)
	}
}
