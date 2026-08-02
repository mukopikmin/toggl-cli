import { loadConfig } from "../config.ts";
import type { TogglClient } from "../toggl/api.ts";
import type { TimeEntry } from "../toggl/types.ts";

export type TimeEntriesFormat = "csv" | "json";

export interface TimeEntriesCommand {
  startDay: Temporal.PlainDate;
  endDay: Temporal.PlainDate;
  separator: string;
  format: TimeEntriesFormat;
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

export function prepareTimeEntries(
  entries: TimeEntry[],
  now = Date.now(),
): OutputTimeEntry[] {
  const nowSeconds = Math.floor(now / 1000);

  return entries
    .toSorted((a, b) => a.start.localeCompare(b.start) || a.id - b.id)
    .map((entry) => ({
      id: entry.id,
      description: entry.description,
      project_id: entry.project_id,
      start: entry.start,
      stop: entry.stop,
      duration_minutes: roundMinutes(
        (entry.duration < 0 ? nowSeconds + entry.duration : entry.duration) /
          60,
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

export function formatTimeEntriesCsv(
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

export function formatTimeEntriesJson(entries: OutputTimeEntry[]): string {
  return JSON.stringify(entries, null, 2);
}

export async function runTimeEntriesCommand(
  cmd: TimeEntriesCommand,
  toggl: TogglClient,
): Promise<void> {
  const config = await loadConfig();
  const entries = prepareTimeEntries(
    await toggl.getTimeEntries(config, cmd.startDay, cmd.endDay),
  );

  console.log(
    cmd.format === "json"
      ? formatTimeEntriesJson(entries)
      : formatTimeEntriesCsv(entries, cmd.separator),
  );
}
