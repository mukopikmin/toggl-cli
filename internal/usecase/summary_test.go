package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mukopikmin/toggl-cli/internal/model"
)

type summaryReader struct {
	start, end time.Time
	entries    []model.TimeEntry
	err        error
}

func (r *summaryReader) TimeEntries(_ context.Context, start, end time.Time) ([]model.TimeEntry, error) {
	r.start, r.end = start, end
	return r.entries, r.err
}

func TestLoadSummaryUsesCalendarBoundsAndProjectSettings(t *testing.T) {
	tokyo := time.FixedZone("Tokyo", 9*60*60)
	from := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 1)
	start := from.Add(-9 * time.Hour)
	visible, hidden := int64(1), int64(2)
	reader := &summaryReader{entries: []model.TimeEntry{
		{ProjectID: &hidden, Start: start, DurationSeconds: 60},
		{ProjectID: &visible, Start: start, DurationSeconds: 90},
	}}
	settings := map[int64]model.ProjectSettings{hidden: {Hidden: true}}
	got, err := LoadSummary(context.Background(), reader, from, to, tokyo, to, settings)
	if err != nil || len(got["2026-05-01"]) != 1 || got["2026-05-01"][visible] != 1.5 {
		t.Fatalf("unexpected summary %v, error %v", got, err)
	}
	if !reader.start.Equal(start) || !reader.end.Equal(start.Add(48*time.Hour)) {
		t.Fatalf("requested %s through %s", reader.start, reader.end)
	}
	if reader.entries[0].ProjectID != &hidden || reader.entries[1].ProjectID != &visible {
		t.Fatal("summary changed the reader's entries")
	}
}

func TestLoadSummaryReturnsReaderError(t *testing.T) {
	want := errors.New("entries unavailable")
	reader := &summaryReader{err: want}
	date := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	got, err := LoadSummary(context.Background(), reader, date, date, time.UTC, date, nil)
	if got != nil || !errors.Is(err, want) {
		t.Fatalf("got %v, %v; want the reader error", got, err)
	}
}
