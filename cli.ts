import { parseArgs } from "node:util";
import { type DateTime, datetime } from "ptera";
import type { ProjectFormat } from "./command/project.ts";
import type { SummaryFormat } from "./command/summary.ts";
import type { TimeEntryListFormat } from "./command/time_entry.ts";

export function createHelpText(): string {
  return `Usage:
  toggl summary <start-date> <end-date> [options]
  toggl summary --days <days> [options]
  toggl time-entry list <start-day> <end-day> [options]
  toggl project list [options]
  toggl project reorder
  toggl project sync
  toggl config [options]
  toggl init
  toggl update [--channel stable|nightly]

Commands:
  init        Create the configuration file
  project     List, reorder, and sync projects
  time-entry  List individual time entries for a range of days
  config      Show configuration values
  summary     Summarize time entries for a range of days
  update      Update the installed Toggl CLI binary

Options:
  -s, --separator <text> Set the output delimiter (default: tab)
  -f, --format <format>  Set the output format: csv or json (default: csv)
  -d, --days <days>      Aggregate from this many days ago through today
      --clipboard        Copy the output to the clipboard as well as stdout
  -h, --help             Show this help
      --no-project       Omit the project column (summary CSV only)
      --no-date          Omit the date header row from CSV output
      --version          Show the version`;
}

export const HELP_TEXT = createHelpText();

export type CliCommand =
  | { name: "help" }
  | { name: "version" }
  | { name: "init" }
  | { name: "update"; channel?: "stable" | "nightly" }
  | { name: "project-list"; format: ProjectFormat }
  | { name: "project-reorder" }
  | { name: "config"; format: ProjectFormat }
  | { name: "project-sync" }
  | {
    name: "summary";
    separator: string;
    format: SummaryFormat;
    noProject: boolean;
    noDate: boolean;
    clipboard: boolean;
  }
    & (
      | { startDay: Temporal.PlainDate; endDay: Temporal.PlainDate }
      | { days: number }
    )
  | {
    name: "time-entry-list";
    startDay: Temporal.PlainDate;
    endDay: Temporal.PlainDate;
    separator: string;
    format: TimeEntryListFormat;
  };

export class CliUsageError extends Error {}

function parseFormat(value: string | undefined): ProjectFormat {
  const format = value ?? "csv";
  if (format !== "csv" && format !== "json") {
    throw new CliUsageError("format must be csv or json");
  }
  return format;
}

function parseProjectListArgs(args: string[]): CliCommand {
  const parsed = parseArgs({
    args,
    options: {
      format: { type: "string", short: "f", default: "csv" },
    },
    allowPositionals: true,
    strict: true,
  });

  if (parsed.positionals.length > 0) {
    throw new CliUsageError(
      "project list does not accept positional arguments",
    );
  }

  return { name: "project-list", format: parseFormat(parsed.values.format) };
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

function parseUpdateArgs(args: string[]): CliCommand {
  const parsed = parseArgs({
    args,
    options: { channel: { type: "string" } },
    allowPositionals: true,
    strict: true,
  });
  if (parsed.positionals.length > 0) {
    throw new CliUsageError("update does not accept positional arguments");
  }
  const channel = parsed.values.channel;
  if (channel !== undefined && channel !== "stable" && channel !== "nightly") {
    throw new CliUsageError("channel must be stable or nightly");
  }
  return { name: "update", channel };
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
      days: { type: "string", short: "d" },
      "no-project": { type: "boolean", default: false },
      "no-date": { type: "boolean", default: false },
      clipboard: { type: "boolean", default: false },
    },
    allowPositionals: true,
    strict: true,
  });

  const common = {
    name: "summary",
    separator: parsed.values.separator ?? "\t",
    format: parseFormat(parsed.values.format),
    noProject: parsed.values["no-project"] ?? false,
    noDate: parsed.values["no-date"] ?? false,
    clipboard: parsed.values.clipboard ?? false,
  } as const;

  if (parsed.values.days !== undefined) {
    if (parsed.positionals.length > 0) {
      throw new CliUsageError(
        "summary accepts either start and end date or --days, not both",
      );
    }

    if (!/^\d+$/.test(parsed.values.days)) {
      throw new CliUsageError("days must be a non-negative integer");
    }
    const days = Number(parsed.values.days);
    if (!Number.isSafeInteger(days)) {
      throw new CliUsageError("days must be a non-negative integer");
    }

    return { ...common, days };
  }

  if (parsed.positionals.length !== 2) {
    throw new CliUsageError("summary requires start and end date or --days");
  }

  const startDay = parseIsoDate(parsed.positionals[0]);
  const endDay = parseIsoDate(parsed.positionals[1]);

  if (Temporal.PlainDate.compare(startDay, endDay) > 0) {
    throw new CliUsageError("start date must not be after end date");
  }

  return { ...common, startDay, endDay };
}

