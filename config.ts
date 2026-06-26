import { join } from "node:path";
import { parse } from "@std/toml";
import type { TogglConfig } from "./toggl/types.ts";

export const CONFIG_FILE_DISPLAY = "~/.config/toggl-cli/config.toml";

export interface ProjectConfig {
  displayName?: string;
  hidden: boolean;
}

export interface Config extends TogglConfig {
  PROJECTS: Record<number, ProjectConfig>;
}

export interface ConfigDocument {
  configFile: string;
  text: string;
  config: Config;
}

export function getConfigFile(home: string): string {
  return join(home, ".config", "toggl-cli", "config.toml");
}

function readString(value: unknown): string | undefined {
  return typeof value === "string" && value ? value : undefined;
}

function readBoolean(value: unknown): boolean | undefined {
  return typeof value === "boolean" ? value : undefined;
}

export function parseProjectsConfig(
  value: unknown,
): Record<number, ProjectConfig> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return {};
  }

  const projects: Record<number, ProjectConfig> = {};
  for (const [key, rawProject] of Object.entries(value)) {
    const projectId = Number(key);
    if (
      Number.isNaN(projectId) || !rawProject ||
      typeof rawProject !== "object" || Array.isArray(rawProject)
    ) {
      continue;
    }

    const project = rawProject as Record<string, unknown>;
    projects[projectId] = {
      displayName: readString(project.display_name),
      hidden: readBoolean(project.hidden) ?? false,
    };
  }

  return projects;
}

export function parseConfigToml(text: string): Config {
  const parsed = parse(text);
  const config = {
    WORKSPACE: readString(parsed.workspace),
    TOKEN: readString(parsed.token),
    TIMEZONE: readString(parsed.timezone),
    PROJECTS: parseProjectsConfig(parsed.projects),
  };

  const missingKeys = [
    !config.WORKSPACE ? "workspace" : undefined,
    !config.TOKEN ? "token" : undefined,
  ].filter((key): key is string => key !== undefined);

  if (missingKeys.length > 0) {
    console.error(
      `Error: Missing required configuration in ${CONFIG_FILE_DISPLAY}: ${
        missingKeys.join(", ")
      }`,
    );
    Deno.exit(1);
  }

  return {
    WORKSPACE: config.WORKSPACE,
    TOKEN: config.TOKEN,
    TIMEZONE: config.TIMEZONE,
    PROJECTS: config.PROJECTS,
  } as Config;
}

export async function loadConfigDocument(): Promise<ConfigDocument> {
  const home = Deno.env.get("HOME");
  if (!home) {
    console.error("Error: HOME environment variable not set");
    Deno.exit(1);
  }
  const configFile = getConfigFile(home);

  try {
    const text = await Deno.readTextFile(configFile);
    return {
      configFile,
      text,
      config: parseConfigToml(text),
    };
  } catch (error) {
    if (error instanceof Deno.errors.NotFound) {
      console.error(`Error: ${CONFIG_FILE_DISPLAY} file not found`);
      console.error(
        `Please create ${CONFIG_FILE_DISPLAY} with the following format:`,
      );
      console.error('workspace = "your_workspace_id"');
      console.error('token = "your_api_token"');
      Deno.exit(1);
    }
    throw error;
  }
}

export async function loadConfig(): Promise<Config> {
  return (await loadConfigDocument()).config;
}
