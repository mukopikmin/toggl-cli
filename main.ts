import { parseArgs } from "node:util";
import { TogglClient, togglClient } from "./toggl/api.ts";
import { loadConfig } from "./config.ts";
import { DateTime, datetime } from "ptera";

interface Args {
  startDay: DateTime;
  endDay: DateTime;
  lastMonth: boolean;
  separator: string;
}

interface Command {
  startDay: DateTime;
  endDay: DateTime;
  separator: string;
}

const toCommand = (args: Args): Command => {
  const { startDay, endDay, lastMonth, separator } = args;
  const now = datetime();
  let targetYear = now.year;
  let targetMonth = now.month;

  if (lastMonth) {
    targetMonth -= 1;
    if (targetMonth === 0) {
      targetMonth = 12;
      targetYear -= 1;
    }
  }

  return {
    startDay: startDay.add({ month: -1 }),
    endDay: endDay.add({ month: -1 }),
    separator,
  };
};

const main = async (args: Args, toggl: TogglClient) => {
  const { startDay, endDay, separator } = toCommand(args);

  const config = await loadConfig();
  const projects = await toggl.getProjects(config);

  // Generate days array
  const days: DateTime[] = [];

  for (
    let d = startDay;
    !d.isAfter(endDay);
    d = d.add({ day: 1 })
  ) {
    days.push(d);
  }

  const dateEntries = await toggl.getTimeEntriesForDays(
    config,
    startDay,
    endDay,
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
    const y = d.year;
    const m = String(d.month).padStart(2, "0");
    const day = String(d.day).padStart(2, "0");
    return `${y}-${m}-${day}`;
  }).join(separator);
  console.log(header);

  for (const project of projects) {
    const row = days.map((d) => {
      const dayNum = d.day;
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
  const now = datetime();
  const startDay = datetime({
    year: now.year,
    month: now.month,
    day: Number(args.positionals[0]),
  });
  const endDay = datetime({
    year: now.year,
    month: now.month,
    day: Number(args.positionals[1]),
  });
  const { lastMonth, separator } = args.values;

  if (!startDay.isValid() || !endDay.isValid() || startDay.isAfter(endDay)) {
    console.error("Error: startDay and endDay must be valid dates");
    Deno.exit(1);
  }

  main({ startDay, endDay, lastMonth, separator }, togglClient);
}
