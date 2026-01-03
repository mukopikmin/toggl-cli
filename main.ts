import { parseArgs } from "node:util";
import { TogglClient, togglClient } from "./toggl/api.ts";
import { loadConfig } from "./config.ts";

interface Args {
  startDay: number;
  endDay: number;
  lastMonth: boolean;
  separator: string;
}

const main = async (
  { startDay, endDay, lastMonth, separator }: Args,
  toggl: TogglClient,
) => {
  const now = new Date();
  let targetYear = now.getFullYear();
  let targetMonth = now.getMonth() + 1; // 1-12

  if (lastMonth) {
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
  }).join(separator);
  console.log(header);

  for (const project of projects) {
    const row = days.map((d) => {
      const dayNum = d.getDate();
      const duration = dateEntries[dayNum]?.[project.id];
      return duration ? duration.toString() : "";
    }).join(separator);
    console.log(row);
  }
};

if (import.meta.main) {
  const args = parseArgs({
    options: {
      lastMonth: {
        type: "boolean",
        short: "l",
        default: false,
      },
      separator: {
        type: "string",
        short: "s",
        default: "\t",
      },
    },
    allowPositionals: true,
  });
  const startDay = Number(args.positionals[0]);
  const endDay = Number(args.positionals[1]);
  const { lastMonth, separator } = args.values;

  if (Number.isNaN(startDay) || Number.isNaN(endDay)) {
    console.error("Error: startDay and endDay must be numbers");
    Deno.exit(1);
  }

  main({ startDay, endDay, lastMonth, separator }, togglClient);
}
