
export interface TogglConfig {
  WORKSPACE: string;
  TOKEN: string;
}

export interface Project {
  id: number;
  name: string;
  active: boolean;
}

export interface TimeEntry {
  id: number;
  project_id: number;
  start: string;
  stop: string;
  duration: number;
  description: string;
}

export async function getProjects(config: TogglConfig): Promise<Project[]> {
  const url = `https://api.track.toggl.com/api/v9/workspaces/${config.WORKSPACE}/projects`;
  const response = await fetch(url, {
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Basic ${btoa(`${config.TOKEN}:api_token`)}`,
    },
  });

  if (!response.ok) {
    console.error(`Failed to fetch projects: ${response.statusText}`);
    Deno.exit(1);
  }

  const projects: Project[] = await response.json();
  return projects.filter((p) => p.active);
}

export async function getTimeEntriesForDays(
  config: TogglConfig,
  fromDay: number,
  toDay: number,
  year: number,
  month: number
): Promise<Record<number, Record<number, number>>> {
  const fromDate = new Date(Date.UTC(year, month - 1, fromDay));
  const toDate = new Date(Date.UTC(year, month - 1, toDay));

  // Calculate start_time (fromDay - 1 day at 15:00 UTC)
  const startTimeDate = new Date(fromDate);
  startTimeDate.setUTCDate(startTimeDate.getUTCDate() - 1);
  const startTimeStr = `${startTimeDate.toISOString().split('T')[0]}T15:00:00Z`;

  // Calculate end_time (toDay at 15:00 UTC)
  const endTimeStr = `${toDate.toISOString().split('T')[0]}T15:00:00Z`;

  console.log(`Start time: ${startTimeStr}`);
  console.log(`End time: ${endTimeStr}`);

  const params = new URLSearchParams({
    start_date: startTimeStr,
    end_date: endTimeStr,
    meta: "true",
  });

  const url = `https://api.track.toggl.com/api/v9/me/time_entries?${params.toString()}`;
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

  const entries: TimeEntry[] = await response.json();

  // Aggregation
  // Group by Day -> Project -> Sum Duration
  const result: Record<number, Record<number, number>> = {};

  for (const entry of entries) {
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

  return result;
}
