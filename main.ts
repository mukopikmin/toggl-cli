import { parseArgs } from "node:util";
import { datetime } from "ptera";
import { runSummaryCommand } from "./command/summary.ts";
import { togglClient } from "./toggl/api.ts";

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

  runSummaryCommand({ startDay, endDay, separator }, togglClient);
}
