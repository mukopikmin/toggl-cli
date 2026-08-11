/**
 * Converts Toggl API project DTOs at the external-service boundary into the
 * domain model used by commands. This is not a CLI output formatter: output
 * formatting remains in `command/projects.ts` and `command/summary.ts`.
 */
import type { Project } from "../model/project.ts";
import type { TogglProject } from "./types.ts";

/**
 * Domain-facing display settings. This deliberately does not expose the
 * configuration file's `ProjectConfig` type to the domain model.
 */
export interface ProjectDisplaySettings {
  displayName?: string;
  hidden: boolean;
  displayOrder?: number;
}

export function mapTogglProject(
  project: TogglProject,
  settings?: ProjectDisplaySettings,
): Project {
  return {
    ...project,
    displayName: settings?.displayName ?? project.name,
    hidden: settings?.hidden ?? false,
    ...(settings?.displayOrder === undefined
      ? {}
      : { displayOrder: settings.displayOrder }),
  };
}

export function mapTogglProjects(
  projects: TogglProject[],
  settingsByProjectId: Record<number, ProjectDisplaySettings>,
): Project[] {
  return projects.map((project) =>
    mapTogglProject(project, settingsByProjectId[project.id])
  );
}
