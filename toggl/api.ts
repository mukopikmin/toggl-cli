import { DateTime } from "https://deno.land/x/ptera@v1.0.2/datetime.ts";
import { getProjects } from "./projects.ts";
import { getTimeEntriesForDays } from "./time_entries.ts";
import { Project, TogglConfig } from "./types.ts";

export const apiEndpoint = "https://api.track.toggl.com/api/v9";

export interface TogglClient {
  getProjects: (config: TogglConfig) => Promise<Project[]>;
  getTimeEntriesForDays: (
    config: TogglConfig,
    fromDay: DateTime,
    toDay: DateTime,
  ) => Promise<Record<number, Record<number, number>>>;
}

export const togglClient: TogglClient = {
  getProjects: getProjects,
  getTimeEntriesForDays: getTimeEntriesForDays,
};
