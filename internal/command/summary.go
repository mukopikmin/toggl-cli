package command

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
	"github.com/mukopikmin/toggl-cli/internal/template"
	"github.com/mukopikmin/toggl-cli/internal/usecase"
)

func parseMonth(v string) (time.Time, error) {
	if !regexp.MustCompile(`^\d{4}[-/]?\d{2}$`).MatchString(v) {
		return time.Time{}, usage("month must use YYYY-MM, YYYYMM, or YYYY/MM")
	}
	v = strings.ReplaceAll(v, "/", "-")
	if len(v) == 6 {
		v = v[:4] + "-" + v[4:]
	}
	t, err := time.Parse("2006-01", v)
	if err != nil {
		return time.Time{}, usage("month must be a valid month")
	}
	return t, nil
}
func (a App) summary(ctx context.Context, args []string) error {
	allowed := summaryFlags()
	allowed["template"] = true
	allowed["month"] = true
	allowed["watch"] = true
	allowed["daily-total"] = true
	o, e := parseOpts(args, allowed)
	if e != nil {
		return e
	}
	if o.monthSet {
		if o.days != nil || len(o.pos) > 0 {
			return usage("summary accepts only one of start and end date, --days, or --month")
		}
	}
	base := model.SummaryOptions{}
	if o.templateSet {
		if err := template.ValidateName(o.template); err != nil {
			return usage(err.Error())
		}
		d, err := template.Load()
		if err != nil {
			return err
		}
		var exists bool
		base, exists = d.Templates[o.template]
		if !exists {
			return fmt.Errorf("unknown template: %s", o.template)
		}
	}
	settings, e := summaryOptions(base, o)
	if e != nil {
		return e
	}
	if o.monthSet {
		monthStart, err := parseMonth(o.month)
		if err != nil {
			return err
		}
		settings.Days = nil
		settings.StartDate = monthStart.Format(time.DateOnly)
		settings.EndDate = monthStart.AddDate(0, 1, -1).Format(time.DateOnly)
	}
	if o.dailyTotal && settings.Format == "json" {
		return usage("--daily-total cannot be used with json format")
	}
	if e = settings.Validate(true); e != nil {
		return usage(e.Error())
	}
	if err := a.validateTableWatch(o.watch, settings.Format); err != nil {
		return err
	}
	d, e := config.Load()
	if e != nil {
		return e
	}
	loc, e := location(d.Settings)
	if e != nil {
		return e
	}
	now := a.Now()
	from, to, e := settings.DateRange(now, loc)
	if e != nil {
		return usage(e.Error())
	}
	client := a.NewClient(d.Settings.Workspace, d.Settings.Token)
	sum, e := usecase.LoadSummary(ctx, client, from, to, loc, now, d.Settings.Projects)
	if e != nil {
		return e
	}
	var projects []model.Project
	if settings.Format != "json" {
		projects, e = usecase.LoadProjects(ctx, client, d.Settings.Projects)
		if e != nil {
			return e
		}
	}
	return a.writeSummary(ctx, settings, from, to, sum, projects, o.watch, o.dailyTotal)
}
