package command

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

type projectField struct {
	name, header, shortHeader string
	value                     func(model.Project) any
}

var projectFields = []projectField{
	{"name", "Toggl name", "Name", func(p model.Project) any { return p.Name }},
	{"id", "ID", "ID", func(p model.Project) any { return p.ID }},
	{"alias", "Alias", "Alias", func(p model.Project) any {
		if p.Alias == "" {
			return nil
		}
		return p.Alias
	}},
	{"displayName", "Display name", "Display", func(p model.Project) any { return p.DisplayName }},
	{"hidden", "Hidden", "Hidden", func(p model.Project) any { return p.Hidden }},
	{"active", "Active", "Active", func(p model.Project) any { return p.Active }},
	{"displayOrder", "Display order", "Order", func(p model.Project) any {
		if p.DisplayOrder == nil {
			return nil
		}
		return *p.DisplayOrder
	}},
}

func projectFieldNames(fields []projectField) []string {
	names := make([]string, len(fields))
	for i, field := range fields {
		names[i] = field.name
	}
	return names
}

type projectListOptions struct {
	opts
	selectedFields []projectField
}

func (a App) parseProjectListOptions(args []string) (projectListOptions, error) {
	o, err := parseOpts(args, map[string]bool{
		"format": true, "watch": true, "fields": true, "include-hidden": true,
	})
	if err != nil {
		return projectListOptions{}, err
	}
	if len(o.pos) > 0 {
		return projectListOptions{}, usage("project list does not accept positional arguments")
	}
	if err := a.validateTableWatch(o.watch, o.format); err != nil {
		return projectListOptions{}, err
	}
	result := projectListOptions{opts: o, selectedFields: projectFields}
	if o.fields == nil {
		return result, nil
	}
	result.selectedFields = make([]projectField, 0, len(o.fields))
	seen := make(map[string]bool, len(o.fields))
	for _, name := range o.fields {
		name = strings.TrimSpace(name)
		if name == "" {
			return projectListOptions{}, usage("--fields requires a comma-separated list of field names")
		}
		i := slices.IndexFunc(projectFields, func(field projectField) bool { return field.name == name })
		if i < 0 {
			return projectListOptions{}, usagef("unknown project field %q (expected one of %s)", name, strings.Join(projectFieldNames(projectFields), ", "))
		}
		if seen[name] {
			return projectListOptions{}, usagef("duplicate project field %q", name)
		}
		seen[name] = true
		result.selectedFields = append(result.selectedFields, projectFields[i])
	}
	return result, nil
}

func (a App) writeProjectList(ctx context.Context, projects []model.Project, o projectListOptions) error {
	projects = model.SortedProjects(projects, o.includeHidden)
	fields := o.selectedFields
	if o.format == "json" {
		objects := make([]map[string]any, len(projects))
		for i, p := range projects {
			object := make(map[string]any, len(fields))
			for _, field := range fields {
				object[field.name] = field.value(p)
			}
			objects[i] = object
		}
		b, err := json.MarshalIndent(objects, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(a.Out, string(b))
		return err
	}

	rows := make([][]string, len(projects))
	for i, p := range projects {
		row := make([]string, len(fields))
		for j, field := range fields {
			if value := field.value(p); value != nil {
				row[j] = fmt.Sprint(value)
			}
		}
		rows[i] = row
	}
	if o.format == "table" {
		headers := make([]string, len(fields))
		shortHeaders := make([]string, len(fields))
		for i, field := range fields {
			headers[i], shortHeaders[i] = field.header, field.shortHeader
		}
		first := fields[0].name
		firstIsName := first == "name" || first == "alias" || first == "displayName"
		return a.writeTable(ctx, tableOutputOptions{watch: o.watch, truncateProjects: firstIsName}, headers, rows, shortHeaders)
	}
	_, err := fmt.Fprintln(a.Out, renderDelimited(projectFieldNames(fields), rows, o.format, true))
	return err
}
