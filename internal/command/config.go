package command

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
)

func (a App) showConfig(ctx context.Context, args []string) error {
	o, e := parseOpts(args, map[string]bool{"format": true, "watch": true})
	if e != nil {
		return e
	}
	if len(o.pos) > 0 {
		return usage("config does not accept positional arguments")
	}
	if err := a.validateTableWatch(o.watch, o.format); err != nil {
		return err
	}
	d, e := config.Load()
	if e != nil {
		return e
	}
	v := map[string]string{"WORKSPACE": d.Settings.Workspace}
	if d.Settings.Timezone != "" {
		v["TIMEZONE"] = d.Settings.Timezone
	}
	if o.format == "json" {
		visible := struct {
			Workspace string `json:"WORKSPACE"`
			Timezone  string `json:"TIMEZONE,omitempty"`
		}{d.Settings.Workspace, d.Settings.Timezone}
		b, _ := json.MarshalIndent(visible, "", "  ")
		fmt.Fprintln(a.Out, string(b))
		return nil
	}
	rows := [][]string{}
	for _, k := range []string{"WORKSPACE", "TIMEZONE"} {
		if x, ok := v[k]; ok {
			rows = append(rows, []string{k, spreadsheetText(o.format, x)})
		}
	}
	if o.format == "table" {
		return a.writeTable(ctx, tableOutputOptions{watch: o.watch}, []string{"Setting", "Value"}, rows, []string{"Key", "Val"})
	} else {
		fmt.Fprintln(a.Out, a.terminalDelimited([]string{"Setting", "Value"}, rows, o.format))
	}
	return nil
}

func location(settings model.Settings) (*time.Location, error) {
	if settings.Timezone == "" {
		return time.Local, nil
	}
	return time.LoadLocation(settings.Timezone)
}
