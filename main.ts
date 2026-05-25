import { parseArgs } from "node:util";
import { TogglClient, togglClient } from "./toggl/api.ts";
import { loadConfig } from "./config.ts";
import { DateTime, datetime } from "ptera";
import type { Project } from "./toggl/types.ts";

interface Command {
  startDay: DateTime;
  endDay: DateTime;
  separator: string;
}

export interface WorkTimeTable {
  projectNames: string[];
  headers: string[];
  rows: string[][];
}

function formatDay(day: DateTime): string {
  const y = day.year;
  const m = String(day.month).padStart(2, "0");
  const d = String(day.day).padStart(2, "0");

  return `${y}-${m}-${d}`;
}

function daysBetween(startDay: DateTime, endDay: DateTime): DateTime[] {
  const days: DateTime[] = [];

  for (
    let d = startDay;
    !d.isAfter(endDay);
    d = d.add({ day: 1 })
  ) {
    days.push(d);
  }

  return days;
}

export function buildWorkTimeTable(
  projects: Project[],
  dateEntries: Record<string, Record<number, number>>,
  startDay: DateTime,
  endDay: DateTime,
): WorkTimeTable {
  const days = daysBetween(startDay, endDay);
  const headers = days.map(formatDay);
  const rows = projects.map((project) =>
    headers.map((date) => {
      const duration = dateEntries[date]?.[project.id];
      if (duration) {
        return (Math.round(duration * 100) / 100).toString();
      }
      return " ";
    })
  );

  return {
    projectNames: projects.map((project) => project.name),
    headers,
    rows,
  };
}

export function outputWorkTimeTable(
  table: WorkTimeTable,
  separator: string,
): void {
  console.log("--- Project list ---");
  for (const projectName of table.projectNames) {
    console.log(projectName);
  }
  console.log();
  console.log(
    "--- Work time table (The order of the rows follows the projects list) ---",
  );
  console.log(table.headers.join(separator));

  for (const row of table.rows) {
    console.log(row.join(separator));
  }
}

const main = async (cmd: Command, toggl: TogglClient) => {
  const { startDay, endDay, separator } = cmd;

  const config = await loadConfig();
  const projects = await toggl.getProjects(config);
  const dateEntries = await toggl.getTimeEntriesForDays(
    config,
    startDay,
    endDay,
  );
  const table = buildWorkTimeTable(projects, dateEntries, startDay, endDay);

  outputWorkTimeTable(table, separator);
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
    hour: 0,
    minute: 0,
    second: 0,
    millisecond: 0,
  });
  const endDay = datetime({
    year: targetYear,
    month: targetMonth,
    day: endDayNum,
    hour: 0,
    minute: 0,
    second: 0,
    millisecond: 0,
  });

  if (!startDay.isValid() || !endDay.isValid() || startDay.isAfter(endDay)) {
    console.error("Error: startDay and endDay must be valid dates");
    Deno.exit(1);
  }

  main({ startDay, endDay, separator }, togglClient);
}
