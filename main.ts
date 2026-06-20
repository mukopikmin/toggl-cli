import { parseArgs } from "node:util";
import { datetime } from "ptera";
import { runInitCommand } from "./command/init.ts";
import {
  runProjectsCommand,
  runProjectsSyncCommand,
} from "./command/projects.ts";
import { runSummaryCommand } from "./command/summary.ts";
import { togglClient } from "./toggl/api.ts";
import { version } from "./version.ts";

export type TargetMonth = {
  year: number;
  month: number;
};

export function createHelpText(): string {
  return `Usage:
  toggl summary <start-day> <end-day> [options]
  toggl <start-day> <end-day> [options]
  toggl projects [options]
  toggl projects sync
  toggl init

Options:
  -l, --lastMonth        Aggregate the previous month
  -s, --separator <text> Set the output delimiter (default: tab)
  -f, --format <format>  Set the output format: csv or json (default: csv)
  -h, --help             Show this help
      --version          Show the version`;
}

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
      version: {
        type: "boolean",
        default: false,
      },
      help: {
        type: "boolean",
        short: "h",
        default: false,
      },
    },
    allowPositionals: true,
  });
  const { format, help, lastMonth, separator, version: showVersion } =
    args.values;

  if (help) {
    console.log(createHelpText());
    Deno.exit(0);
  }

  if (showVersion) {
    console.log(version);
    Deno.exit(0);
  }

  if (args.positionals[0] === "init") {
    await runInitCommand();
    Deno.exit(0);
  }

  if (format !== "csv" && format !== "json") {
    console.error("Error: format must be csv or json");
    Deno.exit(1);
  }

  if (args.positionals[0] === "projects") {
    if (args.positionals[1] === "sync") {
      await runProjectsSyncCommand(togglClient);
    } else {
      await runProjectsCommand({ format }, togglClient);
    }
    Deno.exit(0);
  }

  const command = args.positionals[0];
  const summaryPositionals = command === "summary"
    ? args.positionals.slice(1)
    : args.positionals;

  if (command && command !== "summary" && isNaN(Number(command))) {
    console.error(`Error: Unknown command: ${command}`);
    Deno.exit(1);
  }

  const { year: targetYear, month: targetMonth } = resolveTargetMonth(
    datetime(),
    lastMonth,
  );

  if (summaryPositionals.length < 2) {
    console.error("Error: Please specify start and end day");
    Deno.exit(1);
  }

  if (summaryPositionals.length > 2) {
    console.error("Error: Too many arguments for summary");
    Deno.exit(1);
  }

  const startDayNum = Number(summaryPositionals[0]);
  const endDayNum = Number(summaryPositionals[1]);

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

  runSummaryCommand({ startDay, endDay, separator, format }, togglClient);
}
