import { apiEndpoint } from "./api.ts";
import type { TimeEntry, TogglConfig } from "./types.ts";
import { buildTimeEntriesDateRange } from "./date_range.ts";
import { TogglApiError } from "./error.ts";

interface TimeEntryResponse {
  id: number;
  workspace_id: number;
  project_id: number;
  task_id: number;
  billable: boolean;
  start: string;
  stop: string;
  duration: number;
  description: string;
  duronly: boolean;
  at: string;
  server_deleted_at: string;
  user_id: number;
  uid: number;
  wid: number;
  pid: number;
  client_name: string;
  project_name: string;
  project_color: string;
  project_active: boolean;
  project_billable: boolean;
  user_name: string;
  user_avatar_url: string;
}

// TODO: Fix for all locales
export async function getTimeEntries(
  config: TogglConfig,
  fromDay: Temporal.PlainDate,
  toDay: Temporal.PlainDate,
): Promise<TimeEntry[]> {
  const { startDate, endDate } = buildTimeEntriesDateRange(
    fromDay,
    toDay,
    config.TIMEZONE,
  );
  const params = new URLSearchParams({
    start_date: startDate,
    end_date: endDate,
    meta: "true",
  });

  const url = `${apiEndpoint}/me/time_entries?${params.toString()}`;
  const response = await fetch(url, {
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Basic ${btoa(`${config.TOKEN}:api_token`)}`,
    },
  });

  if (!response.ok) {
    throw new TogglApiError(
      "fetch time entries",
      response.status,
      url,
      response.statusText,
    );
  }

  const entries = await response.json() as TimeEntryResponse[];

  // Map to TimeEntry
  return entries.map((e) => ({
    id: e.id,
    project_id: e.project_id ?? e.pid,
    start: e.start,
    stop: e.stop,
    duration: e.duration,
    description: e.description,
  }));
}
