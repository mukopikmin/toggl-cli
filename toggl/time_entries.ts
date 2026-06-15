import { apiEndpoint } from "./api.ts";
import type { TimeEntry, TogglConfig } from "./types.ts";
import { buildTimeEntriesDateRange } from "./date_range.ts";
import { formatTimeEntryDate } from "./date.ts";

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
export async function getTimeEntriesForDays(
  config: TogglConfig,
  fromDay: Temporal.PlainDate,
  toDay: Temporal.PlainDate,
): Promise<Record<string, Record<number, number>>> {
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
    console.error(`Failed to fetch time entries: ${response.statusText}`);
    Deno.exit(1);
  }

  const entries = await response.json() as TimeEntryResponse[];

  // Map to TimeEntry
  const timeEntries: TimeEntry[] = entries.map((e) => ({
    id: e.id,
    project_id: e.project_id ?? e.pid,
    start: e.start,
    stop: e.stop,
    duration: e.duration,
    description: e.description,
  }));

  // Aggregation
  // Group by YYYY-MM-DD -> Project -> Sum Duration
  const result: Record<string, Record<number, number>> = {};

  for (const entry of timeEntries) {
    const dateStr = formatTimeEntryDate(entry.start, config.TIMEZONE);

    if (!result[dateStr]) {
      result[dateStr] = {};
    }

    if (!result[dateStr][entry.project_id]) {
      result[dateStr][entry.project_id] = 0;
    }

    // Duration is in seconds, convert to minutes
    let dur = entry.duration;
    if (dur < 0) {
      dur = Math.floor(Date.now() / 1000) + dur;
    }
    result[dateStr][entry.project_id] += dur / 60;
  }

  return result;
}