function parseTimeEntryListArgs(args: string[], now: DateTime): CliCommand {
  const parsed = parseArgs({
    args,
    options: {
      separator: { type: "string", short: "s", default: "\t" },
      format: { type: "string", short: "f", default: "csv" },
    },
    allowPositionals: true,
    strict: true,
  });

  if (parsed.positionals.length !== 2) {
    throw new CliUsageError("time-entry list requires start and end day");
  }

  if (!parsed.positionals.every((value) => /^\d+$/.test(value))) {
    throw new CliUsageError("start and end day must be valid integers");
  }

  let startDay: Temporal.PlainDate;
  let endDay: Temporal.PlainDate;
  try {
    startDay = Temporal.PlainDate.from(
      {
        year: now.year,
        month: now.month,
        day: Number(parsed.positionals[0]),
      },
      { overflow: "reject" },
    );
    endDay = Temporal.PlainDate.from(
      {
        year: now.year,
        month: now.month,
        day: Number(parsed.positionals[1]),
      },
      { overflow: "reject" },
    );
  } catch {
    throw new CliUsageError("start and end day must be valid dates");
  }
  if (Temporal.PlainDate.compare(startDay, endDay) > 0) {
    throw new CliUsageError("start day must not be after end day");
  }

  const separator = parsed.values.separator ?? "\t";
  if (separator.length === 0) {
    throw new CliUsageError("separator must not be empty");
  }

  return {
    name: "time-entry-list",
    startDay,
    endDay,
    separator,
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
      case "project":
        switch (commandArgs[0]) {
          case "list":
            return parseProjectListArgs(commandArgs.slice(1));
          case "sync":
            if (commandArgs.length > 1) {
              throw new CliUsageError("project sync does not accept arguments");
            }
            return { name: "project-sync" };
          case "reorder":
            if (commandArgs.length > 1) {
              throw new CliUsageError(
                "project reorder does not accept arguments",
              );
            }
            return { name: "project-reorder" };
          case undefined:
            throw new CliUsageError(
              "project requires a subcommand: list, reorder, or sync",
            );
          default:
            throw new CliUsageError(
              `unknown project subcommand: ${commandArgs[0]}`,
            );
        }
      case "config":
        return parseConfigArgs(commandArgs);
      case "summary":
        return parseSummaryArgs(commandArgs);
      case "time-entry":
        switch (commandArgs[0]) {
          case "list":
            return parseTimeEntryListArgs(commandArgs.slice(1), now);
          case undefined:
            throw new CliUsageError(
              "time-entry requires a subcommand: list",
            );
          default:
            throw new CliUsageError(
              `unknown time-entry subcommand: ${commandArgs[0]}`,
            );
        }
      case "update":
        return parseUpdateArgs(commandArgs);
      default:
        throw new CliUsageError(`unknown command: ${command}`);
    }
  } catch (error) {
    if (error instanceof CliUsageError) throw error;
    if (error instanceof Error) throw new CliUsageError(error.message);
    throw error;
  }
}
