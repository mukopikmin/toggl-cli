import { assertEquals, assertThrows } from "@std/assert";
import { formatTable } from "./table.ts";

Deno.test("formatTable aligns empty, Unicode, multiline, and long cells", () => {
  assertEquals(
    formatTable(["名前", "Note"], [["開発", "first\nsecond"], [
      "",
      "long value",
    ]]),
    [
      "┌──────┬────────────┐",
      "│ 名前 │ Note       │",
      "├──────┼────────────┤",
      "│ 開発 │ first      │",
      "│      │ second     │",
      "│      │ long value │",
      "└──────┴────────────┘",
    ].join("\n"),
  );
});

Deno.test("formatTable renders headers and borders for an empty dataset", () => {
  assertEquals(
    formatTable(["A", "B"], []),
    "┌───┬───┐\n│ A │ B │\n├───┼───┤\n└───┴───┘",
  );
});

Deno.test("formatTable emits no ANSI decoration", () => {
  assertEquals(formatTable(["A"], [["value"]]).includes("\x1b"), false);
});

Deno.test("formatTable rejects inconsistent rows", () => {
  assertThrows(() => formatTable(["A"], [["one", "two"]]), Error);
});
