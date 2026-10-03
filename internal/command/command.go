package command

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
	"github.com/mukopikmin/toggl-cli/internal/toggl"
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
		fmt.Fprintln(a.Out, Help)
		return nil
	}
	if args[0] == "--version" {
		if len(args) > 1 {
			return errors.New("--version does not accept arguments")
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
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

type opts struct {
	format, sep                  string
	days                         *int
	noProject, noDate, clipboard bool
	pos                          []string
}

func parseOpts(args []string, allowed map[string]bool) (opts, error) {
	o := opts{format: "csv", sep: "\t"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		key, val, has := arg, "", false
		if strings.HasPrefix(arg, "--") && strings.Contains(arg, "=") {
			key, val, has = strings.Cut(arg, "=")
		}
		take := func() (string, error) {
			if has {
				return val, nil
			}
			if i+1 >= len(args) {
				return "", fmt.Errorf("option %s requires a value", key)
			}
			i++
			return args[i], nil
		}
		switch key {
		case "-f", "--format":
			if !allowed["format"] {
				return o, fmt.Errorf("unknown option: %s", key)
			}
			v, e := take()
			if e != nil {
				return o, e
			}
			o.format = v
		case "-s", "--separator":
			if !allowed["separator"] {
				return o, fmt.Errorf("unknown option: %s", key)
			}
			v, e := take()
			if e != nil {
				return o, e
			}
			o.sep = v
		case "-d", "--days":
			if !allowed["days"] {
				return o, fmt.Errorf("unknown option: %s", key)
			}
			v, e := take()
			if e != nil {
				return o, e
			}
			n, e := strconv.Atoi(v)
			if e != nil || n < 0 {
				return o, errors.New("days must be a non-negative integer")
			}
			o.days = &n
		case "--no-project":
			o.noProject = true
		case "--no-date":
			o.noDate = true
		case "--clipboard":
			o.clipboard = true
		default:
			if strings.HasPrefix(arg, "-") {
				return o, fmt.Errorf("unknown option: %s", arg)
			}
			o.pos = append(o.pos, arg)
		}
	}
	if o.format != "csv" && o.format != "json" && o.format != "table" {
		return o, errors.New("format must be csv, json, or table")
	}
	if o.sep == "" {
		return o, errors.New("separator must not be empty")
	}
	return o, nil
}
func loadCfg() (config.Document, error) { return config.Load() }
func (a App) showConfig(args []string) error {
	o, e := parseOpts(args, map[string]bool{"format": true})
	if e != nil {
		return e
	}
	if len(o.pos) > 0 {
		return errors.New("config does not accept positional arguments")
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
		b, _ := json.MarshalIndent(v, "", "  ")
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
		return errors.New("init does not accept arguments")
	}
	home := os.Getenv("HOME")
	if home == "" {
		return errors.New("HOME environment variable not set")
	}
	p := config.Path(home)
	if _, e := os.Stat(p); e == nil {
		return fmt.Errorf("%s already exists", config.DisplayPath)
	}
	r := bufio.NewReader(a.In)
	ask := func(label, def string) (string, error) {
		for {
			if def != "" {
				fmt.Fprintf(a.Out, "%s [%s]: ", label, def)
			} else {
				fmt.Fprintf(a.Out, "%s: ", label)
			}
			v, e := r.ReadString('\n')
			v = strings.TrimSpace(v)
			if v != "" {
				return v, nil
			}
			if def != "" {
				return def, nil
			}
			if e != nil {
				return "", fmt.Errorf("Input ended before %s was provided", label)
			}
		}
	}
	w, e := ask("Workspace", "")
	if e != nil {
		return e
	}
	t, e := ask("API token", "")
	if e != nil {
		return e
	}
	z, e := ask("Timezone", "Asia/Tokyo")
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		return e
	}
	text := fmt.Sprintf("workspace = %q\ntoken = %q\ntimezone = %q\n", w, t, z)
	if e = os.WriteFile(p, []byte(text), 0600); e != nil {
		return e
	}
	fmt.Fprintf(a.Out, "Wrote %s\n", config.DisplayPath)
	return nil
}

func (a App) project(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("project requires a subcommand: list, reorder, or sync")
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
		o, e := parseOpts(args[1:], map[string]bool{"format": true})
		if e != nil {
			return e
		}
		if len(o.pos) > 0 {
			return errors.New("project list does not accept positional arguments")
		}
		ps = model.VisibleSorted(ps)
		if o.format == "json" {
			b, _ := json.MarshalIndent(ps, "", "  ")
			fmt.Fprintln(a.Out, string(b))
		} else if o.format == "table" {
			rows := [][]string{}
			for _, p := range ps {
				rows = append(rows, []string{p.DisplayName})
			}
			fmt.Fprintln(a.Out, table([]string{"Project"}, rows))
		} else {
			for _, p := range ps {
				fmt.Fprintln(a.Out, p.DisplayName)
			}
		}
		return nil
	case "sync":
		if len(args) > 1 {
			return errors.New("project sync does not accept arguments")
		}
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
			text += fmt.Sprintf("\n# %s\n[projects.\"%d\"]\nhidden = false\n", strings.ReplaceAll(p.Name, "\n", "\n# "), p.ID)
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
		return errors.New("project reorder requires an interactive terminal")
	default:
		return fmt.Errorf("unknown project subcommand: %s", args[0])
	}
}

