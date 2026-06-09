import { join } from "@std/path";
import type { TogglConfig } from "./toggl/types.ts";

export interface Config extends TogglConfig {
  PROJECT_NAMES: Record<number, string>;
}

export async function loadConfig(): Promise<Config> {
  const home = Deno.env.get("HOME");
  if (!home) {
    console.error("Error: HOME environment variable not set");
    Deno.exit(1);
  }
  const configFile = join(home, ".toggl_config");

  try {
    const text = await Deno.readTextFile(configFile);
    const config: Partial<TogglConfig> = {};
    const projectNames: Record<number, string> = {};

    for (const line of text.split("\n")) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith("#")) continue;

      if (trimmed.includes("=")) {
        const [key, value] = trimmed.split("=", 2);
        const normalizedKey = key.trim();
        const normalizedValue = value.trim();

        if (normalizedKey.startsWith("PROJECT_NAME_")) {
          const projectId = Number(normalizedKey.slice("PROJECT_NAME_".length));
          if (!Number.isNaN(projectId)) {
            projectNames[projectId] = normalizedValue;
          }
          continue;
        }

        if (normalizedKey === "WORKSPACE" || normalizedKey === "TOKEN") {
          config[normalizedKey] = normalizedValue;
        }
      }
    }

    const missingKeys = [
      !config.WORKSPACE ? "WORKSPACE" : undefined,
      !config.TOKEN ? "TOKEN" : undefined,
    ].filter((key): key is string => key !== undefined);

    if (missingKeys.length > 0) {
      console.error(
        `Error: Missing required configuration in ~/.toggl_config: ${
          missingKeys.join(", ")
        }`,
      );
      Deno.exit(1);
    }

    return {
      WORKSPACE: config.WORKSPACE,
      TOKEN: config.TOKEN,
      PROJECT_NAMES: projectNames,
    } as Config;
  } catch (error) {
    if (error instanceof Deno.errors.NotFound) {
      console.error("Error: ~/.toggl_config file not found");
      console.error("Please create ~/.toggl_config with the following format:");
      console.error("WORKSPACE=your_workspace_id");
      console.error("TOKEN=your_api_token");
      Deno.exit(1);
    }
    throw error;
  }
}
