package command

import (
	"fmt"

	"github.com/mukopikmin/toggl-cli/internal/model"
	"github.com/mukopikmin/toggl-cli/internal/template"
)

func summaryFlags() map[string]bool {
	return map[string]bool{
		"format": true, "separator": true, "days": true,
		"no-project": true, "project": true, "no-date": true, "date": true,
		"clipboard": true, "no-clipboard": true,
	}
}

// summaryOptions merges only explicit CLI settings; a period is replaced as a unit.
func summaryOptions(base model.SummaryOptions, o opts) (model.SummaryOptions, error) {
	if o.days != nil && len(o.pos) > 0 {
		return base, usage("summary accepts either start and end date or --days, not both")
	}
	if len(o.pos) != 0 && len(o.pos) != 2 {
		return base, usage("summary requires start and end date or --days")
	}
	if o.days != nil {
		base.Days, base.StartDate, base.EndDate = o.days, "", ""
	} else if len(o.pos) == 2 {
		base.Days, base.StartDate, base.EndDate = nil, o.pos[0], o.pos[1]
		// Empty CLI dates are invalid, rather than an output-only template.
		if base.StartDate == "" || base.EndDate == "" {
			return base, usage("start and end date must use YYYY-MM-DD")
		}
	}
	if o.formatSet {
		base.Format = o.format
	}
	if o.separatorSet {
		base.Separator = &o.sep
	}
	if o.noProjectSet {
		base.NoProject = o.noProject
	}
	if o.noDateSet {
		base.NoDate = o.noDate
	}
	if o.clipboardSet {
		base.Clipboard = o.clipboard
	}
	if base.Format == "" {
		base.Format = "csv"
	}
	return base, nil
}

func (a App) template(args []string) error {
	if len(args) == 0 {
		return usage("template requires a subcommand: add, list, show, or delete")
	}
	var name string
	var settings model.SummaryOptions
	var force bool
	switch args[0] {
	case "add":
		allowed := summaryFlags()
		allowed["force"] = true
		o, err := parseOpts(args[1:], allowed)
		if err != nil {
			return err
		}
		if len(o.pos) == 0 {
			return usage("template add requires a name")
		}
		name, o.pos = o.pos[0], o.pos[1:]
		force = o.force
		settings, err = summaryOptions(model.SummaryOptions{}, o)
		if err != nil {
			return err
		}
		if err = settings.Validate(false); err != nil {
			return usage(err.Error())
		}
	case "list":
		if len(args) != 1 {
			return usage("template list does not accept arguments")
		}
	case "show", "delete":
		o, err := parseOpts(args[1:], nil)
		if err != nil {
			return err
		}
		if len(o.pos) != 1 {
			return usagef("template %s requires a name", args[0])
		}
		name = o.pos[0]
	default:
		return usagef("unknown template subcommand: %s", args[0])
	}
	if args[0] != "list" {
		if err := template.ValidateName(name); err != nil {
			return usage(err.Error())
		}
	}
	d, err := template.Load()
	if err != nil {
		return err
	}
	existing, exists := d.Templates[name]
	switch args[0] {
	case "add":
		if exists && !force {
			return fmt.Errorf("template %q already exists; use template add --force to replace it", name)
		}
		d.Templates[name] = settings
		if err := d.Write(); err != nil {
			return err
		}
		fmt.Fprintf(a.Out, "Added template %q\n", name)
	case "list":
		for _, name := range d.Names() {
			fmt.Fprintln(a.Out, name)
		}
	case "show":
		if !exists {
			return fmt.Errorf("unknown template: %s", name)
		}
		text, err := (template.Document{Templates: map[string]model.SummaryOptions{name: existing}}).Encode()
		if err != nil {
			return err
		}
		fmt.Fprint(a.Out, text)
	case "delete":
		if !exists {
			return fmt.Errorf("unknown template: %s", name)
		}
		delete(d.Templates, name)
		if err := d.Write(); err != nil {
			return err
		}
		fmt.Fprintf(a.Out, "Deleted template %q\n", name)
	}
	return nil
}
