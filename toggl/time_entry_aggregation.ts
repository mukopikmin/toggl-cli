import { formatTimeEntryDate } from "./date.ts";
import type { TimeEntry } from "./types.ts";

export type TimeEntriesByDateAndProject = Record<
  string,
  Record<number, number>
>;

/** Aggregate time-entry durations, in minutes, by local start date and project. */
export function aggregateTimeEntries(
  entries: TimeEntry[],
  timeZone: string | undefined,
  now: Temporal.Instant,
): TimeEntriesByDateAndProject {
  const result: TimeEntriesByDateAndProject = {};
  const nowEpochSeconds = Math.floor(now.epochMilliseconds / 1000);

  for (const entry of entries) {
    const date = formatTimeEntryDate(entry.start, timeZone);
    const projects = result[date] ??= {};
    projects[entry.project_id] ??= 0;

    const duration = entry.duration < 0
      ? nowEpochSeconds + entry.duration
      : entry.duration;
    projects[entry.project_id] += duration / 60;
  }

  return result;
}
