import { apiEndpoint } from "./api.ts";
import { TogglApiError } from "./error.ts";
import type {
  Project,
  ProjectDisplaySettings,
} from "../model/project.ts";
import type { TogglConfig } from "./types.ts";
export function mapProjectResponse(
  project: ProjectResponse,
  settings?: ProjectDisplaySettings,
): Project {
  const name = project.name ?? project.project_name;
  return {
    id: project.id,
    name,
    displayName: settings?.displayName ?? name,
    active: project.active ?? project.project_active,
    hidden: settings?.hidden ?? false,
    ...(settings?.displayOrder === undefined
      ? {}
      : { displayOrder: settings.displayOrder }),
  };
}

  settingsByProjectId: Record<number, ProjectDisplaySettings> = {},
): Promise<Project[]> {
    .map((project) =>
      mapProjectResponse(project, settingsByProjectId[project.id])
    );
  wid: number;
  pid: number;
  project_name: string;
  project_color: string;
  project_active: boolean;
  project_billable: boolean;
  user_name: string;
  user_avatar_url: string;
  // Fallbacks matching app.rb
  name?: string;
  active?: boolean;
}

export async function getProjects(
  config: TogglConfig,
): Promise<TogglProject[]> {
  const url = `${apiEndpoint}/workspaces/${config.WORKSPACE}/projects`;
  const response = await fetch(url, {
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Basic ${btoa(`${config.TOKEN}:api_token`)}`,
    },
  });

  if (!response.ok) {
    throw new TogglApiError(
      "fetch projects",
      response.status,
      url,
      response.statusText,
    );
  }

  const projects = await response.json() as ProjectResponse[];

  return projects
    .filter((p) => p.active ?? p.project_active)
    .map((p) => ({
      id: p.id,
      name: p.name ?? p.project_name,
      active: p.active ?? p.project_active,
    }));
}
