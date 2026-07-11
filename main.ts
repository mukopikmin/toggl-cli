import { CliUsageError, HELP_TEXT, parseCliArgs } from "./cli.ts";
import { runConfigCommand } from "./command/config.ts";
import { runInitCommand } from "./command/init.ts";
import {
  runProjectsCommand,
  runProjectsSyncCommand,
} from "./command/projects.ts";
import { runSummaryCommand } from "./command/summary.ts";
import { runTimeEntriesCommand } from "./command/time_entries.ts";
import { togglClient } from "./toggl/api.ts";
import { version } from "./version.ts";

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

export async function main(args: string[]): Promise<number> {
  let command;
  try {
    command = parseCliArgs(args);
  } catch (error) {
    if (!(error instanceof CliUsageError)) throw error;
    console.error(`Error: ${error.message}\n\n${HELP_TEXT}`);
    return 1;
  }

  switch (command.name) {
    case "help":
      console.log(HELP_TEXT);
      return 0;
    case "version":
      console.log(version);
      return 0;
    case "init":
      await runInitCommand();
      return 0;
    case "projects":
      await runProjectsCommand({ format: command.format }, togglClient);
      return 0;
    case "config":
      await runConfigCommand(command.format);
      return 0;
    case "projects-sync":
      await runProjectsSyncCommand(togglClient);
      return 0;
    case "summary":
      await runSummaryCommand(command, togglClient);
      return 0;
    case "time-entries":
      await runTimeEntriesCommand(command, togglClient);
      return 0;
  }
}

if (import.meta.main) {
  Deno.exit(await main(Deno.args));
}
