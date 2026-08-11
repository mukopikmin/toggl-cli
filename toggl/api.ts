import { getProjects } from "./projects.ts";
import { getSummaryTimeEntries } from "./summary.ts";
import { getTimeEntries } from "./time_entries.ts";
import type { TimeEntry } from "../model/time_entry.ts";
import { summarizeTimeEntries } from "../model/time_entry_summary.ts";
import type { Project, ProjectDisplaySettings } from "../model/project.ts";
import type {
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
  return summarizeTimeEntries(entries, config.TIMEZONE, now);
}

export interface TogglClient {
  getProjects: (
    config: TogglConfig,
    settingsByProjectId?: Record<number, ProjectDisplaySettings>,
  ) => Promise<Project[]>;
  getSummaryTimeEntries: (
    config: TogglConfig,
    fromDay: Temporal.PlainDate,
    toDay: Temporal.PlainDate,
  ) => Promise<SummaryTimeEntriesResponse>;
  getTimeEntries: (
    config: TogglConfig,
    fromDay: Temporal.PlainDate,
    toDay: Temporal.PlainDate,
  ) => Promise<TimeEntry[]>;
  getTimeEntriesForDays: (
    config: TogglConfig,
    fromDay: Temporal.PlainDate,
    toDay: Temporal.PlainDate,
  ) => Promise<Record<string, Record<number, number>>>;
}

export const togglClient: TogglClient = {
  getProjects: getProjects,
  getSummaryTimeEntries: getSummaryTimeEntries,
  getTimeEntries: getTimeEntries,
  getTimeEntriesForDays: getTimeEntriesForDays,
};
