import { parseArgs } from "node:util";
import { type DateTime, datetime } from "ptera";
import type { ProjectsFormat } from "./command/projects.ts";
import type { SummaryFormat } from "./command/summary.ts";

export function createHelpText(): string {
  return `Usage:
  toggl summary <start-date> <end-date> [options]
  toggl projects [options]
  toggl projects sync
  toggl config [options]
  toggl init

Commands:
  init      Create the configuration file
  projects  List projects
  config    Show configuration values
  summary   Summarize time entries for a range of days

Options:
  -s, --separator <text> Set the output delimiter (default: tab)
  -f, --format <format>  Set the output format: csv or json (default: csv)
  -h, --help             Show this help
      --no-project       Omit the project column from CSV output
      --version          Show the version`;
}

export const HELP_TEXT = createHelpText();

export type CliCommand =
  | { name: "help" }
  | { name: "version" }
  | { name: "init" }
  | { name: "projects"; format: ProjectsFormat }
  | { name: "config"; format: ProjectsFormat }
  | { name: "projects-sync" }
  | {
    name: "summary";
    startDay: Temporal.PlainDate;
    endDay: Temporal.PlainDate;
    separator: string;
    format: SummaryFormat;
    noProject: boolean;
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

function parseIsoDate(value: string): Temporal.PlainDate {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    throw new CliUsageError("start and end date must use YYYY-MM-DD");
  }

  try {
    return Temporal.PlainDate.from(value);
  } catch {
    throw new CliUsageError("start and end date must be valid dates");
  }
}

function parseSummaryArgs(args: string[]): CliCommand {
  const parsed = parseArgs({
    args,
    options: {
      separator: { type: "string", short: "s", default: "\t" },
      format: { type: "string", short: "f", default: "csv" },
      "no-project": { type: "boolean", default: false },
    },
    allowPositionals: true,
    strict: true,
  });

  if (parsed.positionals.length !== 2) {
    throw new CliUsageError("summary requires start and end date");
  }

  const startDay = parseIsoDate(parsed.positionals[0]);
  const endDay = parseIsoDate(parsed.positionals[1]);

  if (Temporal.PlainDate.compare(startDay, endDay) > 0) {
    throw new CliUsageError("start date must not be after end date");
  }

  return {
    name: "summary",
    startDay,
    endDay,
    separator: parsed.values.separator ?? "\t",
    format: parseFormat(parsed.values.format),
    noProject: parsed.values["no-project"] ?? false,
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
      case "--help":
      case "-h":
        if (commandArgs.length > 0) {
          throw new CliUsageError("--help does not accept arguments");
        }
        return { name: "help" };
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
        return parseSummaryArgs(commandArgs);
      default:
        throw new CliUsageError(`unknown command: ${command}`);
    }
  } catch (error) {
    if (error instanceof CliUsageError) throw error;
    if (error instanceof Error) throw new CliUsageError(error.message);
    throw error;
  }
}
