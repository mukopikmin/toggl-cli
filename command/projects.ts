import { loadConfig } from "../config.ts";
import type { TogglClient } from "../toggl/api.ts";
import type { Project } from "../toggl/types.ts";

export type ProjectsFormat = "csv" | "json";

export interface ProjectsCommand {
  format: ProjectsFormat;
}

export function applyProjectDisplayNames(
  projects: Project[],
  projectNames: Record<number, string>,
): Project[] {
  return projects.map((project) => ({
    ...project,
    name: projectNames[project.id] ?? project.name,
  }));
}

export function formatProjectList(projects: Project[]): string {
  return projects.map((p) => p.name).join("\n");
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
  const projects = applyProjectDisplayNames(
    await toggl.getProjects(config),
    config.PROJECT_NAMES,
  );

  outputProjects(projects, cmd.format);
}
