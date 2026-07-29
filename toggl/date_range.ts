import { resolveTimeZone } from "./date.ts";

export interface TimeEntriesDateRange {
  startDate: string;
  endDate: string;
}

function startOfDayInTimeZoneUtcIso(
  day: Temporal.PlainDate,
  timeZone: string,
): string {
  return day.toZonedDateTime(timeZone).toInstant().toString();
}

export function buildTimeEntriesDateRange(
  fromDay: Temporal.PlainDate,
  toDay: Temporal.PlainDate,
  timeZone?: string,
): TimeEntriesDateRange {
  const resolvedTimeZone = resolveTimeZone(timeZone);

  return {
    startDate: startOfDayInTimeZoneUtcIso(fromDay, resolvedTimeZone),
    endDate: startOfDayInTimeZoneUtcIso(
      toDay.add({ days: 1 }),
      resolvedTimeZone,
    ),
  };
}
