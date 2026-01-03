import { apiEndpoint } from "./api.ts";
import type { Project, TogglConfig } from "./types.ts";

/**
 * https://engineering.toggl.com/docs/api/projects/
 */

interface ProjectResponse {
  id: number;
  workspace_id: number;
  project_id: number;
  task_id: number;
  billable: boolean;
  start: string;
  stop: string;
  duration: number;
  description: string;
  duronly: boolean;
  at: string;
  server_deleted_at: string;
  user_id: number;
  uid: number;
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

export async function getProjects(config: TogglConfig): Promise<Project[]> {
  const url = `${apiEndpoint}/workspaces/${config.WORKSPACE}/projects`;
  const response = await fetch(url, {
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Basic ${btoa(`${config.TOKEN}:api_token`)}`,
    },
  });

  if (!response.ok) {
    console.error(`Failed to fetch projects: ${response.statusText}`);
    Deno.exit(1);
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
