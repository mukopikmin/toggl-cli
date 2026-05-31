import { DateTime } from "ptera";
import { getProjects } from "./projects.ts";
import { getSummaryTimeEntries } from "./summary.ts";
import { getTimeEntriesForDays } from "./time_entries.ts";
import { Project, SummaryTimeEntriesResponse, TogglConfig } from "./types.ts";

export const apiEndpoint = "https://api.track.toggl.com/api/v9";
export const reportsApiEndpoint = "https://api.track.toggl.com/reports/api/v3";

export interface TogglClient {
  getProjects: (config: TogglConfig) => Promise<Project[]>;
  getSummaryTimeEntries: (
    config: TogglConfig,
    fromDay: DateTime,
    toDay: DateTime,
  ) => Promise<SummaryTimeEntriesResponse>;
  getTimeEntriesForDays: (
    config: TogglConfig,
    fromDay: DateTime,
    toDay: DateTime,
  ) => Promise<Record<string, Record<number, number>>>;
}

export const togglClient: TogglClient = {
  getProjects: getProjects,
  getSummaryTimeEntries: getSummaryTimeEntries,
  getTimeEntriesForDays: getTimeEntriesForDays,
};
