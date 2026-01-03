import { parseArgs } from "@std/cli/parse-args";
import { TogglClient, togglClient } from "./toggl/api.ts";
import { loadConfig } from "./config.ts";

const PREVIOUS_MONTH_FLAG = "-p";
const SEPARATOR = ",";

const main = async (toggl: TogglClient) => {
  const args = Deno.args;

  if (args.length < 2) {
    console.log(
      `Usage: deno run --allow-net --allow-read --allow-env main.ts <start_day> <end_day> [${PREVIOUS_MONTH_FLAG}]`,
    );
    console.log(
      "Example: deno run --allow-net --allow-read --allow-env main.ts 1 15",
    );
    console.log(
      `Example: deno run --allow-net --allow-read --allow-env main.ts 1 15 ${PREVIOUS_MONTH_FLAG}`,
    );
    Deno.exit(1);
  }

  const startDay = parseInt(args[0], 10);
  const endDay = parseInt(args[1], 10);
  const usePreviousMonth = args.includes(PREVIOUS_MONTH_FLAG);

  const now = new Date();
  let targetYear = now.getFullYear();
  let targetMonth = now.getMonth() + 1; // 1-12

  if (usePreviousMonth) {
    targetMonth -= 1;
    if (targetMonth === 0) {
      targetMonth = 12;
      targetYear -= 1;
    }
  }

  const config = await loadConfig();
  const projects = await toggl.getProjects(config);

  // Generate days array
  const days: Date[] = [];

  const current = new Date(targetYear, targetMonth - 1, startDay);
  const end = new Date(targetYear, targetMonth - 1, endDay);

  for (const d = new Date(current); d <= end; d.setDate(d.getDate() + 1)) {
    days.push(new Date(d));
  }

  const dateEntries = await toggl.getTimeEntriesForDays(
    config,
    startDay,
    endDay,
    targetYear,
    targetMonth,
  );

  console.log("--- Project list ---");
  for (const p of projects) {
    console.log(p.name);
  }
  console.log();
  console.log(
    "--- Work time table (The order of the rows follows the projects list) ---",
  );

  const header = days.map((d) => {
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, "0");
    const day = String(d.getDate()).padStart(2, "0");
    return `${y}-${m}-${day}`;
  }).join(SEPARATOR);
  console.log(header);

  for (const project of projects) {
    const row = days.map((d) => {
      const dayNum = d.getDate();
      const duration = dateEntries[dayNum]?.[project.id];
      return duration ? duration.toString() : "";
    }).join(SEPARATOR);
    console.log(row);
  }
};

if (import.meta.main) {
  main(togglClient);
}
