import { ClipboardUnavailableError } from "./clipboard.ts";
import { CliUsageError, HELP_TEXT, parseCliArgs } from "./cli.ts";
import { runConfigCommand } from "./command/config.ts";
import { runInitCommand } from "./command/init.ts";
import {
  ProjectReorderUnavailableError,
  runProjectListCommand,
  runProjectReorderCommand,
  runProjectSyncCommand,
} from "./command/project.ts";
import { runSummaryCommand } from "./command/summary.ts";
import { runTimeEntryListCommand } from "./command/time_entry.ts";
import { runUpdateCommand } from "./command/update.ts";
import { togglClient } from "./toggl/api.ts";
import { TogglApiError } from "./toggl/error.ts";
import { version } from "./version.ts";
import {
  ConfigFileNotFoundError,
  ConfigFileReadError,
  ConfigValidationError,
  HomeNotSetError,
} from "./config.ts";

function reportConfigError(error: unknown): boolean {
  if (error instanceof ConfigValidationError) {
    const details = [
      ...(error.missingKeys.length
        ? [`Missing required configuration: ${error.missingKeys.join(", ")}`]
        : []),
      ...(error.invalidProjects.length
        ? [`Invalid project configuration: ${error.invalidProjects.join(", ")}`]
        : []),
    ].join("; ");
    console.error(`Error: ${details}`);
    return true;
  }
  if (error instanceof ConfigFileNotFoundError) {
    console.error(`Error: ${error.message}`);
    console.error(
      "Please create ~/.config/toggl-cli/config.toml with the following format:",
    );
    console.error('workspace = "your_workspace_id"');
    console.error('token = "your_api_token"');
    return true;
  }
  if (
    error instanceof HomeNotSetError || error instanceof ConfigFileReadError
  ) {
    console.error(`Error: ${error.message}`);
    return true;
  }
  return false;
}

export interface MainDependencies {
  runConfigCommand: typeof runConfigCommand;
}

const defaultMainDependencies: MainDependencies = { runConfigCommand };

export async function main(
  args: string[],
  dependencies: MainDependencies = defaultMainDependencies,
): Promise<number> {
  let command;
  try {
    command = parseCliArgs(args);
  } catch (error) {
    if (!(error instanceof CliUsageError)) throw error;
    console.error(`Error: ${error.message}\n\n${HELP_TEXT}`);
    return 1;
  }

  try {
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
      case "project-list":
        await runProjectListCommand({ format: command.format }, togglClient);
        return 0;
      case "project-reorder":
        await runProjectReorderCommand(togglClient);
        return 0;
      case "config":
        await dependencies.runConfigCommand(command.format);
        return 0;
      case "project-sync":
        await runProjectSyncCommand(togglClient);
        return 0;
      case "summary":
        await runSummaryCommand(command, togglClient);
        return 0;
      case "time-entry-list":
        await runTimeEntryListCommand(command, togglClient);
        return 0;
      case "update": {
        const result = await runUpdateCommand({
          channel: command.channel,
          currentVersion: version,
        });
        console.log(
          result.status === "updated"
            ? `Updated to ${result.version}.`
            : `Already up to date (${result.version}).`,
        );
        return 0;
      }
    }
  } catch (error) {
    if (reportConfigError(error)) return 1;
    if (error instanceof ProjectReorderUnavailableError) {
      console.error(`Error: ${error.message}`);
      return 1;
    }
    if (
      !(error instanceof ClipboardUnavailableError) &&
      !(error instanceof TogglApiError)
    ) {
      throw error;
    }
    console.error(`Error: ${error.message}`);
    return 1;
  }
}

if (import.meta.main) {
  Deno.exit(await main(Deno.args));
}
