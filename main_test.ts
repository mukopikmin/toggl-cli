import { assertEquals } from "@std/assert";
import { formatProjectList } from "./main.ts";

Deno.test("formatProjectList returns one project name per line", () => {
  assertEquals(
    formatProjectList([
      { id: 1, name: "Project Alpha", active: true },
      { id: 2, name: "Project Beta", active: true },
    ]),
    "Project Alpha\nProject Beta",
  );
});

Deno.test("formatProjectList returns an empty string for no projects", () => {
  assertEquals(formatProjectList([]), "");
});
