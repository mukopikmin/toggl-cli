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

Deno.test("compiled permissions defer GitHub access until update", async () => {
  const denoConfig = JSON.parse(
    await Deno.readTextFile(new URL("../deno.json", import.meta.url)),
  );

  assertEquals(denoConfig.permissions.app.net, ["api.track.toggl.com"]);
});
