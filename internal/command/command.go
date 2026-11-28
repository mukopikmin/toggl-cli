package command

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
	"golang.org/x/term"
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

func (a App) showConfig(ctx context.Context, args []string) error {
	o, e := parseOpts(args, map[string]bool{"format": true, "watch": true})
	if e != nil {
		return e
	}
	if len(o.pos) > 0 {
		return usage("config does not accept positional arguments")
	}
	if err := a.validateTableWatch(o.watch, o.format); err != nil {
		return err
	}
	d, e := config.Load()
	if e != nil {
		return e
	}
	v := map[string]string{"WORKSPACE": d.Settings.Workspace}
	if d.Settings.Timezone != "" {
		v["TIMEZONE"] = d.Settings.Timezone
	}
	if o.format == "json" {
		visible := struct {
			Workspace string `json:"WORKSPACE"`
			Timezone  string `json:"TIMEZONE,omitempty"`
		}{d.Settings.Workspace, d.Settings.Timezone}
		b, _ := json.MarshalIndent(visible, "", "  ")
		fmt.Fprintln(a.Out, string(b))
		return nil
	}
	rows := [][]string{}
	for _, k := range []string{"WORKSPACE", "TIMEZONE"} {
		if x, ok := v[k]; ok {
			rows = append(rows, []string{k, spreadsheetText(o.format, x)})
		}
	}
	if o.format == "table" {
		return a.writeTable(ctx, tableOutputOptions{watch: o.watch}, []string{"Setting", "Value"}, rows, []string{"Key", "Val"})
	} else {
		fmt.Fprintln(a.Out, a.terminalDelimited([]string{"Setting", "Value"}, rows, o.format))
	}
	return nil
}
func (a App) init(args []string) error {
	if len(args) > 0 {
		return usage("init does not accept arguments")
	}
	path, err := config.PrepareCreate()
	if err != nil {
		return err
	}
	r := bufio.NewReader(a.In)
	interactive := false
	if in, ok := a.In.(*os.File); ok {
		interactive = term.IsTerminal(int(in.Fd()))
	}
	ask := func(label, def string, secret bool) (string, error) {
		for {
			if def != "" {
				fmt.Fprintf(a.Out, "%s [%s]: ", label, def)
			} else {
				fmt.Fprintf(a.Out, "%s: ", label)
			}
			var v string
			var e error
			if secret {
				if in, ok := a.In.(*os.File); ok && term.IsTerminal(int(in.Fd())) {
					var value []byte
					value, e = term.ReadPassword(int(in.Fd()))
					fmt.Fprintln(a.Out)
					v = string(value)
				} else {
					v, e = r.ReadString('\n')
				}
			} else {
				v, e = r.ReadString('\n')
			}
			if e != nil && !(errors.Is(e, io.EOF) && v != "") {
				if errors.Is(e, io.EOF) {
					return "", fmt.Errorf("Input ended before %s was provided", label)
				}
				return "", fmt.Errorf("Unable to read %s: %w", label, e)
			}
			v = strings.TrimSpace(v)
			if v != "" {
				return v, nil
			}
			if def != "" {
				return def, nil
			}
			if !interactive {
				return "", fmt.Errorf("%s is required", label)
			}
			fmt.Fprintf(a.Out, "%s is required. Please enter a value.\n", label)
		}
	}
	w, e := ask("Workspace", "", false)
	if e != nil {
		return e
	}
	t, e := ask("API token", "", true)
	if e != nil {
		return e
	}
	z, e := ask("Timezone", "Asia/Tokyo", false)
	if e != nil {
		return e
	}
	settings := model.Settings{Workspace: w, Token: t, Timezone: z}
	if err := config.Create(path, settings); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "Wrote %s\n", config.DisplayPath)
	return nil
}

func location(settings model.Settings) (*time.Location, error) {
	if settings.Timezone == "" {
		return time.Local, nil
	}
	return time.LoadLocation(settings.Timezone)
}
