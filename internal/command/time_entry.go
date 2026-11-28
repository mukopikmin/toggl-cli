package command

import (
	"context"
	"sort"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
)

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
	d, e := config.Load()
	if e != nil {
		return e
	}
	loc, e := location(d.Settings)
	if e != nil {
		return e
	}
	from, end, valid := model.MonthDayRange(now, sd, ed, loc)
	if !valid {
		return usage("start and end day must be valid dates")
	}
	entries, e := a.NewClient(d.Settings.Workspace, d.Settings.Token).TimeEntries(ctx, from, end)
	if e != nil {
		return e
	}
	sort.Slice(entries, func(i, j int) bool {
		if order := entries[i].Start.Compare(entries[j].Start); order != 0 {
			return order < 0
		}
		return entries[i].ID < entries[j].ID
	})
	objects := make([]outputTimeEntry, 0, len(entries))
	for _, x := range entries {
		minutes := float64(int(float64(x.ElapsedSeconds(now))/60*100+0.5)) / 100
		start := spreadsheetText(o.format, x.Start.Format(time.RFC3339Nano))
		var stop *string
		if x.Stop != nil {
			formatted := spreadsheetText(o.format, x.Stop.Format(time.RFC3339Nano))
			stop = &formatted
		}
		objects = append(objects, outputTimeEntry{ID: x.ID, Description: spreadsheetText(o.format, x.Description), ProjectID: x.ProjectID, Start: start, Stop: stop, DurationMinutes: minutes})
	}
	return writeList(ctx, a, objects, fields, o.format, tableOutputOptions{watch: o.watch})
}
