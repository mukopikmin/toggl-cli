package command

import (
	"context"
	"fmt"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/usecase"
)

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
	d, e := config.Load()
	if e != nil {
		return e
	}
	client := a.NewClient(d.Settings.Workspace, d.Settings.Token)
	if args[0] == "sync" {
		count, err := usecase.SyncProjects(ctx, client, d, d.Settings.Projects)
		if err != nil {
			return err
		}
		if count == 0 {
			fmt.Fprintln(a.Out, "All active projects are already configured")
			return nil
		}
		fmt.Fprintf(a.Out, "Added %d project(s) to the config file\n", count)
		return nil
	}
	projects, err := usecase.LoadProjects(ctx, client, d.Settings.Projects, listOptions.includeHidden)
	if err != nil {
		return err
	}
	if args[0] == "list" {
		return a.writeProjectList(ctx, projects, listOptions)
	}
	return a.reorderProjects(d, projects)
}
