import { formatTimeEntryDate } from "./date.ts";
import type { TimeEntry } from "./time_entry.ts";

export type TimeEntrySummary = Record<
  string,
  Record<number, number>
>;

/** Aggregate time-entry durations, in minutes, by local start date and project. */
export function summarizeTimeEntries(
  entries: TimeEntry[],
  timeZone: string | undefined,
  now: Temporal.Instant,
): TimeEntrySummary {
  const result: TimeEntrySummary = {};
  const nowEpochSeconds = Math.floor(now.epochMilliseconds / 1000);

  for (const entry of entries) {
    const date = formatTimeEntryDate(entry.start, timeZone);
    const projects = result[date] ??= {};
    projects[entry.projectId] ??= 0;

    const duration = entry.durationSeconds < 0
      ? nowEpochSeconds + entry.durationSeconds
      : entry.durationSeconds;
    projects[entry.projectId] += duration / 60;
  }

  return result;
}
