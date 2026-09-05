import { stringify } from "@std/toml";
import { loadConfig, loadConfigDocument, parseConfigToml } from "../config.ts";
import {
  sortProjectsByDisplayOrder,
  visibleProjects,
} from "../model/project.ts";
import type { Project } from "../model/project.ts";
import type { TogglClient } from "../toggl/api.ts";

export type ProjectFormat = "csv" | "json";

export interface ProjectListCommand {
  format: ProjectFormat;
}

export type ProjectReorderAction =
  | "select-up"
  | "select-down"
  | "move-up"
  | "move-down";

export interface ProjectReorderState {
  projects: Project[];
  selectedIndex: number;
}

export interface ProjectReorderTerminal {
  isTerminal(): boolean;
  setRaw(enabled: boolean): void;
  read(buffer: Uint8Array): Promise<number | null>;
  write(text: string): void;
}

export class ProjectReorderUnavailableError extends Error {
  constructor() {
    super("project reorder requires an interactive terminal");
    this.name = "ProjectReorderUnavailableError";
  }
}

export function formatProjectList(projects: Project[]): string {
  return projects.map((p) => p.displayName).join("\n");
}

export function formatProjectsJson(projects: Project[]): string {
  return JSON.stringify(projects, null, 2);
}

export function outputProjects(
  projects: Project[],
  format: ProjectFormat,
): void {
  console.log(
    format === "json"
      ? formatProjectsJson(projects)
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

export function updateProjectReorderState(
  state: ProjectReorderState,
  action: ProjectReorderAction,
): ProjectReorderState {
  if (state.projects.length === 0) return state;

  const lastIndex = state.projects.length - 1;
  if (action === "select-up") {
    return { ...state, selectedIndex: Math.max(0, state.selectedIndex - 1) };
  }
  if (action === "select-down") {
    return {
      ...state,
      selectedIndex: Math.min(lastIndex, state.selectedIndex + 1),
    };
  }

  const destination = action === "move-up"
    ? state.selectedIndex - 1
    : state.selectedIndex + 1;
  if (destination < 0 || destination > lastIndex) return state;

  const projects = [...state.projects];
  [projects[state.selectedIndex], projects[destination]] = [
    projects[destination],
    projects[state.selectedIndex],
  ];
  return { projects, selectedIndex: destination };
}

export function updateProjectDisplayOrders(
  configText: string,
  projectIds: number[],
): string {
  if (projectIds.length === 0) return configText;

  const newline = configText.includes("\r\n") ? "\r\n" : "\n";
  const endsWithNewline = configText.endsWith("\n");
  const lines = configText.split(/\r?\n/);
  if (endsWithNewline) lines.pop();
  const remaining = new Map(projectIds.map((id, index) => [id, index + 1]));
  const output: string[] = [];

  for (let index = 0; index < lines.length;) {
    const header = lines[index].match(
      /^\s*\[\s*projects\.(?:"(\d+)"|'(\d+)'|(\d+))\s*\]\s*(?:#.*)?$/,
    );
    if (!header) {
      output.push(lines[index++]);
      continue;
    }

    const projectId = Number(header[1] ?? header[2] ?? header[3]);
    const order = remaining.get(projectId);
    output.push(lines[index++]);
    let foundDisplayOrder = false;
    while (index < lines.length && !/^\s*\[/.test(lines[index])) {
      if (order !== undefined && /^\s*display_order\s*=/.test(lines[index])) {
        const indent = lines[index].match(/^\s*/)?.[0] ?? "";
        const comment = lines[index].match(/\s+#.*$/)?.[0] ?? "";
        output.push(`${indent}display_order = ${order}${comment}`);
        foundDisplayOrder = true;
      } else {
        output.push(lines[index]);
      }
      index++;
    }
    if (order !== undefined) {
      if (!foundDisplayOrder) output.push(`display_order = ${order}`);
      remaining.delete(projectId);
    }
  }

  for (const [projectId, order] of remaining) {
    if (output.length > 0 && output.at(-1) !== "") output.push("");
    output.push(`[projects."${projectId}"]`, `display_order = ${order}`);
  }

  return `${output.join(newline)}${endsWithNewline ? newline : ""}`;
}

function renderProjectReorder(state: ProjectReorderState): string {
  const rows = state.projects.map((project, index) =>
    `${index === state.selectedIndex ? ">" : " "} ${project.displayName}`
  );
  return [
    "\x1b[2J\x1b[HReorder projects",
    "Up/Down: select  Ctrl+Up/Ctrl+Down: move  Enter: save  q/Esc: cancel",
    "",
    ...rows,
  ].join("\n");
}

function projectReorderAction(input: string): ProjectReorderAction | undefined {
  switch (input) {
    case "\x1b[A":
    case "k":
      return "select-up";
    case "\x1b[B":
    case "j":
      return "select-down";
    case "\x1b[1;5A":
    case "K":
      return "move-up";
    case "\x1b[1;5B":
    case "J":
      return "move-down";
  }
}

const defaultProjectReorderTerminal: ProjectReorderTerminal = {
  isTerminal: () => Deno.stdin.isTerminal() && Deno.stdout.isTerminal(),
  setRaw: (enabled) => Deno.stdin.setRaw(enabled),
  read: (buffer) => Deno.stdin.read(buffer),
  write: (text) => Deno.stdout.writeSync(new TextEncoder().encode(text)),
};

export async function selectProjectOrder(
  projects: Project[],
  terminal: ProjectReorderTerminal = defaultProjectReorderTerminal,
): Promise<Project[] | undefined> {
  if (projects.length === 0) return projects;
  if (!terminal.isTerminal()) throw new ProjectReorderUnavailableError();

  let state: ProjectReorderState = { projects, selectedIndex: 0 };
  const buffer = new Uint8Array(16);
  terminal.setRaw(true);
  terminal.write("\x1b[?25l");
  try {
    while (true) {
      terminal.write(renderProjectReorder(state));
      const bytesRead = await terminal.read(buffer);
      if (bytesRead === null) return undefined;
      const input = new TextDecoder().decode(buffer.subarray(0, bytesRead));
      if (input === "\r" || input === "\n") return state.projects;
      if (input === "q" || input === "\x1b") return undefined;
      const action = projectReorderAction(input);
      if (action) state = updateProjectReorderState(state, action);
    }
  } finally {
    terminal.write("\x1b[2J\x1b[H\x1b[?25h");
    terminal.setRaw(false);
  }
}

export async function runProjectReorderCommand(
  toggl: TogglClient,
): Promise<void> {
  const document = await loadConfigDocument();
  const projects = sortProjectsByDisplayOrder(visibleProjects(
    await toggl.getProjects(document.config, document.config.PROJECTS),
  ));
  const reordered = await selectProjectOrder(projects);
  if (!reordered) {
    console.log("Project order was not changed");
    return;
  }
  if (reordered.length === 0) {
    console.log("No visible projects to reorder");
    return;
  }

  const text = updateProjectDisplayOrders(
    document.text,
    reordered.map((project) => project.id),
  );
  parseConfigToml(text);
  await Deno.writeTextFile(document.configFile, text);
  console.log(`Saved the order of ${reordered.length} project(s)`);
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
