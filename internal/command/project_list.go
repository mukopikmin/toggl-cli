package command

import (
	"context"
	"github.com/mukopikmin/toggl-cli/internal/model"
)

var projectFields = listFields[model.Project]{
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
		return optionalListValue(p.DisplayOrder)
	}},
}

type projectListOptions struct {
	opts
	selectedFields listFields[model.Project]
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
	fields, err := projectFields.selectFields("project", o.fields)
	if err != nil {
		return projectListOptions{}, err
	}
	return projectListOptions{opts: o, selectedFields: fields}, nil
}

func (a App) writeProjectList(ctx context.Context, projects []model.Project, o projectListOptions) error {
	projects = model.SortedProjects(projects, o.includeHidden)
	first := o.selectedFields[0].name
	options := tableOutputOptions{watch: o.watch, truncateProjects: first == "name" || first == "alias" || first == "displayName"}
	return writeList(ctx, a, projects, o.selectedFields, o.format, options)
}
