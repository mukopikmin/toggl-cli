import { apiEndpoint } from "./api.ts";
import type { TimeEntry, TogglConfig } from "./types.ts";
import { DateTime } from "ptera";

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
  fromDay: DateTime,
  toDay: DateTime,
): Promise<Record<number, Record<number, number>>> {
  const startTimeStr = fromDay.toUTC().toISO();
  const endTimeStr = toDay.add({ day: 1 }).toUTC().toISO();
  const params = new URLSearchParams({
    start_date: startTimeStr,
    end_date: endTimeStr,
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
  // Group by Day -> Project -> Sum Duration
  const result: Record<number, Record<number, number>> = {};

  for (const entry of timeEntries) {
    const startDate = new Date(entry.start);
    const day = startDate.getDate(); // Local day

    console.log(`${day}:${entry.description || "(no description)"}`);

    if (!result[day]) {
      result[day] = {};
    }

    if (!result[day][entry.project_id]) {
      result[day][entry.project_id] = 0;
    }

    // Duration is in seconds, convert to minutes
    result[day][entry.project_id] += entry.duration / 60;
  }

  console.log(JSON.stringify(result, null, 2));

  return result;
}
