package command

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
	"github.com/mukopikmin/toggl-cli/internal/template"
	"github.com/mukopikmin/toggl-cli/internal/toggl"
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
	Client   toggl.Client
	Out, Err io.Writer
	In       io.Reader
	Now      func() time.Time
	Version  string
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
		return a.update(args[1:])
	default:
		return usagef("unknown command: %s", args[0])
	}
}

type outputTimeEntry struct {
	ID              int64   `json:"id"`
	Description     string  `json:"description"`
	ProjectID       *int64  `json:"project_id"`
	Start           string  `json:"start"`
	Stop            *string `json:"stop"`
	DurationMinutes float64 `json:"duration_minutes"`
}

func loadCfg() (config.Document, error) { return config.Load() }
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
	d, e := loadCfg()
	if e != nil {
		return e
	}
	v := map[string]string{"WORKSPACE": d.Config.Workspace}
	if d.Config.Timezone != "" {
		v["TIMEZONE"] = d.Config.Timezone
	}
	if o.format == "json" {
		visible := struct {
			Workspace string `json:"WORKSPACE"`
			Timezone  string `json:"TIMEZONE,omitempty"`
		}{d.Config.Workspace, d.Config.Timezone}
		b, _ := json.MarshalIndent(visible, "", "  ")
		fmt.Fprintln(a.Out, string(b))
		return nil
	}
	rows := [][]string{}
	for _, k := range []string{"WORKSPACE", "TIMEZONE"} {
		if x, ok := v[k]; ok {
			rows = append(rows, []string{k, x})
		}
	}
	if o.format == "table" {
		return a.writeTable(ctx, tableOutputOptions{watch: o.watch}, []string{"Setting", "Value"}, rows, []string{"Key", "Val"})
	} else {
		fmt.Fprintln(a.Out, delimited([]string{"Setting", "Value"}, rows, o.format))
	}
	return nil
}
func (a App) init(args []string) error {
	if len(args) > 0 {
		return usage("init does not accept arguments")
	}
	home := os.Getenv("HOME")
	if home == "" {
		return errors.New("HOME environment variable not set")
	}
	p := config.Path(home)
	if _, e := os.Stat(p); e == nil {
		return fmt.Errorf("%s already exists", config.DisplayPath)
	} else if !os.IsNotExist(e) {
		return e
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
	if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		return e
	}
	text, e := config.EncodeInitial(w, t, z)
	if e != nil {
		return e
	}
	file, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	if _, e = file.WriteString(text); e != nil {
		file.Close()
		os.Remove(p)
		return e
	}
	if e = file.Close(); e != nil {
		os.Remove(p)
		return e
	}
	fmt.Fprintf(a.Out, "Wrote %s\n", config.DisplayPath)
	return nil
}

func (a App) project(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usage("project requires a subcommand: list, reorder, or sync")
	}
	var listOptions opts
	switch args[0] {
	case "list":
		var err error
		listOptions, err = parseOpts(args[1:], map[string]bool{"format": true, "watch": true})
		if err != nil {
			return err
		}
		if len(listOptions.pos) > 0 {
			return usage("project list does not accept positional arguments")
		}
		if err := a.validateTableWatch(listOptions.watch, listOptions.format); err != nil {
			return err
		}
	case "sync":
		if len(args) > 1 {
			return usage("project sync does not accept arguments")
		}
	case "reorder":
		if len(args) > 1 {
			return usage("project reorder does not accept arguments")
		}
	default:
		return usagef("unknown project subcommand: %s", args[0])
	}
	d, e := loadCfg()
	if e != nil {
		return e
	}
	ps, e := a.Client.Projects(ctx, d.Config)
	if e != nil {
		return e
	}
	switch args[0] {
	case "list":
		ps = model.VisibleSorted(ps)
		if listOptions.format == "json" {
			b, _ := json.MarshalIndent(ps, "", "  ")
			fmt.Fprintln(a.Out, string(b))
		} else if listOptions.format == "table" {
			rows := [][]string{}
			for _, p := range ps {
				rows = append(rows, []string{p.DisplayName})
			}
			options := tableOutputOptions{watch: listOptions.watch, truncateProjects: true}
			return a.writeTable(ctx, options, []string{"Project"}, rows, []string{"Proj."})
		} else {
			rows := make([][]string, len(ps))
			for i, p := range ps {
				rows[i] = []string{p.DisplayName}
			}
			fmt.Fprintln(a.Out, delimited(nil, rows, listOptions.format))
		}
		return nil
	case "sync":
		count, err := d.SyncProjects(ps)
		if err != nil {
			return err
		}
		if count == 0 {
			fmt.Fprintln(a.Out, "All active projects are already configured")
			return nil
		}
		fmt.Fprintf(a.Out, "Added %d project(s) to the config file\n", count)
		return nil
	case "reorder":
		return a.reorderProjects(d, model.VisibleSorted(ps))
	}
	return nil
}

