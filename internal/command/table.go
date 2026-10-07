package command

import (
	"strings"
	"unicode"
)

func table(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}
	all := append([][]string{headers}, rows...)
	cells := make([][][]string, len(all))
	widths := make([]int, len(headers))
	for i, row := range all {
		if len(row) != len(headers) {
			panic("table rows must have the same number of cells as headers")
		}
		cells[i] = make([][]string, len(row))
		for j, value := range row {
			normalized := strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n")
			cells[i][j] = strings.Split(normalized, "\n")
			for _, line := range cells[i][j] {
				if w := displayWidth(line); w > widths[j] {
					widths[j] = w
				}
			}
		}
	}
	border := func(left, middle, right string) string {
		parts := make([]string, len(widths))
		for i, w := range widths {
			parts[i] = strings.Repeat("─", w+2)
		}
		return left + strings.Join(parts, middle) + right
	}
	render := func(row [][]string) []string {
		height := 1
		for _, cell := range row {
			if len(cell) > height {
				height = len(cell)
			}
		}
		lines := make([]string, height)
		for line := 0; line < height; line++ {
			parts := make([]string, len(row))
			for col, cell := range row {
				value := ""
				if line < len(cell) {
					value = cell[line]
				}
				parts[col] = " " + value + strings.Repeat(" ", widths[col]-displayWidth(value)) + " "
			}
			lines[line] = "│" + strings.Join(parts, "│") + "│"
		}
		return lines
	}
	out := []string{border("┌", "┬", "┐")}
	out = append(out, render(cells[0])...)
	out = append(out, border("├", "┼", "┤"))
	for _, row := range cells[1:] {
		out = append(out, render(row)...)
	}
	out = append(out, border("└", "┴", "┘"))
	return strings.Join(out, "\n")
}

func displayWidth(value string) int {
	width := 0
	for _, r := range value {
		if unicode.Is(unicode.M, r) {
			continue
		}
		if isWide(r) {
			width += 2
		} else {
			width++
		}
	}
	return width
}
func isWide(r rune) bool {
	return r >= 0x1100 && (r <= 0x115f || r == 0x2329 || r == 0x232a || r >= 0x2e80 && r <= 0xa4cf || r >= 0xac00 && r <= 0xd7a3 || r >= 0xf900 && r <= 0xfaff || r >= 0xfe10 && r <= 0xfe6f || r >= 0xff00 && r <= 0xff60 || r >= 0x1f300 && r <= 0x1faff || r >= 0x20000 && r <= 0x3fffd)
}
