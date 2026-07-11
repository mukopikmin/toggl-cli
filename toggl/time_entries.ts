import { apiEndpoint } from "./api.ts";
import type { TimeEntry, TogglConfig } from "./types.ts";
import { buildTimeEntriesDateRange } from "./date_range.ts";
import { formatTimeEntryDate } from "./date.ts";

interface TimeEntryResponse {
  id: number;
  workspace_id: number;
  project_id: number | null;
  task_id: number;
  billable: boolean;
  start: string;
  stop: string | null;
  duration: number;
  description: string;
  duronly: boolean;
  at: string;
  server_deleted_at: string;
  user_id: number;
  uid: number;
  wid: number;
  pid: number | null;
  client_name: string;
  project_name: string;
  project_color: string;
  project_active: boolean;
  project_billable: boolean;
  user_name: string;
  user_avatar_url: string;
}

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
    console.error(`Failed to fetch time entries: ${response.statusText}`);
    Deno.exit(1);
  }

  const entries = await response.json() as TimeEntryResponse[];

  // Map to TimeEntry
  return entries.map((e) => ({
    id: e.id,
    project_id: e.project_id ?? e.pid ?? null,
    start: e.start,
    stop: e.stop ?? null,
    duration: e.duration,
    description: e.description,
  }));
}

export function aggregateTimeEntriesForDays(
  timeEntries: TimeEntry[],
  timezone?: string,
  now = Date.now(),
): Record<string, Record<number, number>> {
  // Aggregation
  // Group by YYYY-MM-DD -> Project -> Sum Duration
  const result: Record<string, Record<number, number>> = {};

  for (const entry of timeEntries) {
    if (entry.project_id === null) continue;
    const dateStr = formatTimeEntryDate(entry.start, timezone);

    if (!result[dateStr]) {
      result[dateStr] = {};
    }

    if (!result[dateStr][entry.project_id]) {
      result[dateStr][entry.project_id] = 0;
    }

    // Duration is in seconds, convert to minutes
    let dur = entry.duration;
    if (dur < 0) {
      dur = Math.floor(now / 1000) + dur;
    }
    result[dateStr][entry.project_id] += dur / 60;
  }

  return result;
}

// TODO: Fix for all locales
export async function getTimeEntriesForDays(
  config: TogglConfig,
  fromDay: Temporal.PlainDate,
  toDay: Temporal.PlainDate,
): Promise<Record<string, Record<number, number>>> {
  return aggregateTimeEntriesForDays(
    await getTimeEntries(config, fromDay, toDay),
    config.TIMEZONE,
  );
}
