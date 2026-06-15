import { parseArgs } from "node:util";
import { runSummaryCommand } from "./command/summary.ts";
import { loadConfig } from "./config.ts";
import { TogglClient, togglClient } from "./toggl/api.ts";
import type { Project } from "./toggl/types.ts";

export function formatProjectList(projects: Project[]): string {
  return projects.map((p) => p.name).join("\n");
}

export function formatProjectsJson(projects: Project[]): string {
  return JSON.stringify(projects, null, 2);
}

export type TargetMonth = {
  year: number;
  month: number;
};

export function resolveTargetMonth(
  now: TargetMonth,
  lastMonth: boolean,
): TargetMonth {
  if (!lastMonth) {
    return { year: now.year, month: now.month };
  }

  if (now.month === 1) {
    return { year: now.year - 1, month: 12 };
  }

  return { year: now.year, month: now.month - 1 };
}

const listProjects = async (toggl: TogglClient, format: "csv" | "json") => {
  const config = await loadConfig();
  const projects = await toggl.getProjects(config);
  console.log(
    format === "json"
      ? formatProjectsJson(projects)
      : formatProjectList(projects),
  );
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
      format: {
        type: "string",
        short: "f",
        default: "csv",
      },
    },
    allowPositionals: true,
  });
  const { format, lastMonth, separator } = args.values;

  if (format !== "csv" && format !== "json") {
    console.error("Error: format must be csv or json");
    Deno.exit(1);
  }

  if (args.positionals[0] === "projects") {
    await listProjects(togglClient, format);
    Deno.exit(0);
  }

  const { year: targetYear, month: targetMonth } = resolveTargetMonth(
    Temporal.Now.plainDateISO(),
    lastMonth,
  );

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

  let startDay: Temporal.PlainDate;
  let endDay: Temporal.PlainDate;
  try {
    startDay = Temporal.PlainDate.from(
      { year: targetYear, month: targetMonth, day: startDayNum },
      { overflow: "reject" },
    );
    endDay = Temporal.PlainDate.from(
      { year: targetYear, month: targetMonth, day: endDayNum },
      { overflow: "reject" },
    );
  } catch (error) {
    if (!(error instanceof RangeError)) throw error;
    console.error("Error: startDay and endDay must be valid dates");
    Deno.exit(1);
  }

  if (Temporal.PlainDate.compare(startDay, endDay) > 0) {
    console.error("Error: startDay and endDay must be valid dates");
    Deno.exit(1);
  }

  runSummaryCommand({ startDay, endDay, separator, format }, togglClient);
}
