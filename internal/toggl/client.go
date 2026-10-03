package toggl

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
	"net/http"
	"net/url"
	"time"
)

const Endpoint = "https://api.track.toggl.com/api/v9"

type Client interface {
	Projects(context.Context, config.Config) ([]model.Project, error)
	TimeEntries(context.Context, config.Config, time.Time, time.Time) ([]model.TimeEntry, error)
}
type HTTPClient struct {
	Client   *http.Client
	Endpoint string
}

func (c HTTPClient) do(ctx context.Context, cfg config.Config, path string, out any) error {
	ep := c.Endpoint
	if ep == "" {
		ep = Endpoint
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", ep+path, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(cfg.Token+":api_token")))
	res, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("Failed to fetch Toggl data: HTTP %d %s", res.StatusCode, res.Status)
	}
	return json.NewDecoder(res.Body).Decode(out)
}
func (c HTTPClient) http() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return http.DefaultClient
}
func (c HTTPClient) Projects(ctx context.Context, cfg config.Config) ([]model.Project, error) {
	var dto []struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		ProjectName   string `json:"project_name"`
		Active        *bool  `json:"active"`
		ProjectActive *bool  `json:"project_active"`
	}
	if err := c.do(ctx, cfg, "/workspaces/"+url.PathEscape(cfg.Workspace)+"/projects", &dto); err != nil {
		return nil, err
	}
	var out []model.Project
	for _, d := range dto {
		active := d.Active != nil && *d.Active
		if d.Active == nil && d.ProjectActive != nil {
			active = *d.ProjectActive
		}
		if !active {
			continue
		}
		name := d.Name
		if name == "" {
			name = d.ProjectName
		}
		p := model.Project{ID: d.ID, Name: name, DisplayName: name, Active: true}
		if s, ok := cfg.Projects[d.ID]; ok {
			if s.DisplayName != "" {
				p.DisplayName = s.DisplayName
			}
			p.Hidden = s.Hidden
			p.DisplayOrder = s.DisplayOrder
		}
		out = append(out, p)
	}
	return out, nil
}
func (c HTTPClient) TimeEntries(ctx context.Context, cfg config.Config, from, to time.Time) ([]model.TimeEntry, error) {
	loc := time.Local
	if cfg.Timezone != "" {
		var err error
		loc, err = time.LoadLocation(cfg.Timezone)
		if err != nil {
			return nil, err
		}
	}
	start := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, loc).UTC()
	end := time.Date(to.Year(), to.Month(), to.Day()+1, 0, 0, 0, 0, loc).UTC()
	q := url.Values{"start_date": {start.Format(time.RFC3339Nano)}, "end_date": {end.Format(time.RFC3339Nano)}, "meta": {"true"}}
	var dto []struct {
		ID          int64   `json:"id"`
		ProjectID   *int64  `json:"project_id"`
		PID         *int64  `json:"pid"`
		Start       string  `json:"start"`
		Stop        *string `json:"stop"`
		Duration    int64   `json:"duration"`
		Description string  `json:"description"`
	}
	if err := c.do(ctx, cfg, "/me/time_entries?"+q.Encode(), &dto); err != nil {
		return nil, err
	}
	out := make([]model.TimeEntry, len(dto))
	for i, d := range dto {
		pid := d.ProjectID
		if pid == nil {
			pid = d.PID
		}
		out[i] = model.TimeEntry{ID: d.ID, ProjectID: pid, Start: d.Start, Stop: d.Stop, DurationSeconds: d.Duration, Description: d.Description}
	}
	return out, nil
}
