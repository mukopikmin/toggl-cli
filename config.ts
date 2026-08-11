import { join } from "node:path";
import { parse } from "@std/toml";
import type { TogglConfig } from "./toggl/types.ts";

export const CONFIG_FILE_DISPLAY = "~/.config/toggl-cli/config.toml";

export interface ProjectConfig {
  displayName?: string;
  hidden: boolean;
  displayOrder?: number;
}

export interface Config extends TogglConfig {
  PROJECTS: Record<number, ProjectConfig>;
}

export interface ConfigDocument {
  configFile: string;
  text: string;
  config: Config;
}

export interface ConfigFileAdapter {
  getHome(): string | undefined;
  readTextFile(path: string): Promise<string>;
  isNotFound(error: unknown): boolean;
}

export interface ConfigValidationIssue {
  path: string;
  code: "missing_required" | "invalid_project";
}

export class ConfigValidationError extends Error {
  constructor(public readonly issues: ConfigValidationIssue[]) {
    super("Invalid configuration");
    this.name = "ConfigValidationError";
  }

  get missingKeys(): string[] {
    return this.issues.filter((issue) => issue.code === "missing_required")
      .map((issue) => issue.path);
  }

  get invalidProjects(): string[] {
    return this.issues.filter((issue) => issue.code === "invalid_project")
      .map((issue) => issue.path);
  }
}

export class HomeNotSetError extends Error {
  constructor() {
    super("HOME environment variable not set");
    this.name = "HomeNotSetError";
  }
}

export class ConfigFileNotFoundError extends Error {
  constructor() {
    super(`${CONFIG_FILE_DISPLAY} file not found`);
    this.name = "ConfigFileNotFoundError";
  }
}

export class ConfigFileReadError extends Error {
  constructor(public readonly configFile: string, options: ErrorOptions) {
    super(`Unable to read ${CONFIG_FILE_DISPLAY}`, options);
    this.name = "ConfigFileReadError";
  }
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

function readNumber(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value)
    ? value
    : undefined;
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
    const displayOrder = readNumber(project.display_order);
    projects[projectId] = {
      displayName: readString(project.display_name),
      hidden: readBoolean(project.hidden) ?? false,
      ...(displayOrder === undefined ? {} : { displayOrder }),
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

  const issues: ConfigValidationIssue[] = [
    ...(!config.WORKSPACE
      ? [{ path: "workspace", code: "missing_required" as const }]
      : []),
    ...(!config.TOKEN
      ? [{ path: "token", code: "missing_required" as const }]
      : []),
    ...validateProjectsConfig(parsed.projects),
  ];

  if (issues.length > 0) throw new ConfigValidationError(issues);

  return {
    WORKSPACE: config.WORKSPACE,
    TOKEN: config.TOKEN,
    TIMEZONE: config.TIMEZONE,
    PROJECTS: config.PROJECTS,
  } as Config;
}

function validateProjectsConfig(value: unknown): ConfigValidationIssue[] {
  if (value === undefined) return [];
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return [{ path: "projects", code: "invalid_project" }];
  }

  const issues: ConfigValidationIssue[] = [];
  for (const [id, valueForProject] of Object.entries(value)) {
    const path = `projects.${id}`;
    if (
      !/^\d+$/.test(id) || !valueForProject ||
      typeof valueForProject !== "object" || Array.isArray(valueForProject)
    ) {
      issues.push({ path, code: "invalid_project" });
      continue;
    }
    const project = valueForProject as Record<string, unknown>;
    if (
      ("display_name" in project &&
        readString(project.display_name) === undefined) ||
      ("hidden" in project && readBoolean(project.hidden) === undefined) ||
      ("display_order" in project &&
        readNumber(project.display_order) === undefined)
    ) {
      issues.push({ path, code: "invalid_project" });
    }
  }
  return issues;
}

const denoConfigFileAdapter: ConfigFileAdapter = {
  getHome: () => Deno.env.get("HOME"),
  readTextFile: (path) => Deno.readTextFile(path),
  isNotFound: (error) => error instanceof Deno.errors.NotFound,
};

export async function loadConfigDocument(
  adapter: ConfigFileAdapter = denoConfigFileAdapter,
): Promise<ConfigDocument> {
  const home = adapter.getHome();
  if (!home) throw new HomeNotSetError();
  const configFile = getConfigFile(home);

  let text: string;
  try {
    text = await adapter.readTextFile(configFile);
  } catch (error) {
    if (adapter.isNotFound(error)) throw new ConfigFileNotFoundError();
    throw new ConfigFileReadError(configFile, { cause: error });
  }

  return {
    configFile,
    text,
    config: parseConfigToml(text),
  };
}

export async function loadConfig(): Promise<Config> {
  return (await loadConfigDocument()).config;
}
