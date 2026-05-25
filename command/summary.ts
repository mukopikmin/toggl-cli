import { DateTime } from "ptera";
import { loadConfig } from "../config.ts";
import type { TogglClient } from "../toggl/api.ts";
import type { Project } from "../toggl/types.ts";

export interface SummaryCommand {
  startDay: DateTime;
  endDay: DateTime;
  separator: string;
}

export interface WorkTimeTable {
  projectNames: string[];
  headers: string[];
  rows: string[][];
}

function formatDay(day: DateTime): string {
  const y = day.year;
  const m = String(day.month).padStart(2, "0");
  const d = String(day.day).padStart(2, "0");

  return `${y}-${m}-${d}`;
}

function daysBetween(startDay: DateTime, endDay: DateTime): DateTime[] {
  const days: DateTime[] = [];

  for (
    let d = startDay;
    !d.isAfter(endDay);
    d = d.add({ day: 1 })
  ) {
    days.push(d);
  }

  return days;
}

export function buildWorkTimeTable(
  projects: Project[],
  dateEntries: Record<string, Record<number, number>>,
  startDay: DateTime,
  endDay: DateTime,
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
    projectNames: projects.map((project) => project.name),
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

export async function runSummaryCommand(
  cmd: SummaryCommand,
  toggl: TogglClient,
): Promise<void> {
  const { startDay, endDay, separator } = cmd;

  const config = await loadConfig();
  const projects = await toggl.getProjects(config);
  const dateEntries = await toggl.getTimeEntriesForDays(
    config,
    startDay,
    endDay,
  );
  const table = buildWorkTimeTable(projects, dateEntries, startDay, endDay);

  outputWorkTimeTable(table, separator);
}
