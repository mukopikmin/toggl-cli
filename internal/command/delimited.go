package command

import (
	"encoding/csv"
	"strings"
)

func delimited(headers []string, rows [][]string, format string) string {
	var text strings.Builder
	w := csv.NewWriter(&text)
	if format == "tsv" {
		w.Comma = '\t'
	}
	// The delimiter is fixed and strings.Builder writes cannot fail.
	if headers != nil {
		_ = w.Write(headers)
	}
	_ = w.WriteAll(rows)
	return strings.TrimSuffix(text.String(), "\n")
}
