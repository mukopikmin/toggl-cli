package usecase

import (
	"testing"
	"time"
)

func TestStartOfDate(t *testing.T) {
	for _, tt := range []struct{ zone, date, want string }{
		{"America/Sao_Paulo", "2018-11-04", "2018-11-04T01:00:00-02:00"},
		{"America/Sao_Paulo", "2018-11-05", "2018-11-05T00:00:00-02:00"},
		{"America/Havana", "2024-11-03", "2024-11-03T00:00:00-04:00"},
		{"Pacific/Apia", "2011-12-30", "2011-12-31T00:00:00+14:00"},
		{"America/New_York", "2040-12-31", "2040-12-31T00:00:00-05:00"},
	} {
		t.Run(tt.zone+"/"+tt.date, func(t *testing.T) {
			loc, err := time.LoadLocation(tt.zone)
			if err != nil {
				t.Fatal(err)
			}
			date, err := time.Parse(time.DateOnly, tt.date)
			if err != nil {
				t.Fatal(err)
			}
			if got := startOfDate(date, loc).Format(time.RFC3339); got != tt.want {
				t.Fatalf("start = %s, want %s", got, tt.want)
			}
		})
	}
}
