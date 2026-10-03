package toggl

import (
	"context"
	"github.com/mukopikmin/toggl-cli/internal/config"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestTimeEntriesDateRange(t *testing.T) {
	tests := []struct{ name, zone, wantStart, wantEnd string }{{"UTC month start", "UTC", "2026-05-01T00:00:00Z", "2026-05-03T00:00:00Z"}, {"Tokyo UTC difference", "Asia/Tokyo", "2026-04-30T15:00:00Z", "2026-05-02T15:00:00Z"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var q url.Values
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { q = r.URL.Query(); w.Write([]byte("[]")) }))
			defer s.Close()
			c := HTTPClient{Endpoint: s.URL}
			from, _ := time.Parse(time.DateOnly, "2026-05-01")
			to, _ := time.Parse(time.DateOnly, "2026-05-02")
			_, e := c.TimeEntries(context.Background(), config.Config{Timezone: tt.zone}, from, to)
			if e != nil {
				t.Fatal(e)
			}
			if q.Get("start_date") != tt.wantStart || q.Get("end_date") != tt.wantEnd {
				t.Fatalf("%v", q)
			}
		})
	}
}
