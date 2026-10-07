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
	"github.com/mukopikmin/toggl-cli/internal/toggl"
	"golang.org/x/term"
)

const Help = `Usage:
  toggl summary <start-date> <end-date> [options]
  toggl summary --days <days> [options]
  toggl time-entry list <start-day> <end-day> [options]
  toggl project list [options]
  toggl project reorder
  toggl project sync
  toggl config [options]
  toggl init
  toggl update [--channel stable|nightly]

Commands:
  init        Create the configuration file
  project     List, reorder, and sync projects
  time-entry  List individual time entries for a range of days
  config      Show configuration values
  summary     Summarize time entries for a range of days
  update      Update the installed Toggl CLI binary

Options:
  -s, --separator <text> Set the CSV delimiter (default: tab; CSV only)
  -f, --format <format>  Set the output format: csv, json, or table (default: csv)
  -d, --days <days>      Aggregate from this many days ago through today
      --clipboard        Copy the output to the clipboard as well as stdout
  -h, --help             Show this help
      --no-project       Omit the project column (summary CSV only)
      --no-date          Omit the date header row from CSV output
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
	case "init":
		return a.init(args[1:])
	case "config":
		return a.showConfig(args[1:])
	case "project":
		return a.project(ctx, args[1:])
	case "summary":
		return a.summary(ctx, args[1:])
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
func (a App) showConfig(args []string) error {
	o, e := parseOpts(args, map[string]bool{"format": true})
	if e != nil {
		return e
	}
	if len(o.pos) > 0 {
		return usage("config does not accept positional arguments")
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
		fmt.Fprintln(a.Out, table([]string{"Setting", "Value"}, rows))
	} else {
		for _, r := range rows {
			fmt.Fprintf(a.Out, "%s=%s\n", r[0], r[1])
		}
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
		listOptions, err = parseOpts(args[1:], map[string]bool{"format": true})
		if err != nil {
			return err
		}
		if len(listOptions.pos) > 0 {
			return usage("project list does not accept positional arguments")
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
			fmt.Fprintln(a.Out, table([]string{"Project"}, rows))
		} else {
			names := make([]string, len(ps))
			for i, p := range ps {
				names[i] = p.DisplayName
			}
			fmt.Fprintln(a.Out, strings.Join(names, "\n"))
		}
		return nil
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

func parseDate(v string) (time.Time, error) {
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(v) {
		return time.Time{}, usage("start and end date must use YYYY-MM-DD")
	}
	t, e := time.Parse(time.DateOnly, v)
	if e != nil {
		return t, usage("start and end date must be valid dates")
	}
	return t, nil
}
func location(c config.Config) (*time.Location, error) {
	if c.Timezone == "" {
		return time.Local, nil
	}
	return time.LoadLocation(c.Timezone)
}
func (a App) summary(ctx context.Context, args []string) error {
	o, e := parseOpts(args, map[string]bool{"format": true, "separator": true, "days": true, "no-project": true, "no-date": true, "clipboard": true})
	if e != nil {
		return e
	}
	if o.format == "table" && (o.separatorSet || o.noProject || o.noDate) {
		return usage("--separator, --no-project, and --no-date cannot be used with table format")
	}
	var from, to time.Time
	if o.days != nil {
		if len(o.pos) > 0 {
			return usage("summary accepts either start and end date or --days, not both")
		}
	} else {
		if len(o.pos) != 2 {
			return usage("summary requires start and end date or --days")
		}
		from, e = parseDate(o.pos[0])
		if e != nil {
			return e
		}
		to, e = parseDate(o.pos[1])
		if e != nil {
			return e
		}
	}
	if from.After(to) {
		return usage("start date must not be after end date")
	}
	d, e := loadCfg()
	if e != nil {
		return e
	}
	loc, e := location(d.Config)
	if e != nil {
		return e
	}
	if o.days != nil {
		now := a.Now().In(loc)
		to = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		from = to.AddDate(0, 0, -*o.days)
	}
	entries, e := a.Client.TimeEntries(ctx, d.Config, from, to)
	if e != nil {
		return e
	}
	sum, e := model.Summarize(entries, loc, a.Now())
	if e != nil {
		return e
	}
	var text string
	if o.format == "json" {
		b, _ := json.MarshalIndent(sum, "", "  ")
		text = string(b)
	} else {
		ps, e := a.Client.Projects(ctx, d.Config)
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
		headers := append([]string{"Project"}, days...)
		if o.format == "table" {
			text = table(headers, rows)
		} else {
			var lines []string
			if !o.noDate {
				h := headers
				if o.noProject {
					h = days
				}
				lines = append(lines, strings.Join(h, o.sep))
			}
			for _, r := range rows {
				if o.noProject {
					r = r[1:]
				}
				lines = append(lines, strings.Join(r, o.sep))
			}
			text = strings.Join(lines, "\n")
		}
	}
	fmt.Fprintln(a.Out, text)
	if o.clipboard {
		return clipboard(text)
	}
	return nil
}

func (a App) timeEntry(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usage("time-entry requires a subcommand: list")
	}
	if args[0] != "list" {
		return usagef("unknown time-entry subcommand: %s", args[0])
	}
	o, e := parseOpts(args[1:], map[string]bool{"format": true, "separator": true})
	if e != nil {
		return e
	}
	if len(o.pos) != 2 {
		return usage("time-entry list requires start and end day")
	}
	if o.sep == "" {
		return usage("separator must not be empty")
	}
	if o.format == "table" && o.separatorSet {
		return usage("--separator cannot be used with table format")
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
	rows := [][]string{}
	objects := []outputTimeEntry{}
	for _, x := range entries {
		seconds := x.DurationSeconds
		if seconds < 0 {
			seconds = a.Now().Unix() + seconds
		}
		minutes := float64(int(float64(seconds)/60*100+0.5)) / 100
		pid := ""
		if x.ProjectID != nil {
			pid = strconv.FormatInt(*x.ProjectID, 10)
		}
		stop := ""
		if x.Stop != nil {
			stop = *x.Stop
		}
		rows = append(rows, []string{strconv.FormatInt(x.ID, 10), x.Description, pid, x.Start, stop, strconv.FormatFloat(minutes, 'f', -1, 64)})
		objects = append(objects, outputTimeEntry{ID: x.ID, Description: x.Description, ProjectID: x.ProjectID, Start: x.Start, Stop: x.Stop, DurationMinutes: minutes})
	}
	headers := []string{"id", "description", "project_id", "start", "stop", "duration_minutes"}
	if o.format == "json" {
		b, _ := json.MarshalIndent(objects, "", "  ")
		fmt.Fprintln(a.Out, string(b))
	} else if o.format == "table" {
		fmt.Fprintln(a.Out, table(headers, rows))
	} else {
		fmt.Fprintln(a.Out, csv(headers, rows, o.sep))
	}
	return nil
}
func csv(h []string, rows [][]string, sep string) string {
	all := append([][]string{h}, rows...)
	var lines []string
	for _, r := range all {
		for i, v := range r {
			if strings.ContainsAny(v, "\r\n\"") || strings.Contains(v, sep) {
				r[i] = "\"" + strings.ReplaceAll(v, "\"", "\"\"") + "\""
			}
		}
		lines = append(lines, strings.Join(r, sep))
	}
	return strings.Join(lines, "\n")
}
