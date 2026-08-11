import { getProjects } from "./projects.ts";
import { getSummaryTimeEntries } from "./summary.ts";
import { aggregateTimeEntries } from "./time_entry_aggregation.ts";
import { getTimeEntries } from "./time_entries.ts";
import {
  SummaryTimeEntriesResponse,
  TogglConfig,
  TogglProject,
} from "./types.ts";

export const apiEndpoint = "https://api.track.toggl.com/api/v9";
export const reportsApiEndpoint = "https://api.track.toggl.com/reports/api/v3";

export async function getTimeEntriesForDays(
  config: TogglConfig,
  fromDay: Temporal.PlainDate,
  toDay: Temporal.PlainDate,
  now: Temporal.Instant = Temporal.Now.instant(),
): Promise<Record<string, Record<number, number>>> {
  const entries = await getTimeEntries(config, fromDay, toDay);
  return aggregateTimeEntries(entries, config.TIMEZONE, now);
}

export interface TogglClient {
  getProjects: (config: TogglConfig) => Promise<TogglProject[]>;
  getSummaryTimeEntries: (
    config: TogglConfig,
    fromDay: Temporal.PlainDate,
    toDay: Temporal.PlainDate,
  ) => Promise<SummaryTimeEntriesResponse>;
  getTimeEntriesForDays: (
    config: TogglConfig,
    fromDay: Temporal.PlainDate,
    toDay: Temporal.PlainDate,
  ) => Promise<Record<string, Record<number, number>>>;
}

export const togglClient: TogglClient = {
  getProjects: getProjects,
  getSummaryTimeEntries: getSummaryTimeEntries,
  getTimeEntriesForDays: getTimeEntriesForDays,
};
