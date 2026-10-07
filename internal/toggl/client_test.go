package toggl

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/config"
)

func TestTimeEntriesDateRange(t *testing.T) {
	tests := []struct{ name, zone, from, to, wantStart, wantEnd string }{{"UTC month start", "UTC", "2026-05-01", "2026-05-02", "2026-05-01T00:00:00Z", "2026-05-03T00:00:00Z"}, {"Tokyo UTC difference", "Asia/Tokyo", "2026-05-01", "2026-05-02", "2026-04-30T15:00:00Z", "2026-05-02T15:00:00Z"}, {"New York month end", "America/New_York", "2026-01-31", "2026-01-31", "2026-01-31T05:00:00Z", "2026-02-01T05:00:00Z"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var q url.Values
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { q = r.URL.Query(); w.Write([]byte("[]")) }))
			defer s.Close()
			c := HTTPClient{Endpoint: s.URL}
			from, _ := time.Parse(time.DateOnly, tt.from)
			to, _ := time.Parse(time.DateOnly, tt.to)
			_, err := c.TimeEntries(context.Background(), config.Config{Timezone: tt.zone}, from, to)
			if err != nil {
				t.Fatal(err)
			}
			if q.Get("start_date") != tt.wantStart || q.Get("end_date") != tt.wantEnd {
				t.Fatalf("%v", q)
			}
		})
	}
}
func TestProjectsMapsDTOSettingsAndAuthorization(t *testing.T) {
	var auth, path string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, path = r.Header.Get("Authorization"), r.URL.Path
		w.Write([]byte(`[{"id":1,"project_name":"Old","project_active":true},{"id":2,"name":"New","active":false}]`))
	}))
	defer s.Close()
	order := float64(4)
	client := HTTPClient{Endpoint: s.URL}
	projects, err := client.Projects(context.Background(), config.Config{Workspace: "a/b", Token: "secret", Projects: map[int64]config.Project{1: {DisplayName: "Custom", Hidden: true, DisplayOrder: &order}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].DisplayName != "Custom" || !projects[0].Hidden || *projects[0].DisplayOrder != 4 {
		t.Fatalf("%+v", projects)
	}
	if auth != "Basic "+base64.StdEncoding.EncodeToString([]byte("secret:api_token")) {
		t.Fatal(auth)
	}
	if !strings.Contains(path, "/workspaces/a/b/projects") {
		t.Fatal(path)
	}
}
func TestTimeEntriesMapsFallbackPIDAndErrors(t *testing.T) {
	pid := int64(9)
	_ = pid
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":1,"pid":9,"start":"2026-01-01T00:00:00Z","stop":null,"duration":60,"description":"work"}]`))
	}))
	defer s.Close()
	client := HTTPClient{Endpoint: s.URL}
	day, _ := time.Parse(time.DateOnly, "2026-01-01")
	entries, err := client.TimeEntries(context.Background(), config.Config{Timezone: "UTC"}, day, day)
	if err != nil || len(entries) != 1 || entries[0].ProjectID == nil || *entries[0].ProjectID != 9 {
		t.Fatalf("%+v %v", entries, err)
	}
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", http.StatusUnauthorized) }))
	defer bad.Close()
	if _, err = (HTTPClient{Endpoint: bad.URL}).Projects(context.Background(), config.Config{}); err == nil {
		t.Fatal("expected API error")
	}
}
