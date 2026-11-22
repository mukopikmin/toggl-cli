package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

var projectColumns = map[string]struct {
	header      string
	shortHeader string
	value       func(model.Project) string
}{
	"id":          {"ID", "ID", func(p model.Project) string { return strconv.FormatInt(p.ID, 10) }},
	"displayName": {"Project", "Proj.", func(p model.Project) string { return p.DisplayName }},
	"name":        {"Toggl name", "Name", func(p model.Project) string { return p.Name }},
	"hidden":      {"Hidden", "Hidden", func(p model.Project) string { return strconv.FormatBool(p.Hidden) }},
}

func (a App) parseProjectListOptions(args []string) (opts, error) {
	o, err := parseOpts(args, map[string]bool{
		"format": true, "watch": true, "columns": true, "include-hidden": true,
	})
	if err != nil {
		return o, err
	}
	if len(o.pos) > 0 {
		return o, usage("project list does not accept positional arguments")
	}
	if o.columns != nil && o.format == "json" {
		return o, usage("--columns requires table, csv, or tsv format")
	}
	for i, name := range o.columns {
		name = strings.TrimSpace(name)
		if name == "" {
			return o, usage("--columns requires a comma-separated list of column names")
		}
		if _, ok := projectColumns[name]; !ok {
			return o, usagef("unknown project column %q (expected id, displayName, name, or hidden)", name)
		}
		o.columns[i] = name
	}
	return o, a.validateTableWatch(o.watch, o.format)
}

func (a App) writeProjectList(ctx context.Context, projects []model.Project, o opts) error {
	projects = model.SortedProjects(projects, o.includeHidden)
	if o.format == "json" {
		b, err := json.MarshalIndent(projects, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(a.Out, string(b))
		return err
	}

	columns := o.columns
	if columns == nil {
		columns = []string{"displayName"}
	}
	rows := make([][]string, len(projects))
	for i, p := range projects {
		row := make([]string, len(columns))
		for j, name := range columns {
			row[j] = projectColumns[name].value(p)
		}
		rows[i] = row
	}
	if o.format == "table" {
		headers := make([]string, len(columns))
		shortHeaders := make([]string, len(columns))
		for i, name := range columns {
			column := projectColumns[name]
			headers[i], shortHeaders[i] = column.header, column.shortHeader
		}
		firstIsName := columns[0] == "displayName" || columns[0] == "name"
		return a.writeTable(ctx, tableOutputOptions{watch: o.watch, truncateProjects: firstIsName}, headers, rows, shortHeaders)
	}
	// Keep the single-column output headerless unless columns are selected explicitly.
	_, err := fmt.Fprintln(a.Out, delimited(o.columns, rows, o.format))
	return err
}
