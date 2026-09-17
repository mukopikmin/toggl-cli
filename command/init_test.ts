import { assertEquals, assertRejects } from "@std/assert";
import { join } from "node:path";
import { writeInitialConfig } from "./init.ts";

Deno.test("writeInitialConfig creates a protected config without overwriting it", async () => {
  const directory = await Deno.makeTempDir();

  try {
    const configFile = join(directory, "nested", "config.toml");
    const configToml = 'workspace = "123"\ntoken = "secret"\n';

    await writeInitialConfig(configFile, configToml);

    assertEquals(await Deno.readTextFile(configFile), configToml);

    if (Deno.build.os !== "windows") {
      const { mode } = await Deno.stat(configFile);
      assertEquals(mode! & 0o777, 0o600);
    }

    await assertRejects(
      () => writeInitialConfig(configFile, 'token = "replacement"\n'),
      Deno.errors.AlreadyExists,
    );
    assertEquals(await Deno.readTextFile(configFile), configToml);
  } finally {
    await Deno.remove(directory, { recursive: true });
  }
});
