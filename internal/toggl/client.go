package toggl

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/mukopikmin/toggl-cli/internal/model"
	"net/http"
	"net/url"
	"time"
)

const Endpoint = "https://api.track.toggl.com/api/v9"

type HTTPClient struct {
	Client           *http.Client
	Endpoint         string
	Workspace, Token string
}

func (c HTTPClient) do(ctx context.Context, operation, path string, out any) error {
	ep := c.Endpoint
	if ep == "" {
		ep = Endpoint
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", ep+path, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.Token+":api_token")))
	res, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("Failed to %s: HTTP %s", operation, res.Status)
	}
	return json.NewDecoder(res.Body).Decode(out)
}
func (c HTTPClient) http() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return http.DefaultClient
}
func (c HTTPClient) Clients(ctx context.Context) ([]model.Client, error) {
	var dto []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	if err := c.do(ctx, "fetch clients", "/workspaces/"+url.PathEscape(c.Workspace)+"/clients", &dto); err != nil {
		return nil, err
	}
	out := make([]model.Client, len(dto))
	for i, d := range dto {
		out[i] = model.Client{ID: d.ID, Name: d.Name}
	}
	return out, nil
}

func (c HTTPClient) Projects(ctx context.Context) ([]model.Project, error) {
	var dto []struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		ProjectName   string `json:"project_name"`
		Active        *bool  `json:"active"`
		ProjectActive *bool  `json:"project_active"`
	}
	if err := c.do(ctx, "fetch projects", "/workspaces/"+url.PathEscape(c.Workspace)+"/projects", &dto); err != nil {
		return nil, err
	}
	var out []model.Project
	for _, d := range dto {
		active := d.Active != nil && *d.Active
		if d.Active == nil && d.ProjectActive != nil {
			active = *d.ProjectActive
		}
		name := d.Name
		if name == "" {
			name = d.ProjectName
		}
		out = append(out, model.Project{ID: d.ID, Name: name, Active: active})
	}
	return out, nil
}
func (c HTTPClient) TimeEntries(ctx context.Context, from, end time.Time) ([]model.TimeEntry, error) {
	q := url.Values{"start_date": {from.UTC().Format(time.RFC3339Nano)}, "end_date": {end.UTC().Format(time.RFC3339Nano)}, "meta": {"true"}}
	var dto []struct {
		ID          int64   `json:"id"`
		ProjectID   *int64  `json:"project_id"`
		PID         *int64  `json:"pid"`
		Start       string  `json:"start"`
		Stop        *string `json:"stop"`
		Duration    int64   `json:"duration"`
		Description string  `json:"description"`
	}
	if err := c.do(ctx, "fetch time entries", "/me/time_entries?"+q.Encode(), &dto); err != nil {
		return nil, err
	}
	out := make([]model.TimeEntry, len(dto))
	for i, d := range dto {
		pid := d.ProjectID
		if pid == nil {
			pid = d.PID
		}
		start, err := time.Parse(time.RFC3339Nano, d.Start)
		if err != nil {
			return nil, fmt.Errorf("invalid start for time entry %d: %w", d.ID, err)
		}
		var stop *time.Time
		if d.Stop != nil {
			parsed, err := time.Parse(time.RFC3339Nano, *d.Stop)
			if err != nil {
				return nil, fmt.Errorf("invalid stop for time entry %d: %w", d.ID, err)
			}
			stop = &parsed
		}
		out[i] = model.TimeEntry{
			ID: d.ID, ProjectID: pid, Start: start, Stop: stop, Description: d.Description,
			DurationSeconds: max(0, d.Duration), Running: d.Duration < 0,
		}
	}
	return out, nil
}
