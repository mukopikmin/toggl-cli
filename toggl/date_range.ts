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
  if (!timeZone) {
    return {
      startDate: fromDay.toZonedDateTime("UTC").toInstant().toString({
        fractionalSecondDigits: 3,
      }),
      endDate: toDay.add({ days: 1 }).toZonedDateTime("UTC").toInstant()
        .toString({ fractionalSecondDigits: 3 }),
    };
  }

  return {
    startDate: startOfDayInTimeZoneUtcIso(fromDay, timeZone),
    endDate: startOfDayInTimeZoneUtcIso(toDay.add({ days: 1 }), timeZone),
  };
}
