/** Formats tabular data using plain-text terminal borders and no ANSI escapes. */
export function formatTable(headers: string[], rows: string[][]): string {
  if (headers.length === 0) return "";
  if (rows.some((row) => row.length !== headers.length)) {
    throw new Error("table rows must have the same number of cells as headers");
  }

  const lines = [headers, ...rows].map((row) =>
    row.map((cell) => String(cell).split(/\r\n|\r|\n/))
  );
  const widths = headers.map((_, column) =>
    Math.max(
      ...lines.flatMap((row) => row[column].map(displayWidth)),
    )
  );
  const border = (left: string, middle: string, right: string) =>
    left + widths.map((width) => "─".repeat(width + 2)).join(middle) + right;
  const render = (row: string[][]) => {
    const height = Math.max(...row.map((cell) => cell.length));
    return Array.from(
      { length: height },
      (_, line) =>
        "│" + row.map((cell, column) => {
          const value = cell[line] ?? "";
          return ` ${value}${
            " ".repeat(widths[column] - displayWidth(value))
          } `;
        }).join("│") + "│",
    );
  };

  return [
    border("┌", "┬", "┐"),
    ...render(lines[0]),
    border("├", "┼", "┤"),
    ...lines.slice(1).flatMap(render),
    border("└", "┴", "┘"),
  ].join("\n");
}

function displayWidth(value: string): number {
  let width = 0;
  for (const character of value) {
    if (/\p{Mark}/u.test(character)) continue;
    const codePoint = character.codePointAt(0)!;
    width += isWide(codePoint) ? 2 : 1;
  }
  return width;
}

function isWide(codePoint: number): boolean {
  return codePoint >= 0x1100 && (
    codePoint <= 0x115f || codePoint === 0x2329 || codePoint === 0x232a ||
    (codePoint >= 0x2e80 && codePoint <= 0xa4cf) ||
    (codePoint >= 0xac00 && codePoint <= 0xd7a3) ||
    (codePoint >= 0xf900 && codePoint <= 0xfaff) ||
    (codePoint >= 0xfe10 && codePoint <= 0xfe6f) ||
    (codePoint >= 0xff00 && codePoint <= 0xff60) ||
    (codePoint >= 0x1f300 && codePoint <= 0x1faff) ||
    (codePoint >= 0x20000 && codePoint <= 0x3fffd)
  );
}
