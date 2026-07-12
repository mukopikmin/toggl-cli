import { loadConfig } from "../config.ts";
import {
  createProjects,
  sortProjectsByDisplayOrder,
  visibleProjects,
} from "../model/project.ts";
import type { Project } from "../model/project.ts";
import type { TogglClient } from "../toggl/api.ts";

export type SummaryFormat = "csv" | "json";

interface SummaryOptions {
  separator: string;
  format: SummaryFormat;
  noProject: boolean;
}

export type SummaryDateRange = {
  startDay: Temporal.PlainDate;
  endDay: Temporal.PlainDate;
};

export type SummaryCommand =
  & SummaryOptions
  & (
    | SummaryDateRange
    | { days: number }
  );

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
      return "";
    })
  );

  return {
    projectNames: projects.map((project) => project.displayName),
    headers,
    rows,
  };
}

export function formatWorkTimeTable(
  table: WorkTimeTable,
  separator: string,
  noProject = false,
): string {
  const lines = [
    (noProject ? table.headers : ["Project", ...table.headers]).join(
      separator,
    ),
  ];

  for (const [index, row] of table.rows.entries()) {
    lines.push(
      (noProject ? row : [table.projectNames[index], ...row]).join(separator),
    );
  }

  return lines.join("\n");
}

export function outputWorkTimeTable(
  table: WorkTimeTable,
  separator: string,
  noProject = false,
): void {
  console.log(formatWorkTimeTable(table, separator, noProject));
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

export function resolveSummaryDateRange(
  cmd: SummaryCommand,
  timeZone: string | undefined,
  now: Temporal.Instant = Temporal.Now.instant(),
): SummaryDateRange {
  if (!("days" in cmd)) {
    return { startDay: cmd.startDay, endDay: cmd.endDay };
  }

  const endDay = now.toZonedDateTimeISO(timeZone ?? "UTC").toPlainDate();
  return {
    startDay: endDay.subtract({ days: cmd.days }),
    endDay,
  };
}

export async function runSummaryCommand(
  cmd: SummaryCommand,
  toggl: TogglClient,
): Promise<void> {
  const { separator, format, noProject } = cmd;

  const config = await loadConfig();
  const { startDay, endDay } = resolveSummaryDateRange(cmd, config.TIMEZONE);
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
    sortProjectsByDisplayOrder(visibleProjects(projects)),
    dateEntries,
    startDay,
    endDay,
  );

  outputWorkTimeTable(table, separator, noProject);
}
