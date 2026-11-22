package command

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParseOptsCompatibleForms(t *testing.T) {
	allowed := map[string]bool{"format": true, "days": true, "no-date": true, "daily-total": true}
	one := 1
	maxSafe := 9007199254740991
	tests := []struct {
		name string
		args []string
		want opts
	}{
		{"default format", nil, opts{format: "tsv"}},
		{"attached format", []string{"-fjson"}, opts{format: "json", formatSet: true}},
		{"daily total", []string{"--daily-total"}, opts{format: "tsv", dailyTotal: true}},
		{"attached tsv", []string{"-ftsv"}, opts{format: "tsv", formatSet: true}},
		{"attached days", []string{"-d1", "-fjson"}, opts{format: "json", formatSet: true, days: &one}},
		{"end of options", []string{"--", "--no-date", "-fjson"}, opts{format: "tsv", pos: []string{"--no-date", "-fjson"}}},
		{"trailing end marker", []string{"--"}, opts{format: "tsv"}},
		{"leading zero days", []string{"--days", "01"}, opts{format: "tsv", days: &one}},
		{"safe integer boundary", []string{"--days", "9007199254740991"}, opts{format: "tsv", days: &maxSafe}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOpts(tt.args, allowed)
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %+v, %v; want %+v", got, err, tt.want)
			}
		})
	}
}

func TestParseOptsRejectsInvalidValues(t *testing.T) {
	allowed := map[string]bool{"format": true, "days": true, "no-project": true, "no-date": true, "clipboard": true}
	tests := [][]string{
		{"--no-date=false"}, {"--no-project=true"}, {"--clipboard=false"},
		{"--days", "0"}, {"--days=00"}, {"-d0"},
		{"--days", "+1"}, {"--days=-1"}, {"--days", "1.5"}, {"--days", ""},
		{"--days", "9007199254740992"}, {"--days", "999999999999999999999999"},
		{"--separator", "--no-date"}, {"-s,"},
		{"--format"}, {"--format", "--no-date"}, {"--days"}, {"--unknown=value"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, err := parseOpts(args, allowed)
			var usageError UsageError
			if !errors.As(err, &usageError) {
				t.Fatalf("expected usage error for %v, got %v", args, err)
			}
		})
	}
}

func TestSummaryBooleanOverrides(t *testing.T) {
	for _, tt := range []struct {
		args []string
		want bool
	}{
		{[]string{"--project", "--no-project", "--date", "--no-date", "--no-clipboard", "--clipboard"}, true},
		{[]string{"--no-project", "--project", "--no-date", "--date", "--clipboard", "--no-clipboard"}, false},
	} {
		o, err := parseOpts(tt.args, summaryFlags())
		if err != nil || o.noProject != tt.want || o.noDate != tt.want || o.clipboard != tt.want || !o.noProjectSet || !o.noDateSet || !o.clipboardSet {
			t.Fatalf("parsed %v as %+v, %v", tt.args, o, err)
		}
	}
	for _, arg := range []string{"--project=false", "--date=true", "--no-clipboard=false"} {
		if _, err := parseOpts([]string{arg}, summaryFlags()); err == nil {
			t.Fatalf("accepted boolean value: %s", arg)
		}
	}
}
