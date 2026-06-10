import { parseArgs } from "node:util";
import { datetime } from "ptera";
import { runInitCommand } from "./command/init.ts";
import { runProjectsCommand } from "./command/projects.ts";
import { runSummaryCommand } from "./command/summary.ts";
import { togglClient } from "./toggl/api.ts";
import { version } from "./version.ts";

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
    },
    allowPositionals: true,
  });
  const { format, lastMonth, separator, version: showVersion } = args.values;

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
    await runProjectsCommand({ format }, togglClient);
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