func location(c config.Config) (*time.Location, error) {
	if c.Timezone == "" {
		return time.Local, nil
	}
	return time.LoadLocation(c.Timezone)
}
func parseMonth(v string) (time.Time, error) {
	if !regexp.MustCompile(`^\d{4}[-/]?\d{2}$`).MatchString(v) {
		return time.Time{}, usage("month must use YYYY-MM, YYYYMM, or YYYY/MM")
	}
	v = strings.ReplaceAll(v, "/", "-")
	if len(v) == 6 {
		v = v[:4] + "-" + v[4:]
	}
	t, err := time.Parse("2006-01", v)
	if err != nil {
		return time.Time{}, usage("month must be a valid month")
	}
	return t, nil
}
func (a App) summary(ctx context.Context, args []string) error {
	allowed := summaryFlags()
	allowed["template"] = true
	allowed["month"] = true
	allowed["watch"] = true
	allowed["daily-total"] = true
	o, e := parseOpts(args, allowed)
	if e != nil {
		return e
	}
	if o.monthSet {
		if o.days != nil || len(o.pos) > 0 {
			return usage("summary accepts only one of start and end date, --days, or --month")
		}
	}
	base := model.SummaryOptions{}
	if o.templateSet {
		if err := template.ValidateName(o.template); err != nil {
			return usage(err.Error())
		}
		d, err := template.Load()
		if err != nil {
			return err
		}
		var exists bool
		base, exists = d.Templates[o.template]
		if !exists {
			return fmt.Errorf("unknown template: %s", o.template)
		}
	}
	settings, e := summaryOptions(base, o)
	if e != nil {
		return e
	}
	if o.monthSet {
		monthStart, err := parseMonth(o.month)
		if err != nil {
			return err
		}
		settings.Days = nil
		settings.StartDate = monthStart.Format(time.DateOnly)
		settings.EndDate = monthStart.AddDate(0, 1, -1).Format(time.DateOnly)
	}
	if o.dailyTotal && settings.Format == "json" {
		return usage("--daily-total cannot be used with json format")
	}
	if e = settings.Validate(true); e != nil {
		return usage(e.Error())
	}
	if err := a.validateTableWatch(o.watch, settings.Format); err != nil {
		return err
	}
	d, e := loadCfg()
	if e != nil {
		return e
	}
	loc, e := location(d.Config)
	if e != nil {
		return e
	}
	now := a.Now()
	from, to, e := settings.DateRange(now, loc)
	if e != nil {
		return usage(e.Error())
	}
	start := model.StartOfDate(from, loc)
	end := model.StartOfDate(to.AddDate(0, 0, 1), loc)
	entries, e := a.Client.TimeEntries(ctx, d.Config, start, end)
	if e != nil {
		return e
	}
	sum := model.Summarize(entries, loc, now)
	return a.writeSummary(ctx, d.Config, settings, from, to, sum, o.watch, o.dailyTotal)
}

func (a App) writeSummary(ctx context.Context, cfg config.Config, settings model.SummaryOptions, from, to time.Time, sum model.Summary, watch, dailyTotal bool) error {
	var text string
	if settings.Format == "json" {
		b, _ := json.MarshalIndent(sum, "", "  ")
		text = string(b)
	} else {
		ps, e := a.Client.Projects(ctx, cfg)
		if e != nil {
			return e
		}
		ps = model.VisibleSorted(ps)
		days := model.DateRange(from, to)
		rows := [][]string{}
		for _, p := range ps {
			r := []string{p.DisplayName}
			for _, day := range days {
				v := sum[day][p.ID]
				if v == 0 {
					r = append(r, "")
				} else {
					r = append(r, strconv.FormatFloat(float64(int(v*100+0.5))/100, 'f', -1, 64))
				}
			}
			rows = append(rows, r)
		}
		if dailyTotal {
			totals := model.DailyTotals(sum, ps, days)
			r := []string{"Total"}
			for _, day := range days {
				v := totals[day]
				r = append(r, strconv.FormatFloat(float64(int(v*100+0.5))/100, 'f', -1, 64))
			}
			rows = append(rows, r)
		}
		headers := append([]string{"Project"}, days...)
		if settings.Format == "table" {
			options := tableOutputOptions{watch: watch, clipboard: settings.Clipboard, truncateProjects: true}
			return a.writeTable(ctx, options, headers, rows, summaryHeaderVariants(days, from, to)...)
		}
		if settings.NoProject {
			headers = days
			for i := range rows {
				rows[i] = rows[i][1:]
			}
		}
		if settings.NoDate {
			headers = nil
		}
		text = delimited(headers, rows, settings.Format)
	}
	fmt.Fprintln(a.Out, text)
	if settings.Clipboard {
		return clipboard(text)
	}
	return nil
}

