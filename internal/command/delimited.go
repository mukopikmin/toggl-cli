package command

import (
	"encoding/csv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func delimited(headers []string, rows [][]string, format string) string {
	return renderDelimited(headers, rows, format, false)
}

func renderDelimited(headers []string, rows [][]string, format string, quoteEmptyRecords bool) string {
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
		if quoteEmptyRecords && len(row) == 1 && row[0] == "" {
			// CSV readers skip a blank line. Quote the empty field to keep the record.
			w.Flush()
			text.WriteString("\"\"\n")
			continue
		}
		_ = w.Write(row)
	}
	w.Flush()
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
