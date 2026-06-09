import { join } from "@std/path";
import { parse } from "@std/toml";
import type { TogglConfig } from "./toggl/types.ts";

export interface Config extends TogglConfig {
  PROJECT_NAMES: Record<number, string>;
}

function readString(value: unknown): string | undefined {
  return typeof value === "string" && value ? value : undefined;
}

export function parseProjectNames(value: unknown): Record<number, string> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return {};
  }

  const projectNames: Record<number, string> = {};
  for (const [key, name] of Object.entries(value)) {
    const projectId = Number(key);
    if (!Number.isNaN(projectId) && typeof name === "string") {
      projectNames[projectId] = name;
    }
  }

  return projectNames;
}

export function parseConfigToml(text: string): Config {
  const parsed = parse(text);
  const config = {
    WORKSPACE: readString(parsed.workspace),
    TOKEN: readString(parsed.token),
    PROJECT_NAMES: parseProjectNames(parsed.project_names),
  };

  const missingKeys = [
    !config.WORKSPACE ? "workspace" : undefined,
    !config.TOKEN ? "token" : undefined,
  ].filter((key): key is string => key !== undefined);

  if (missingKeys.length > 0) {
    console.error(
      `Error: Missing required configuration in ~/.toggl_config.toml: ${
        missingKeys.join(", ")
      }`,
    );
    Deno.exit(1);
  }

  return {
    WORKSPACE: config.WORKSPACE,
    TOKEN: config.TOKEN,
    PROJECT_NAMES: config.PROJECT_NAMES,
  } as Config;
}

export async function loadConfig(): Promise<Config> {
  const home = Deno.env.get("HOME");
  if (!home) {
    console.error("Error: HOME environment variable not set");
    Deno.exit(1);
  }
  const configFile = join(home, ".toggl_config.toml");

  try {
    const text = await Deno.readTextFile(configFile);
    return parseConfigToml(text);
  } catch (error) {
    if (error instanceof Deno.errors.NotFound) {
      console.error("Error: ~/.toggl_config.toml file not found");
      console.error(
        "Please create ~/.toggl_config.toml with the following format:",
      );
      console.error('workspace = "your_workspace_id"');
      console.error('token = "your_api_token"');
      Deno.exit(1);
    }
    throw error;
  }
}
