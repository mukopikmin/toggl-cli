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

function startOfDayUtcIso(day: DateTime): string {
  return new Date(Date.UTC(day.year, day.month - 1, day.day)).toISOString();
}

export function buildTimeEntriesDateRange(
  fromDay: DateTime,
  toDay: DateTime,
  timeZone?: string,
): TimeEntriesDateRange {
  if (!timeZone) {
    return {
      startDate: startOfDayUtcIso(fromDay),
      endDate: startOfDayUtcIso(toDay.add({ day: 1 })),
    };
  }

  return {
    startDate: startOfDayInTimeZoneUtcIso(fromDay, timeZone),
    endDate: startOfDayInTimeZoneUtcIso(toDay.add({ day: 1 }), timeZone),
  };
}
