package toggl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/mukopikmin/toggl-cli/internal/config"
	"github.com/mukopikmin/toggl-cli/internal/model"
)

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
	got, err := (HTTPClient{Endpoint: s.URL}).Clients(context.Background(), config.Config{Workspace: "a/b", Token: "test-token"})
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
			got, err := (HTTPClient{Endpoint: s.URL}).Clients(context.Background(), config.Config{Workspace: "1"})
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
