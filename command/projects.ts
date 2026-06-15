import { stringify } from "@std/toml";
import { loadConfig, loadConfigDocument } from "../config.ts";
import { createProjects, visibleProjects } from "../model/project.ts";
import type { Project } from "../model/project.ts";
import type { TogglClient } from "../toggl/api.ts";
import type { TogglProject } from "../toggl/types.ts";

export type ProjectsFormat = "csv" | "json";

export interface ProjectsCommand {
  format: ProjectsFormat;
}

export function formatProjectList(projects: Project[]): string {
  return projects.map((p) => p.displayName).join("\n");
}

export function formatProjectsJson(projects: Project[]): string {
  return JSON.stringify(projects, null, 2);
}

export function outputProjects(
  projects: Project[],
  format: ProjectsFormat,
): void {
  console.log(
    format === "json"
      ? formatProjectsJson(projects)
      : formatProjectList(projects),
  );
}

export async function runProjectsCommand(
  cmd: ProjectsCommand,
  toggl: TogglClient,
): Promise<void> {
  const config = await loadConfig();
  const projects = createProjects(
    await toggl.getProjects(config),
    config.PROJECTS,
  );

  outputProjects(visibleProjects(projects), cmd.format);
}

export function appendMissingProjects(
  configText: string,
  configuredProjectIds: number[],
  projects: TogglProject[],
): { text: string; addedCount: number } {
  const configuredIds = new Set(configuredProjectIds);
  const missingProjects = projects
    .filter((project) => !configuredIds.has(project.id))
    .toSorted((a, b) => a.id - b.id);

  if (missingProjects.length === 0) {
    return { text: configText, addedCount: 0 };
  }

  const additions = missingProjects.map((project) =>
    stringify({
      projects: {
        [String(project.id)]: {
          display_name: project.name,
          hidden: false,
        },
      },
    }).trim()
  ).join("\n\n");
  const separator = configText.endsWith("\n") ? "\n" : "\n\n";

  return {
    text: `${configText}${separator}${additions}\n`,
    addedCount: missingProjects.length,
  };
}

export async function runProjectsSyncCommand(
  toggl: TogglClient,
): Promise<void> {
  const document = await loadConfigDocument();
  const projects = await toggl.getProjects(document.config);
  const result = appendMissingProjects(
    document.text,
    Object.keys(document.config.PROJECTS).map(Number),
    projects,
  );

  if (result.addedCount === 0) {
    console.log("All active projects are already configured");
    return;
  }

  await Deno.writeTextFile(document.configFile, result.text);
  console.log(`Added ${result.addedCount} project(s) to the config file`);
}
