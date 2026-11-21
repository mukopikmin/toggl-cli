package command

import (
	"slices"
	"strconv"
	"strings"
	"unicode"
)

type tableLayout struct {
	maxWidth            int
	truncateFirstColumn bool
}

func table(headers []string, rows [][]string, layout tableLayout, alternatives ...[]string) string {
	if len(headers) == 0 {
		return ""
	}
	measure := func(row []string, widths []int) [][]string {
		if len(row) != len(headers) {
			panic("table rows must have the same number of cells as headers")
		}
		cells := make([][]string, len(row))
		for j, value := range row {
			normalized := strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n")
			cells[j] = strings.Split(normalized, "\n")
			for i, raw := range cells[j] {
				line := terminalText(raw)
				cells[j][i] = line
				widths[j] = max(widths[j], displayWidth(line))
			}
		}
		return cells
	}
	cells := make([][][]string, len(rows))
	bodyWidths := make([]int, len(headers))
	for i, row := range rows {
		cells[i] = measure(row, bodyWidths)
	}
	widths := slices.Clone(bodyWidths)
	header := measure(headers, widths)
	if layout.maxWidth > 0 {
		width := tableWidth(widths)
		for _, alternative := range alternatives {
			if width <= layout.maxWidth {
				break
			}
			candidateWidths := slices.Clone(bodyWidths)
			candidate := measure(alternative, candidateWidths)
			candidateWidth := tableWidth(candidateWidths)
			// A shorter heading can help once the first column is narrowed.
			if candidateWidth < width || (layout.truncateFirstColumn && candidateWidth == width) {
				header, widths, width = candidate, candidateWidths, candidateWidth
			}
		}
		if layout.truncateFirstColumn && width > layout.maxWidth {
			widths[0] = max(1, widths[0]-(width-layout.maxWidth))
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
				valueWidth := displayWidth(value)
				if valueWidth > widths[col] {
					value = clipDisplayWidth(value, widths[col]-1) + "…"
					valueWidth = displayWidth(value)
				}
				parts[col] = " " + value + strings.Repeat(" ", widths[col]-valueWidth) + " "
			}
			lines[line] = "│" + strings.Join(parts, "│") + "│"
		}
		return lines
	}
	out := []string{border("┌", "┬", "┐")}
	out = append(out, render(header)...)
	out = append(out, border("├", "┼", "┤"))
	for _, row := range cells {
		out = append(out, render(row)...)
	}
	out = append(out, border("└", "┴", "┘"))
	return strings.Join(out, "\n")
}

func tableWidth(widths []int) int {
	// Each column adds two padding spaces and a border to the leftmost border.
	width := 1
	for _, w := range widths {
		width += w + 3
	}
	return width
}

func displayWidth(value string) int {
	width := 0
	for _, r := range value {
		width += runeDisplayWidth(r)
	}
	return width
}

func clipDisplayWidth(value string, width int) string {
	for i, r := range value {
		width -= runeDisplayWidth(r)
		if width < 0 {
			return value[:i]
		}
	}
	return value
}

func runeDisplayWidth(r rune) int {
	if unicode.Is(unicode.M, r) {
		return 0
	}
	if isWide(r) {
		return 2
	}
	return 1
}

func isWide(r rune) bool {
	return r >= 0x1100 && (r <= 0x115f || r == 0x2329 || r == 0x232a || r >= 0x2e80 && r <= 0xa4cf || r >= 0xac00 && r <= 0xd7a3 || r >= 0xf900 && r <= 0xfaff || r >= 0xfe10 && r <= 0xfe6f || r >= 0xff00 && r <= 0xff60 || r >= 0x1f300 && r <= 0x1faff || r >= 0x20000 && r <= 0x3fffd)
}

func terminalText(value string) string {
	var out strings.Builder
	for _, r := range value {
		if unicode.IsControl(r) || unicode.In(r, unicode.Bidi_Control) {
			quoted := strconv.QuoteRuneToASCII(r)
			out.WriteString(quoted[1 : len(quoted)-1])
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}
