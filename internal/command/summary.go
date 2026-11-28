package command

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
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
		if err := config.ValidateTemplateName(o.template); err != nil {
			return usage(err.Error())
		}
		d, err := config.LoadTemplates()
		if err != nil {
			return err
		}
		saved, exists := d.Templates[o.template]
		if !exists {
			return fmt.Errorf("unknown template: %s", o.template)
		}
		base = saved.Options
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
		settings.Period = model.SummaryPeriod{
			StartDate: monthStart.Format(time.DateOnly),
			EndDate:   monthStart.AddDate(0, 1, -1).Format(time.DateOnly),
		}
	}
	if o.dailyTotal && settings.Output.Format == "json" {
		return usage("--daily-total cannot be used with json format")
	}
	if e = settings.Validate(true); e != nil {
		return summaryUsage(e)
	}
	if err := a.validateTableWatch(o.watch, settings.Output.Format); err != nil {
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
	from, to, e := settings.Period.DateRange(now, loc)
	if e != nil {
		return summaryUsage(e)
	}
	client := a.NewClient(d.Settings.Workspace, d.Settings.Token)
	sum, e := usecase.LoadSummary(ctx, client, from, to, loc, now, d.Settings.Projects)
	if e != nil {
		return e
	}
	var projects []model.Project
	if settings.Output.Format != "json" {
		projects, e = usecase.LoadProjects(ctx, client, d.Settings.Projects, false)
		if e != nil {
			return e
		}
	}
	return a.writeSummary(ctx, settings.Output, from, to, sum, projects, o.watch, o.dailyTotal)
}
