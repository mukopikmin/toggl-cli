import { stringify } from "@std/toml";
import { loadConfig, loadConfigDocument } from "../config.ts";
import {
  sortProjectsByDisplayOrder,
  visibleProjects,
} from "../model/project.ts";
import type { Project } from "../model/project.ts";
import type { TogglClient } from "../toggl/api.ts";
import type { OutputFormat } from "./output_format.ts";
import { formatTable } from "./table.ts";

export type ProjectFormat = OutputFormat;

export interface ProjectListCommand {
  format: ProjectFormat;
}

export function formatProjectList(projects: Project[]): string {
  return projects.map((p) => p.displayName).join("\n");
}

export function formatProjectsJson(projects: Project[]): string {
  return JSON.stringify(projects, null, 2);
}

export function formatProjectsTable(projects: Project[]): string {
  return formatTable(
    ["Project"],
    projects.map((project) => [project.displayName]),
  );
}

export function outputProjects(
  projects: Project[],
  format: ProjectFormat,
): void {
  console.log(
    format === "json"
      ? formatProjectsJson(projects)
      : format === "table"
      ? formatProjectsTable(projects)
      : formatProjectList(projects),
  );
}

export async function runProjectListCommand(
  cmd: ProjectListCommand,
  toggl: TogglClient,
): Promise<void> {
  const config = await loadConfig();
  const projects = await toggl.getProjects(config, config.PROJECTS);

  outputProjects(
    sortProjectsByDisplayOrder(visibleProjects(projects)),
    cmd.format,
  );
}

export function appendMissingProjects(
  configText: string,
  configuredProjectIds: number[],
  projects: (
    & Pick<Project, "id" | "name">
    & Partial<Pick<Project, "active">>
  )[],
): { text: string; addedCount: number } {
  const configuredIds = new Set(configuredProjectIds);
  const missingProjects = projects
    .filter((project) => !configuredIds.has(project.id))
    .toSorted((a, b) => a.id - b.id);

  if (missingProjects.length === 0) {
    return { text: configText, addedCount: 0 };
  }

  const additions = missingProjects.map((project) => {
    const projectNameComment = project.name
      .split(/\r\n|\r|\n/)
      .map((line) => `# ${line}`)
      .join("\n");
    const projectConfig = stringify({
      projects: {
        [String(project.id)]: {
          hidden: false,
        },
      },
    }).trim();

    return `${projectNameComment}\n${projectConfig}`;
  }).join("\n\n");
  const separator = configText.endsWith("\n") ? "\n" : "\n\n";

  return {
    text: `${configText}${separator}${additions}\n`,
    addedCount: missingProjects.length,
  };
}

export async function runProjectSyncCommand(
  toggl: TogglClient,
): Promise<void> {
  const document = await loadConfigDocument();
  const projects = await toggl.getProjects(
    document.config,
    document.config.PROJECTS,
  );
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