func summaryHeaderVariants(days []string, from, to time.Time) [][]string {
	compact := append([]string{"Proj."}, days...)
	minimal := append([]string{"Proj."}, days...)
	sameYear := from.Year() == to.Year()
	sameMonth := sameYear && from.Month() == to.Month()
	var previousMonth string
	for i, day := range days {
		if sameYear {
			compact[i+1] = day[len(day)-len("01-02"):]
		}
		month := day[:len(day)-len("-02")]
		// Keep the first date of each month identifiable in a multi-month table.
		if sameMonth || month == previousMonth {
			minimal[i+1] = strings.TrimPrefix(day[len(day)-len("02"):], "0")
		} else {
			minimal[i+1] = compact[i+1]
		}
		previousMonth = month
	}
	return [][]string{compact, minimal}
}

func (a App) timeEntry(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usage("time-entry requires a subcommand: list")
	}
	if args[0] != "list" {
		return usagef("unknown time-entry subcommand: %s", args[0])
	}
	o, e := parseOpts(args[1:], map[string]bool{"format": true, "watch": true})
	if e != nil {
		return e
	}
	if len(o.pos) != 2 {
		return usage("time-entry list requires start and end day")
	}
	if err := a.validateTableWatch(o.watch, o.format); err != nil {
		return err
	}
	now := a.Now()
	sd, ok := parseUnsignedInteger(o.pos[0])
	if !ok {
		return usage("start and end day must be valid integers")
	}
	ed, ok := parseUnsignedInteger(o.pos[1])
	if !ok {
		return usage("start and end day must be valid integers")
	}
	if sd < 1 || ed < 1 || sd > 31 || ed > 31 {
		return usage("start and end day must be valid dates")
	}
	if sd > ed {
		return usage("start day must not be after end day")
	}
	d, e := loadCfg()
	if e != nil {
		return e
	}
	loc, e := location(d.Config)
	if e != nil {
		return e
	}
	y, m, _ := now.In(loc).Date()
	lastDay := time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if ed > lastDay {
		return usage("start and end day must be valid dates")
	}
	from := model.StartOfDate(time.Date(y, m, sd, 0, 0, 0, 0, time.UTC), loc)
	end := model.StartOfDate(time.Date(y, m, ed+1, 0, 0, 0, 0, time.UTC), loc)
	entries, e := a.Client.TimeEntries(ctx, d.Config, from, end)
	if e != nil {
		return e
	}
	sort.Slice(entries, func(i, j int) bool {
		if order := entries[i].Start.Compare(entries[j].Start); order != 0 {
			return order < 0
		}
		return entries[i].ID < entries[j].ID
	})
	rows := [][]string{}
	objects := []outputTimeEntry{}
	for _, x := range entries {
		minutes := float64(int(float64(x.ElapsedSeconds(now))/60*100+0.5)) / 100
		start := x.Start.Format(time.RFC3339Nano)
		pid := ""
		if x.ProjectID != nil {
			pid = strconv.FormatInt(*x.ProjectID, 10)
		}
		stop := ""
		var stopJSON *string
		if x.Stop != nil {
			stop = x.Stop.Format(time.RFC3339Nano)
			stopJSON = &stop
		}
		rows = append(rows, []string{strconv.FormatInt(x.ID, 10), x.Description, pid, start, stop, strconv.FormatFloat(minutes, 'f', -1, 64)})
		objects = append(objects, outputTimeEntry{ID: x.ID, Description: x.Description, ProjectID: x.ProjectID, Start: start, Stop: stopJSON, DurationMinutes: minutes})
	}
	headers := []string{"id", "description", "project_id", "start", "stop", "duration_minutes"}
	if o.format == "json" {
		b, _ := json.MarshalIndent(objects, "", "  ")
		fmt.Fprintln(a.Out, string(b))
	} else if o.format == "table" {
		compactHeaders := []string{"id", "desc", "proj_id", "start", "stop", "mins"}
		return a.writeTable(ctx, tableOutputOptions{watch: o.watch}, headers, rows, compactHeaders)
	} else {
		fmt.Fprintln(a.Out, delimited(headers, rows, o.format))
	}
	return nil
}
