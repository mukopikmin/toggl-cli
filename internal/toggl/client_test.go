package toggl

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

func TestTimeEntriesDateRange(t *testing.T) {
	tests := []struct{ name, zone, from, to, wantStart, wantEnd string }{
		{"UTC month start", "UTC", "2026-05-01", "2026-05-02", "2026-05-01T00:00:00Z", "2026-05-03T00:00:00Z"},
		{"Tokyo UTC difference", "Asia/Tokyo", "2026-05-01", "2026-05-02", "2026-04-30T15:00:00Z", "2026-05-02T15:00:00Z"},
		{"New York month end", "America/New_York", "2026-01-31", "2026-01-31", "2026-01-31T05:00:00Z", "2026-02-01T05:00:00Z"},
		{"Tokyo full month", "Asia/Tokyo", "2026-05-01", "2026-05-31", "2026-04-30T15:00:00Z", "2026-05-31T15:00:00Z"},
		{"New York DST month", "America/New_York", "2026-03-01", "2026-03-31", "2026-03-01T05:00:00Z", "2026-04-01T04:00:00Z"},
		{"leap February", "UTC", "2024-02-01", "2024-02-29", "2024-02-01T00:00:00Z", "2024-03-01T00:00:00Z"},
		{"year end", "UTC", "2026-12-01", "2026-12-31", "2026-12-01T00:00:00Z", "2027-01-01T00:00:00Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var q url.Values
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { q = r.URL.Query(); w.Write([]byte("[]")) }))
			defer s.Close()
			c := HTTPClient{Endpoint: s.URL}
			loc, err := time.LoadLocation(tt.zone)
			if err != nil {
				t.Fatal(err)
			}
			from, _ := time.ParseInLocation(time.DateOnly, tt.from, loc)
			to, _ := time.ParseInLocation(time.DateOnly, tt.to, loc)
			_, err = c.TimeEntries(context.Background(), from, to.AddDate(0, 0, 1))
			if err != nil {
				t.Fatal(err)
			}
			if q.Get("start_date") != tt.wantStart || q.Get("end_date") != tt.wantEnd {
				t.Fatalf("%v", q)
			}
		})
	}
}

func TestProjectsMapsDTOAndAuthorization(t *testing.T) {
	var auth, path string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, path = r.Header.Get("Authorization"), r.URL.Path
		w.Write([]byte(`[{"id":1,"project_name":"Old","project_active":true},{"id":2,"name":"New","active":false}]`))
	}))
	defer s.Close()
	client := HTTPClient{Endpoint: s.URL, Workspace: "a/b", Token: "secret"}
	projects, err := client.Projects(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 2 || projects[0].Name != "Old" || !projects[0].Active || projects[1].Name != "New" || projects[1].Active {
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
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":1,"pid":9,"start":"2026-01-01T00:00:00Z","stop":null,"duration":-1767225600,"description":"work"}]`))
	}))
	defer s.Close()
	client := HTTPClient{Endpoint: s.URL}
	day, _ := time.Parse(time.DateOnly, "2026-01-01")
	entries, err := client.TimeEntries(context.Background(), day, day.AddDate(0, 0, 1))
	if err != nil || len(entries) != 1 || entries[0].ProjectID == nil || *entries[0].ProjectID != 9 {
		t.Fatalf("%+v %v", entries, err)
	}
	if entries[0].Stop != nil || !entries[0].Running || entries[0].ElapsedSeconds(day.Add(time.Minute)) != 60 {
		t.Fatalf("unexpected running entry: %+v", entries[0])
	}
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", http.StatusUnauthorized) }))
	defer bad.Close()
	if _, err = (HTTPClient{Endpoint: bad.URL}).Projects(context.Background()); err == nil {
		t.Fatal("expected API error")
	}
}

func TestClientsMapsDTOAndRequest(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/workspaces/a%2Fb/clients" || r.URL.RawQuery != "" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "test-token" || pass != "api_token" {
			t.Error("incorrect API authentication")
		}
		w.Write([]byte(`[{"id":2,"name":"取引先","wid":7,"archived":false},{"id":1,"name":"Other","wid":7,"archived":true}]`))
	}))
	defer s.Close()
	got, err := (HTTPClient{Endpoint: s.URL, Workspace: "a/b", Token: "test-token"}).Clients(context.Background())
	want := []model.Client{{ID: 2, Name: "取引先"}, {ID: 1, Name: "Other"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, %v; want %+v", got, err, want)
	}
}

func TestClientsEmptyAndErrors(t *testing.T) {
	tests := []struct {
		name, body string
		status     int
		wantError  string
	}{
		{"empty", `[]`, http.StatusOK, ""},
		{"null", `null`, http.StatusOK, ""},
		{"unauthorized", `unauthorized`, http.StatusUnauthorized, "Failed to fetch clients: HTTP 401"},
		{"server error", `unavailable`, http.StatusInternalServerError, "Failed to fetch clients: HTTP 500"},
		{"malformed JSON", `[{`, http.StatusOK, "unexpected EOF"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer s.Close()
			got, err := (HTTPClient{Endpoint: s.URL, Workspace: "1"}).Clients(context.Background())
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("expected %q, got %v", tt.wantError, err)
				}
			} else if err != nil || got == nil || len(got) != 0 {
				t.Fatalf("expected empty client slice, got %+v, %v", got, err)
			}
		})
	}
}
