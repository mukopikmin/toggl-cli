import type { TogglProject } from "../toggl/types.ts";
import type { ProjectConfig } from "../config.ts";

export interface Project {
  id: number;
  name: string;
  displayName: string;
  active: boolean;
  hidden: boolean;
  displayOrder?: number;
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
    ...(config?.displayOrder === undefined
      ? {}
      : { displayOrder: config.displayOrder }),
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

export function sortProjectsByDisplayOrder(projects: Project[]): Project[] {
  return projects.toSorted((a, b) => {
    if (a.displayOrder === undefined && b.displayOrder === undefined) {
      return 0;
    }
    if (a.displayOrder === undefined) {
      return 1;
    }
    if (b.displayOrder === undefined) {
      return -1;
    }
    return a.displayOrder - b.displayOrder;
  });
}
