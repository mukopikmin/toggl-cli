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
	ID              int64
	Description     string
	ProjectID       *int64
	Start           string
	Stop            *string
	DurationMinutes float64
}

var timeEntryFields = listFields[outputTimeEntry]{
	{"id", "id", "id", func(e outputTimeEntry) any { return e.ID }},
	{"description", "description", "desc", func(e outputTimeEntry) any { return e.Description }},
	{"project_id", "project_id", "proj_id", func(e outputTimeEntry) any { return optionalListValue(e.ProjectID) }},
	{"start", "start", "start", func(e outputTimeEntry) any { return e.Start }},
	{"stop", "stop", "stop", func(e outputTimeEntry) any { return optionalListValue(e.Stop) }},
	{"duration_minutes", "duration_minutes", "mins", func(e outputTimeEntry) any { return e.DurationMinutes }},
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
	var listOptions projectListOptions
	switch args[0] {
	case "list":
		var err error
		listOptions, err = a.parseProjectListOptions(args[1:])
		if err != nil {
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
		return a.writeProjectList(ctx, ps, listOptions)
	case "sync":
		configured := d.Config.Projects
		sort.Slice(ps, func(i, j int) bool { return ps[i].ID < ps[j].ID })
		text := d.Text
		count := 0
		for _, p := range ps {
			if _, ok := configured[p.ID]; ok {
				continue
			}
			if !strings.HasSuffix(text, "\n") {
				text += "\n"
			}
			name := strings.ReplaceAll(strings.ReplaceAll(p.Name, "\r\n", "\n"), "\r", "\n")
			text += fmt.Sprintf("\n# %s\n[projects.\"%d\"]\nhidden = false\n", strings.ReplaceAll(name, "\n", "\n# "), p.ID)
			count++
		}
		if count == 0 {
			fmt.Fprintln(a.Out, "All active projects are already configured")
			return nil
		}
		if e = os.WriteFile(d.Path, []byte(text), 0600); e != nil {
			return e
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
func (a App) summary(ctx context.Context, args []string) error {
	allowed := summaryFlags()
	allowed["template"] = true
	allowed["watch"] = true
	allowed["daily-total"] = true
	o, e := parseOpts(args, allowed)
	if e != nil {
		return e
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
	entries, e := a.Client.TimeEntries(ctx, d.Config, from, to)
	if e != nil {
		return e
	}
	sum, e := model.Summarize(entries, loc, now)
	if e != nil {
		return e
	}
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
			r := []string{spreadsheetText(settings.Format, p.DisplayName)}
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
		text = a.terminalDelimited(headers, rows, settings.Format)
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
	o, e := parseOpts(args[1:], map[string]bool{"format": true, "watch": true, "fields": true})
	if e != nil {
		return e
	}
	if len(o.pos) != 2 {
		return usage("time-entry list requires start and end day")
	}
	if err := a.validateTableWatch(o.watch, o.format); err != nil {
		return err
	}
	fields, e := timeEntryFields.selectFields("time-entry", o.fields)
	if e != nil {
		return e
	}
	now := a.Now()
	y, m, _ := now.Date()
	sd, ok := parseUnsignedInteger(o.pos[0])
	if !ok {
		return usage("start and end day must be valid integers")
	}
	ed, ok := parseUnsignedInteger(o.pos[1])
	if !ok {
		return usage("start and end day must be valid integers")
	}
	lastDay := time.Date(y, m+1, 0, 0, 0, 0, 0, time.Local).Day()
	if sd < 1 || ed < 1 || sd > lastDay || ed > lastDay {
		return usage("start and end day must be valid dates")
	}
	from := time.Date(y, m, sd, 0, 0, 0, 0, time.Local)
	to := time.Date(y, m, ed, 0, 0, 0, 0, time.Local)
	if from.After(to) {
		return usage("start day must not be after end day")
	}
	d, e := loadCfg()
	if e != nil {
		return e
	}
	entries, e := a.Client.TimeEntries(ctx, d.Config, from, to)
	if e != nil {
		return e
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Start < entries[j].Start || entries[i].Start == entries[j].Start && entries[i].ID < entries[j].ID
	})
	objects := make([]outputTimeEntry, 0, len(entries))
	for _, x := range entries {
		seconds := x.DurationSeconds
		if seconds < 0 {
			seconds = a.Now().Unix() + seconds
		}
		minutes := float64(int(float64(seconds)/60*100+0.5)) / 100
		objects = append(objects, outputTimeEntry{ID: x.ID, Description: x.Description, ProjectID: x.ProjectID, Start: x.Start, Stop: x.Stop, DurationMinutes: minutes})
	}
	return writeList(ctx, a, objects, fields, o.format, tableOutputOptions{watch: o.watch})
}
