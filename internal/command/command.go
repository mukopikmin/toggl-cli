package command

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"
)

const Help = `Usage:
  toggl summary <start-date> <end-date> [options]
  toggl summary --days <days> [options]
  toggl summary --month <month> [options]
  toggl summary --template <name> [options]
  toggl template add <name> [<start-date> <end-date> | --days <days>] [options] [--force]
  toggl template list
  toggl template show <name>
  toggl template delete <name>
  toggl time-entry list <start-day> <end-day> [options]
  toggl client list [options]
  toggl project list [options]
  toggl project reorder
  toggl project sync
  toggl config [options]
  toggl init
  toggl update [--channel stable|nightly]
  toggl completion <bash|zsh|fish>

Commands:
  init        Create the configuration file
  client      List workspace clients as CSV, JSON, or a table
  project     List, reorder, and sync projects
  time-entry  List individual time entries for a range of days
  config      Show configuration values
  summary     Summarize time entries for a range of days
  template    Add, list, show, and delete summary templates
  update      Update the installed Toggl CLI binary
  completion  Print a shell completion script

Options:
  -f, --format <format>  Set the output format: csv, tsv, json, or table (default: table)
      --watch            Redraw table output on terminal resize until Ctrl+C
      --fields <list>    Select comma-separated fields in list output (default: all where supported)
      --include-hidden   Include projects hidden in the configuration (project list)
  -d, --days <days>      Aggregate this many calendar days, including today (>= 1)
      --month <month>    Aggregate an entire month (YYYY-MM, YYYYMM, or YYYY/MM)
      --clipboard        Copy the output to the clipboard as well as stdout
      --no-clipboard     Disable clipboard copying (summary/template add)
      --template <name>  Use a saved summary template
      --force            Replace an existing template (template add only)
  -h, --help             Show this help
      --no-project       Omit the project column (summary CSV/TSV only)
      --no-date          Omit the date header row (summary CSV/TSV only)
      --daily-total      Append daily totals in minutes (summary CSV/TSV/table only)
      --project          Include the project column (summary/template add)
      --date             Include the date header (summary/template add)
      --version          Show the version`

type App struct {
	NewClient func(workspace, token string) TogglClient
	Updater   Updater
	Out, Err  io.Writer
	In        io.Reader
	Now       func() time.Time
	Version   string
}

type UsageError struct{ Message string }

func (e UsageError) Error() string { return e.Message }
func usage(message string) error   { return UsageError{Message: message} }
func usagef(format string, args ...any) error {
	return UsageError{Message: fmt.Sprintf(format, args...)}
}

func (a App) Run(ctx context.Context, args []string) error {
	if a.Out == nil {
		a.Out = os.Stdout
	}
	if a.Err == nil {
		a.Err = os.Stderr
	}
	if a.In == nil {
		a.In = os.Stdin
	}
	if a.Now == nil {
		a.Now = time.Now
	}
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		if len(args) > 1 {
			return usage("--help does not accept arguments")
		}
		fmt.Fprintln(a.Out, Help)
		return nil
	}
	if args[0] == "--version" {
		if len(args) > 1 {
			return usage("--version does not accept arguments")
		}
		fmt.Fprintln(a.Out, a.Version)
		return nil
	}
	switch args[0] {
	case "completion":
		return a.completion(args[1:])
	case "__complete":
		return a.complete(args[1:])
	case "init":
		return a.init(args[1:])
	case "config":
		return a.showConfig(ctx, args[1:])
	case "project":
		return a.project(ctx, args[1:])
	case "client":
		return a.client(ctx, args[1:])
	case "summary":
		return a.summary(ctx, args[1:])
	case "template":
		return a.template(args[1:])
	case "time-entry":
		return a.timeEntry(ctx, args[1:])
	case "update":
		return a.update(ctx, args[1:])
	default:
		return usagef("unknown command: %s", args[0])
	}
}
