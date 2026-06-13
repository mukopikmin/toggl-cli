import { DateTime } from "ptera";

export interface TimeEntriesDateRange {
  startDate: string;
  endDate: string;
}

function startOfDayInTimeZoneUtcIso(day: DateTime, timeZone: string): string {
  return Temporal.ZonedDateTime.from({
    timeZone,
    year: day.year,
    month: day.month,
    day: day.day,
    hour: 0,
    minute: 0,
    second: 0,
  }).toInstant().toString();
}

export function buildTimeEntriesDateRange(
  fromDay: DateTime,
  toDay: DateTime,
  timeZone?: string,
): TimeEntriesDateRange {
  if (!timeZone) {
    return {
      startDate: fromDay.toUTC().toISO(),
      endDate: toDay.add({ day: 1 }).toUTC().toISO(),
    };
  }

  return {
    startDate: startOfDayInTimeZoneUtcIso(fromDay, timeZone),
    endDate: startOfDayInTimeZoneUtcIso(toDay.add({ day: 1 }), timeZone),
  };
}
