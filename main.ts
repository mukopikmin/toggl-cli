import { parseArgs } from "node:util";
import { TogglClient, togglClient } from "./toggl/api.ts";
import { loadConfig } from "./config.ts";
import { DateTime, datetime } from "ptera";

interface Command {
  startDay: DateTime;
  endDay: DateTime;
  separator: string;
}

const main = async (cmd: Command, toggl: TogglClient) => {
  const { startDay, endDay, separator } = cmd;

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
  const { lastMonth, separator } = args.values;

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

  const posLen = args.positionals.length;
  if (posLen < 2) {
    console.error("Error: Please specify start and end day");
    Deno.exit(1);
  }

  const startDayNum = Number(args.positionals[posLen - 2]);
  const endDayNum = Number(args.positionals[posLen - 1]);

  if (isNaN(startDayNum) || isNaN(endDayNum)) {
    console.error("Error: Start and end day must be valid numbers");
    Deno.exit(1);
  }

  const startDay = datetime({
    year: targetYear,
    month: targetMonth,
    day: startDayNum,
  });
  const endDay = datetime({
    year: targetYear,
    month: targetMonth,
    day: endDayNum,
  });

  if (!startDay.isValid() || !endDay.isValid() || startDay.isAfter(endDay)) {
    console.error("Error: startDay and endDay must be valid dates");
    Deno.exit(1);
  }

  main({ startDay, endDay, separator }, togglClient);
}
