import { DateTime } from "ptera";
import { reportsApiEndpoint } from "./api.ts";
import type { SummaryTimeEntriesResponse, TogglConfig } from "./types.ts";

function formatDate(day: DateTime): string {
  const year = day.year;
  const month = String(day.month).padStart(2, "0");
  const date = String(day.day).padStart(2, "0");

  return `${year}-${month}-${date}`;
}

export async function getSummaryTimeEntries(
  config: TogglConfig,
  fromDay: DateTime,
  toDay: DateTime,
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
