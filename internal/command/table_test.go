package command

import (
	"strings"
	"testing"
)

func TestTableUnicodeAndMultiline(t *testing.T) {
	got := table([]string{"Name", "Value"}, [][]string{{"日本", "one\ntwo"}})
	want := "┌──────┬───────┐\n│ Name │ Value │\n├──────┼───────┤\n│ 日本 │ one   │\n│      │ two   │\n└──────┴───────┘"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if strings.Contains(got, "+") {
		t.Fatal(got)
	}
}
