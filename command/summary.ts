import { writeClipboardText } from "../clipboard.ts";
import { loadConfig } from "../config.ts";
import {
  createProjects,
  sortProjectsByDisplayOrder,
  visibleProjects,
} from "../model/project.ts";
import type { Project } from "../model/project.ts";
import type { TogglClient } from "../toggl/api.ts";

export type SummaryFormat = "csv" | "json";

export interface SummaryCommand {
  startDay: Temporal.PlainDate;
  endDay: Temporal.PlainDate;
  separator: string;
  format: SummaryFormat;
  clipboard: boolean;
}

export interface SummaryOutput {
  writeStdout(text: string): void;
  writeClipboard(text: string): Promise<void>;
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
): string {
  const lines = [
    ["Project", ...table.headers].join(separator),
  ];

  for (const [index, row] of table.rows.entries()) {
    lines.push([table.projectNames[index], ...row].join(separator));
  }

  return lines.join("\n");
}

export function outputWorkTimeTable(
  table: WorkTimeTable,
  separator: string,
): void {
  console.log(formatWorkTimeTable(table, separator));
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

const defaultSummaryOutput: SummaryOutput = {
  writeStdout(text: string): void {
    console.log(text);
  },
  writeClipboard: writeClipboardText,
};

export async function outputSummaryText(
  text: string,
  clipboard: boolean,
  output: SummaryOutput = defaultSummaryOutput,
): Promise<void> {
  output.writeStdout(text);

  if (clipboard) {
    await output.writeClipboard(text);
  }
}

export async function runSummaryCommand(
  cmd: SummaryCommand,
  toggl: TogglClient,
  output: SummaryOutput = defaultSummaryOutput,
): Promise<void> {
  const { startDay, endDay, separator, format, clipboard } = cmd;

  const config = await loadConfig();
  const dateEntries = await toggl.getTimeEntriesForDays(
    config,
    startDay,
    endDay,
  );

  if (format === "json") {
    await outputSummaryText(
      formatTimeEntriesJson(dateEntries),
      clipboard,
      output,
    );
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

  await outputSummaryText(
    formatWorkTimeTable(table, separator),
    clipboard,
    output,
  );
}
