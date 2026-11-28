package model

import (
	"testing"
	"time"
)

func TestElapsedSecondsClampsRunningEntriesAfterReportTime(t *testing.T) {
	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	entry := TimeEntry{Start: now.Add(10 * time.Second), Running: true}
	if got := entry.ElapsedSeconds(now); got != 0 {
		t.Fatalf("got %d seconds, want zero", got)
	}
}
