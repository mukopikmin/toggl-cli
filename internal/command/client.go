package command

import (
	"context"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
)

var clientFields = listFields[model.Client]{
	{"id", "ID", "ID", func(c model.Client) any { return c.ID }},
	{"name", "Client", "Clnt", func(c model.Client) any { return c.Name }},
}

func (a App) client(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usage("client requires a subcommand: list")
	}
	if args[0] != "list" {
		return usagef("unknown client subcommand: %s", args[0])
	}
	o, err := parseOpts(args[1:], map[string]bool{"format": true, "watch": true, "fields": true})
	if err != nil {
		return err
	}
	if len(o.pos) > 0 {
		return usage("client list does not accept positional arguments")
	}
	if err := a.validateTableWatch(o.watch, o.format); err != nil {
		return err
	}
	fields, err := clientFields.selectFields("client", o.fields)
	if err != nil {
		return err
	}
	d, err := config.Load()
	if err != nil {
		return err
	}
	clients, err := a.NewClient(d.Settings.Workspace, d.Settings.Token).Clients(ctx)
	if err != nil {
		return err
	}
	return writeList(ctx, a, clients, fields, o.format, tableOutputOptions{watch: o.watch})
}
