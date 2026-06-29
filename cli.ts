import { parseArgs } from "node:util";
import { type DateTime, datetime } from "ptera";
import type { ProjectsFormat } from "./command/projects.ts";
import type { SummaryFormat } from "./command/summary.ts";

export const HELP_TEXT = `Usage:
  toggl --version
  toggl init
  toggl projects [sync] [--format csv|json]
  toggl config [--format csv|json]
  toggl summary [--lastMonth] [--separator VALUE] [--format csv|json] START_DAY END_DAY

Commands:
  init      Create the configuration file
  projects  List projects
  config    Show configuration values
  summary   Summarize time entries for a range of days

Summary options:
  -l, --lastMonth       Use the previous month
  -s, --separator VALUE Set the CSV separator (default: tab)
  -f, --format FORMAT   Output format: csv or json (default: csv)`;

export type CliCommand =
  | { name: "help" }
  | { name: "version" }
  | { name: "init" }
  | { name: "projects"; format: ProjectsFormat }
  | { name: "config"; format: ProjectsFormat }
  | { name: "projects-sync" }
  | {
    name: "summary";
    startDay: DateTime;
    endDay: DateTime;
    separator: string;
    format: SummaryFormat;
  };

export class CliUsageError extends Error {}

function parseFormat(value: string | undefined): ProjectsFormat {
  const format = value ?? "csv";
  if (format !== "csv" && format !== "json") {
    throw new CliUsageError("format must be csv or json");
  }
  return format;
}

function parseProjectsArgs(args: string[]): CliCommand {
  const parsed = parseArgs({
    args,
    options: {
      format: { type: "string", short: "f", default: "csv" },
    },
    allowPositionals: true,
    strict: true,
  });

  if (parsed.positionals.length > 0) {
    throw new CliUsageError("projects does not accept positional arguments");
  }

  return { name: "projects", format: parseFormat(parsed.values.format) };
}

function parseConfigArgs(args: string[]): CliCommand {
  const parsed = parseArgs({
    args,
    options: {
      format: { type: "string", short: "f", default: "csv" },
    },
    allowPositionals: true,
    strict: true,
  });

  if (parsed.positionals.length > 0) {
    throw new CliUsageError("config does not accept positional arguments");
  }

  return { name: "config", format: parseFormat(parsed.values.format) };
}

function parseSummaryArgs(args: string[], now: DateTime): CliCommand {
  const parsed = parseArgs({
    args,
    options: {
      lastMonth: { type: "boolean", short: "l", default: false },
      separator: { type: "string", short: "s", default: "\t" },
      format: { type: "string", short: "f", default: "csv" },
    },
    allowPositionals: true,
    strict: true,
  });

  if (parsed.positionals.length !== 2) {
    throw new CliUsageError("summary requires start and end day");
  }

  const startDayNum = Number(parsed.positionals[0]);
  const endDayNum = Number(parsed.positionals[1]);
  if (isNaN(startDayNum) || isNaN(endDayNum)) {
    throw new CliUsageError("start and end day must be valid numbers");
  }

  let targetYear = now.year;
  let targetMonth = now.month;
  if (parsed.values.lastMonth) {
    targetMonth -= 1;
    if (targetMonth === 0) {
      targetMonth = 12;
      targetYear -= 1;
    }
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
    throw new CliUsageError("start and end day must be valid dates");
  }

  return {
    name: "summary",
    startDay,
    endDay,
    separator: parsed.values.separator ?? "\t",
    format: parseFormat(parsed.values.format),
  };
}

export function parseCliArgs(
  args: string[],
  now: DateTime = datetime(),
): CliCommand {
  const [command, ...commandArgs] = args;
  if (command === undefined) return { name: "help" };

  try {
    switch (command) {
      case "--version":
        if (commandArgs.length > 0) {
          throw new CliUsageError("--version does not accept arguments");
        }
        return { name: "version" };
      case "init":
        if (commandArgs.length > 0) {
          throw new CliUsageError("init does not accept arguments");
        }
        return { name: "init" };
      case "projects":
        if (commandArgs[0] === "sync") {
          if (commandArgs.length > 1) {
            throw new CliUsageError("projects sync does not accept arguments");
          }
          return { name: "projects-sync" };
        }
        return parseProjectsArgs(commandArgs);
      case "config":
        return parseConfigArgs(commandArgs);
      case "summary":
        return parseSummaryArgs(commandArgs, now);
      default:
        throw new CliUsageError(`unknown command: ${command}`);
    }
  } catch (error) {
    if (error instanceof CliUsageError) throw error;
    if (error instanceof Error) throw new CliUsageError(error.message);
    throw error;
  }
}
