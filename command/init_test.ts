import { assertEquals, assertRejects } from "@std/assert";
import {
  InitInput,
  InitInputError,
  promptForInitialConfig,
  runInitCommand,
} from "./init.ts";
import { getConfigFile } from "../config.ts";

function input(
  values: Array<string | null>,
  interactive = true,
): InitInput & { output: string[] } {
  const output: string[] = [];
  return {
    interactive,
    output,
    readLine: () => Promise.resolve(values.shift() ?? null),
    write: (text) => {
      output.push(text);
      return Promise.resolve();
    },
  };
}

Deno.test("promptForInitialConfig accepts valid input", async () => {
  assertEquals(
    await promptForInitialConfig(
      input(["workspace-id", "test-token", "America/New_York"]),
    ),
    {
      workspace: "workspace-id",
      token: "test-token",
      timezone: "America/New_York",
    },
  );
});

Deno.test("promptForInitialConfig retries an empty workspace", async () => {
  const io = input(["", "workspace-id", "test-token", "UTC"]);
  const config = await promptForInitialConfig(io);

  assertEquals(config.workspace, "workspace-id");
  assertEquals(
    io.output.includes("Workspace is required. Please enter a value.\n"),
    true,
  );
});

Deno.test("promptForInitialConfig retries an empty API token", async () => {
  const io = input(["workspace-id", "", "test-token", "UTC"]);
  const config = await promptForInitialConfig(io);

  assertEquals(config.token, "test-token");
  assertEquals(
    io.output.includes("API token is required. Please enter a value.\n"),
    true,
  );
});

Deno.test("promptForInitialConfig defaults an empty timezone", async () => {
  const config = await promptForInitialConfig(
    input(["workspace-id", "test-token", ""]),
  );

  assertEquals(config.timezone, "Asia/Tokyo");
});

Deno.test("promptForInitialConfig rejects EOF during input", async () => {
  await assertRejects(
    () => promptForInitialConfig(input(["workspace-id", null])),
    InitInputError,
    "Input ended before API token was provided",
  );
});

Deno.test("promptForInitialConfig rejects empty non-interactive input", async () => {
  await assertRejects(
    () => promptForInitialConfig(input([""], false)),
    InitInputError,
    "Workspace is required",
  );
});

Deno.test("runInitCommand does not create a config file after invalid input", async () => {
  const previousHome = Deno.env.get("HOME");
  const home = await Deno.makeTempDir();
  Deno.env.set("HOME", home);
  try {
    await assertRejects(
      () => runInitCommand(input(["workspace-id", ""], false)),
      InitInputError,
      "API token is required",
    );
    await assertRejects(
      () => Deno.stat(getConfigFile(home)),
      Deno.errors.NotFound,
    );
  } finally {
    if (previousHome === undefined) Deno.env.delete("HOME");
    else Deno.env.set("HOME", previousHome);
    await Deno.remove(home, { recursive: true });
  }
});
