package command

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParseOptsCompatibleForms(t *testing.T) {
	allowed := map[string]bool{"format": true, "separator": true, "days": true, "no-date": true}
	one := 1
	maxSafe := 9007199254740991
	tests := []struct {
		name string
		args []string
		want opts
	}{
		{"attached format", []string{"-fjson"}, opts{format: "json", sep: "\t"}},
		{"attached days", []string{"-d1", "-fjson"}, opts{format: "json", sep: "\t", days: &one}},
		{"attached separator", []string{"-s,"}, opts{format: "csv", sep: ",", separatorSet: true}},
		{"end of options", []string{"--", "--no-date", "-fjson"}, opts{format: "csv", sep: "\t", pos: []string{"--no-date", "-fjson"}}},
		{"trailing end marker", []string{"--"}, opts{format: "csv", sep: "\t"}},
		{"explicit dash value", []string{"--separator=--no-date"}, opts{format: "csv", sep: "--no-date", separatorSet: true}},
		{"single dash value", []string{"-s", "-"}, opts{format: "csv", sep: "-", separatorSet: true}},
		{"leading zero days", []string{"--days", "01"}, opts{format: "csv", sep: "\t", days: &one}},
		{"safe integer boundary", []string{"--days", "9007199254740991"}, opts{format: "csv", sep: "\t", days: &maxSafe}},
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
	allowed := map[string]bool{"format": true, "separator": true, "days": true, "no-project": true, "no-date": true, "clipboard": true}
	tests := [][]string{
		{"--no-date=false"}, {"--no-project=true"}, {"--clipboard=false"},
		{"--days", "+1"}, {"--days=-1"}, {"--days", "1.5"}, {"--days", ""},
		{"--days", "9007199254740992"}, {"--days", "999999999999999999999999"},
		{"--separator", "--no-date"}, {"--separator", "--"}, {"--separator", "-fjson"},
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
