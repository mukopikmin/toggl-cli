package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

func (a App) client(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usage("client requires a subcommand: list")
	}
	if args[0] != "list" {
		return usagef("unknown client subcommand: %s", args[0])
	}
	o, err := parseOpts(args[1:], map[string]bool{"format": true})
	if err != nil {
		return err
	}
	if len(o.pos) > 0 {
		return usage("client list does not accept positional arguments")
	}
	d, err := loadCfg()
	if err != nil {
		return err
	}
	clients, err := a.Client.Clients(ctx, d.Config)
	if err != nil {
		return err
	}
	switch o.format {
	case "json":
		if clients == nil {
			clients = []model.Client{}
		}
		b, err := json.MarshalIndent(clients, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(a.Out, string(b))
		return err
	case "table":
		rows := make([][]string, len(clients))
		for i, c := range clients {
			rows[i] = []string{c.Name}
		}
		_, err = fmt.Fprintln(a.Out, table([]string{"Client"}, rows))
		return err
	default:
		names := make([]string, len(clients))
		for i, c := range clients {
			names[i] = c.Name
		}
		_, err = fmt.Fprintln(a.Out, strings.Join(names, "\n"))
		return err
	}
}
