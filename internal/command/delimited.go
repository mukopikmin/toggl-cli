package command

import (
	"encoding/csv"
	"strings"
	"unicode"
)

func delimited(headers []string, rows [][]string, format string, textColumns ...int) string {
	var text strings.Builder
	w := csv.NewWriter(&text)
	if format == "tsv" {
		w.Comma = '\t'
	}
	// The delimiter is fixed and strings.Builder writes cannot fail.
	if headers != nil {
		_ = w.Write(headers)
	}
	for _, row := range rows {
		record := append([]string(nil), row...)
		for _, column := range textColumns {
			record[column] = spreadsheetText(record[column])
		}
		_ = w.Write(record)
	}
	w.Flush()
	return strings.TrimSuffix(text.String(), "\n")
}

func spreadsheetText(value string) string {
	trimmed := strings.TrimLeftFunc(value, unicode.IsSpace)
	if trimmed != "" && strings.ContainsRune("=+-@＝＋－＠", []rune(trimmed)[0]) {
		return "'" + value
	}
	return value
}
