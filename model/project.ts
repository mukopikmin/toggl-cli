import type { TogglProject } from "../toggl/types.ts";

export interface Project {
  id: number;
  name: string;
  displayName: string;
  active: boolean;
}

export function createProject(
  project: TogglProject,
  projectNames: Record<number, string>,
): Project {
  return {
    ...project,
    displayName: projectNames[project.id] ?? project.name,
  };
}

export function createProjects(
  projects: TogglProject[],
  projectNames: Record<number, string>,
): Project[] {
  return projects.map((project) => createProject(project, projectNames));
}
