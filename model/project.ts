import type { TogglProject } from "../toggl/types.ts";
import type { ProjectConfig } from "../config.ts";

export interface Project {
  id: number;
  name: string;
  displayName: string;
  active: boolean;
  hidden: boolean;
}

export function createProject(
  project: TogglProject,
  projectConfigs: Record<number, ProjectConfig>,
): Project {
  const config = projectConfigs[project.id];

  return {
    ...project,
    displayName: config?.displayName ?? project.name,
    hidden: config?.hidden ?? false,
  };
}

export function createProjects(
  projects: TogglProject[],
  projectConfigs: Record<number, ProjectConfig>,
): Project[] {
  return projects.map((project) => createProject(project, projectConfigs));
}

export function visibleProjects(projects: Project[]): Project[] {
  return projects.filter((project) => !project.hidden);
}
