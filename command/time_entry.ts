import { loadConfig } from "../config.ts";
import type { TogglClient } from "../toggl/api.ts";
import type { TimeEntry } from "../model/time_entry.ts";
import type { OutputFormat } from "./output_format.ts";
import { formatTable } from "./table.ts";

export type TimeEntryListFormat = OutputFormat;

export interface TimeEntryListCommand {
  startDay: Temporal.PlainDate;
  endDay: Temporal.PlainDate;
  separator: string;
  format: TimeEntryListFormat;
}

export interface OutputTimeEntry {
  id: number;
  description: string;
  project_id: number | null;
  start: string;
  stop: string | null;
  duration_minutes: number;
}

function roundMinutes(value: number): number {
  return Math.round(value * 100) / 100;
}

export function prepareTimeEntryList(
  entries: TimeEntry[],
  now = Date.now(),
): OutputTimeEntry[] {
  const nowSeconds = Math.floor(now / 1000);

  return entries
    .toSorted((a, b) => a.start.localeCompare(b.start) || a.id - b.id)
    .map((entry) => ({
      id: entry.id,
      description: entry.description,
      project_id: entry.projectId,
      start: entry.start,
      stop: entry.stop,
      duration_minutes: roundMinutes(
        (entry.durationSeconds < 0
          ? nowSeconds + entry.durationSeconds
          : entry.durationSeconds) / 60,
      ),
    }));
}

function escapeCsvField(value: string, separator: string): string {
  if (
    value.includes(separator) || value.includes('"') ||
    value.includes("\n") || value.includes("\r")
  ) {
    return `"${value.replaceAll('"', '""')}"`;
  }
  return value;
}

export function formatTimeEntryListCsv(
  entries: OutputTimeEntry[],
  separator: string,
): string {
  const rows = [
    ["id", "description", "project_id", "start", "stop", "duration_minutes"],
    ...entries.map((entry) => [
      String(entry.id),
      entry.description,
      entry.project_id === null ? "" : String(entry.project_id),
      entry.start,
      entry.stop ?? "",
      String(entry.duration_minutes),
    ]),
  ];

  return rows.map((row) =>
    row.map((value) => escapeCsvField(value, separator)).join(separator)
  ).join("\n");
}

export function formatTimeEntryListJson(entries: OutputTimeEntry[]): string {
  return JSON.stringify(entries, null, 2);
}

export function formatTimeEntryListTable(entries: OutputTimeEntry[]): string {
  const headers = [
    "id",
    "description",
    "project_id",
    "start",
    "stop",
    "duration_minutes",
  ];
  return formatTable(
    headers,
    entries.map((entry) => [
      String(entry.id),
      entry.description,
      entry.project_id === null ? "" : String(entry.project_id),
      entry.start,
      entry.stop ?? "",
      String(entry.duration_minutes),
    ]),
  );
}

export async function runTimeEntryListCommand(
  cmd: TimeEntryListCommand,
  toggl: TogglClient,
): Promise<void> {
  const config = await loadConfig();
  const entries = prepareTimeEntryList(
    await toggl.getTimeEntries(config, cmd.startDay, cmd.endDay),
  );

  console.log(
    cmd.format === "json"
      ? formatTimeEntryListJson(entries)
      : cmd.format === "table"
      ? formatTimeEntryListTable(entries)
      : formatTimeEntryListCsv(entries, cmd.separator),
  );
}
