import { reportsApiEndpoint } from "./api.ts";
import type { SummaryTimeEntriesResponse, TogglConfig } from "./types.ts";

function formatDate(day: Temporal.PlainDate): string {
  return day.toString();
}

export async function getSummaryTimeEntries(
  config: TogglConfig,
  fromDay: Temporal.PlainDate,
  toDay: Temporal.PlainDate,
): Promise<SummaryTimeEntriesResponse> {
  const url =
    `${reportsApiEndpoint}/workspace/${config.WORKSPACE}/summary/time_entries`;
  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Basic ${btoa(`${config.TOKEN}:api_token`)}`,
    },
    body: JSON.stringify({
      start_date: formatDate(fromDay),
      end_date: formatDate(toDay),
      grouping: "projects",
    }),
  });

  if (!response.ok) {
    console.error(
      `Failed to fetch summary time entries: ${response.statusText}`,
    );
    Deno.exit(1);
  }

  return await response.json() as SummaryTimeEntriesResponse;
}
