package command

import (
	"encoding/csv"
	"strings"
	"unicode"
	"unicode/utf8"
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

func spreadsheetText(format, value string) string {
	if format != "csv" && format != "tsv" {
		return value
	}
	first, _ := utf8.DecodeRuneInString(strings.TrimLeftFunc(value, unicode.IsSpace))
	if strings.ContainsRune("=+-@＝＋－＠", first) {
		return "'" + value
	}
	return value
}
