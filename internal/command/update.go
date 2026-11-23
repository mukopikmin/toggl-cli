package command

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/mukopikmin/toggl-cli/internal/selfupdate"
)

type Updater interface {
	Check(ctx context.Context, current, channel string) (selfupdate.Plan, error)
	Install(ctx context.Context, plan selfupdate.Plan, out, errOut io.Writer) error
}

func (a App) update(ctx context.Context, args []string) error {
	o, err := parseOpts(args, map[string]bool{"channel": true})
	if err != nil {
		return err
	}
	if len(o.pos) > 0 {
		return usage("update does not accept positional arguments")
	}
	channel := o.channel
	if o.channelSet && channel != "stable" && channel != "nightly" {
		return usage("channel must be stable or nightly")
	}
	plan, err := a.Updater.Check(ctx, a.Version, channel)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "Current version: %s\nUpdate channel: %s\nAvailable version: %s\n", plan.CurrentVersion, plan.Channel, plan.TargetVersion)
	if !plan.Available {
		fmt.Fprintln(a.Out, "Already up to date; no update was installed.")
		return nil
	}
	fmt.Fprint(a.Out, "Download and install this update? [y/N] ")
	line, _ := bufio.NewReader(a.In).ReadString('\n')
	if strings.ToLower(strings.TrimSpace(line)) != "y" {
		fmt.Fprintln(a.Out, "Update cancelled.")
		return nil
	}
	if err = a.Updater.Install(ctx, plan, a.Out, a.Err); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "Installed version %s.\n", plan.TargetVersion)
	return nil
}
