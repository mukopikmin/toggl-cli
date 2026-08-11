import { getProjects } from "./projects.ts";
import { getSummaryTimeEntries } from "./summary.ts";
import { getTimeEntries } from "./time_entries.ts";
import type {
  SummaryTimeEntriesResponse,
  TimeEntry,
  TogglConfig,
  TogglProject,
} from "./types.ts";

export const apiEndpoint = "https://api.track.toggl.com/api/v9";
export const reportsApiEndpoint = "https://api.track.toggl.com/reports/api/v3";

export interface TogglClient {
  getProjects: (config: TogglConfig) => Promise<TogglProject[]>;
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
}

export const togglClient: TogglClient = {
  getProjects: getProjects,
  getSummaryTimeEntries: getSummaryTimeEntries,
  getTimeEntries: getTimeEntries,
};
