import { assertEquals } from "@std/assert";
import { compileArgs } from "./install.ts";

Deno.test("compileArgs grants the installed executable required permissions", () => {
  assertEquals(compileArgs(["--version", "1.2.3"], "/tmp/toggl"), [
    "compile",
    "--quiet",
    "-P=app",
    "--version",
    "1.2.3",
    "--output",
    "/tmp/toggl",
    "main.ts",
  ]);
});
