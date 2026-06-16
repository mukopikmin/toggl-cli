import { loadConfig } from "../config.ts";
import { createProjects, visibleProjects } from "../model/project.ts";
import type { Project } from "../model/project.ts";
import type { TogglClient } from "../toggl/api.ts";

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