func parseDate(v string) (time.Time, error) {
	t, e := time.Parse(time.DateOnly, v)
	if e != nil {
		return t, errors.New("start and end date must use YYYY-MM-DD")
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
	o, e := parseOpts(args, map[string]bool{"format": true, "separator": true, "days": true})
	if e != nil {
		return e
	}
	if o.format == "table" && (o.noProject || o.noDate) {
		return errors.New("--separator, --no-project, and --no-date cannot be used with table format")
	}
	d, e := loadCfg()
	if e != nil {
		return e
	}
	loc, e := location(d.Config)
	if e != nil {
		return e
	}
	var from, to time.Time
	if o.days != nil {
		if len(o.pos) > 0 {
			return errors.New("summary accepts either start and end date or --days, not both")
		}
		now := a.Now().In(loc)
		to = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		from = to.AddDate(0, 0, -*o.days)
	} else {
		if len(o.pos) != 2 {
			return errors.New("summary requires start and end date or --days")
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
		return errors.New("start date must not be after end date")
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
		return errors.New("time-entry requires a subcommand: list")
	}
	if args[0] != "list" {
		return fmt.Errorf("unknown time-entry subcommand: %s", args[0])
	}
	o, e := parseOpts(args[1:], map[string]bool{"format": true, "separator": true})
	if e != nil {
		return e
	}
	if len(o.pos) != 2 {
		return errors.New("time-entry list requires start and end day")
	}
	now := a.Now()
	y, m, _ := now.Date()
	sd, e := strconv.Atoi(o.pos[0])
	if e != nil {
		return errors.New("start and end day must be valid integers")
	}
	ed, e := strconv.Atoi(o.pos[1])
	if e != nil {
		return errors.New("start and end day must be valid integers")
	}
	from := time.Date(y, m, sd, 0, 0, 0, 0, time.Local)
	to := time.Date(y, m, ed, 0, 0, 0, 0, time.Local)
	if from.Month() != m || to.Month() != m {
		return errors.New("start and end day must be valid dates")
	}
	if from.After(to) {
		return errors.New("start day must not be after end day")
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
	objects := []map[string]any{}
	for _, x := range entries {
		seconds := x.DurationSeconds
		if seconds < 0 {
			seconds = a.Now().Unix() + seconds
		}
		minutes := float64(int(float64(seconds)/60*100+0.5)) / 100
		pid := ""
		var jp any = nil
		if x.ProjectID != nil {
			pid = strconv.FormatInt(*x.ProjectID, 10)
			jp = *x.ProjectID
		}
		stop := ""
		var js any = nil
		if x.Stop != nil {
			stop = *x.Stop
			js = *x.Stop
		}
		rows = append(rows, []string{strconv.FormatInt(x.ID, 10), x.Description, pid, x.Start, stop, strconv.FormatFloat(minutes, 'f', -1, 64)})
		objects = append(objects, map[string]any{"id": x.ID, "description": x.Description, "project_id": jp, "start": x.Start, "stop": js, "duration_minutes": minutes})
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
func (a App) update(args []string) error {
	channel := "stable"
	if len(args) == 2 && args[0] == "--channel" {
		channel = args[1]
	} else if len(args) > 0 {
		return errors.New("update accepts only --channel stable|nightly")
	}
	if channel != "stable" && channel != "nightly" {
		return errors.New("channel must be stable or nightly")
	}
	return errors.New("automatic update is unavailable in source builds; use install.sh")
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
func table(h []string, rows [][]string) string {
	width := make([]int, len(h))
	for i, v := range h {
		width[i] = len(v)
	}
	for _, r := range rows {
		for i, v := range r {
			if len(v) > width[i] {
				width[i] = len(v)
			}
		}
	}
	border := func() string {
		var b strings.Builder
		b.WriteByte('+')
		for _, w := range width {
			b.WriteString(strings.Repeat("-", w+2))
			b.WriteByte('+')
		}
		return b.String()
	}
	line := func(r []string) string {
		var b strings.Builder
		b.WriteByte('|')
		for i, v := range r {
			fmt.Fprintf(&b, " %-*s |", width[i], v)
		}
		return b.String()
	}
	out := []string{border(), line(h), border()}
	for _, r := range rows {
		out = append(out, line(r))
	}
	return strings.Join(append(out, border()), "\n")
}
func clipboard(text string) error {
	var candidates [][]string
	switch runtime.GOOS {
	case "darwin":
		candidates = [][]string{{"pbcopy"}}
	case "windows":
		candidates = [][]string{{"clip"}, {"powershell.exe", "-NoProfile", "-Command", "Set-Clipboard"}}
	case "linux":
		candidates = [][]string{{"wl-copy"}, {"xclip", "-selection", "clipboard"}, {"xsel", "--clipboard", "--input"}}
	}
	for _, c := range candidates {
		if _, e := exec.LookPath(c[0]); e != nil {
			continue
		}
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdin = strings.NewReader(text)
		if cmd.Run() == nil {
			return nil
		}
	}
	return errors.New("Could not copy output to the clipboard.")
}
