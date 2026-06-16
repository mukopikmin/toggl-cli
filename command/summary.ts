import { loadConfig } from "../config.ts";
import { createProjects, visibleProjects } from "../model/project.ts";
import type { Project } from "../model/project.ts";
import type { TogglClient } from "../toggl/api.ts";

export type SummaryFormat = "csv" | "json";

export interface SummaryCommand {
  startDay: Temporal.PlainDate;
  endDay: Temporal.PlainDate;
  separator: string;
  format: SummaryFormat;
}

export interface WorkTimeTable {
  projectNames: string[];
  headers: string[];
  rows: string[][];
}

function formatDay(day: Temporal.PlainDate): string {
  return day.toString();
}

function daysBetween(
  startDay: Temporal.PlainDate,
  endDay: Temporal.PlainDate,
): Temporal.PlainDate[] {
  const days: Temporal.PlainDate[] = [];

  for (
    let d = startDay;
    Temporal.PlainDate.compare(d, endDay) <= 0;
    d = d.add({ days: 1 })
  ) {
    days.push(d);
  }

  return days;
}

export function buildWorkTimeTable(
  projects: Project[],
  dateEntries: Record<string, Record<number, number>>,
  startDay: Temporal.PlainDate,
  endDay: Temporal.PlainDate,
): WorkTimeTable {
  const days = daysBetween(startDay, endDay);
  const headers = days.map(formatDay);
  const rows = projects.map((project) =>
    headers.map((date) => {
      const duration = dateEntries[date]?.[project.id];
      if (duration) {
        return (Math.round(duration * 100) / 100).toString();
      }
      return " ";
    })
  );

  return {
    projectNames: projects.map((project) => project.displayName),
    headers,
    rows,
  };
}

export function outputWorkTimeTable(
  table: WorkTimeTable,
  separator: string,
): void {
  console.log("--- Project list ---");
  for (const projectName of table.projectNames) {
    console.log(projectName);
  }
  console.log();
  console.log(
    "--- Work time table (The order of the rows follows the projects list) ---",
  );
  console.log(table.headers.join(separator));

  for (const row of table.rows) {
    console.log(row.join(separator));
  }
}

export function formatTimeEntriesJson(
  dateEntries: Record<string, Record<number, number>>,
): string {
  return JSON.stringify(dateEntries, null, 2);
}

export function outputTimeEntriesJson(
  dateEntries: Record<string, Record<number, number>>,
): void {
  console.log(formatTimeEntriesJson(dateEntries));
}

export async function runSummaryCommand(
  cmd: SummaryCommand,
  toggl: TogglClient,
): Promise<void> {
  const { startDay, endDay, separator, format } = cmd;

  const config = await loadConfig();
  const dateEntries = await toggl.getTimeEntriesForDays(
    config,
    startDay,
    endDay,
  );

  if (format === "json") {
    outputTimeEntriesJson(dateEntries);
    return;
  }

  const projects = createProjects(
    await toggl.getProjects(config),
    config.PROJECTS,
  );
  const table = buildWorkTimeTable(
    visibleProjects(projects),
    dateEntries,
    startDay,
    endDay,
  );

  outputWorkTimeTable(table, separator);
}
