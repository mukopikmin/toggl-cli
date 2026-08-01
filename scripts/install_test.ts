import { assertEquals } from "@std/assert";
import { compileArgs } from "./install.ts";

Deno.test("compileArgs grants the installed executable required permissions", () => {
  assertEquals(compileArgs(["--version", "1.2.3"], "/tmp/toggl"), [
    "compile",
    "--allow-net",
    "--allow-read",
    "--allow-write",
    "--allow-run=pbcopy,wl-copy,xclip,xsel,clip,powershell.exe,powershell",
    "--allow-env",
    "--version",
    "1.2.3",
    "--output",
    "/tmp/toggl",
    "main.ts",
  ]);
});
