export type DateTime = Temporal.PlainDate & {
  year: number;
  month: number;
  day: number;
  hour: number;
  minute: number;
  second: number;
  millisecond: number;
  isValid(): boolean;
  isAfter(other: DateTime): boolean;
};

type DateTimeInput = Partial<
  Pick<
    DateTime,
    "year" | "month" | "day" | "hour" | "minute" | "second" | "millisecond"
  >
>;

function toDateTimeFields(fields?: DateTimeInput): DateTimeInput {
  if (fields === undefined) {
    const now = Temporal.Now.plainDateTimeISO();
    return {
      year: now.year,
      month: now.month,
      day: now.day,
      hour: now.hour,
      minute: now.minute,
      second: now.second,
      millisecond: now.millisecond,
    };
  }

  return {
    year: fields.year,
    month: fields.month,
    day: fields.day,
    hour: fields.hour ?? 0,
    minute: fields.minute ?? 0,
    second: fields.second ?? 0,
    millisecond: fields.millisecond ?? 0,
  };
}

function createValidDateTime(
  fields: Required<DateTimeInput>,
): DateTime {
  const date = Temporal.PlainDate.from(fields);
  return Object.assign(date, {
    hour: fields.hour,
    minute: fields.minute,
    second: fields.second,
    millisecond: fields.millisecond,
    isValid: () => true,
    isAfter(other: DateTime): boolean {
      if (!other.isValid()) return false;
      return Temporal.PlainDate.compare(
        date,
        Temporal.PlainDate.from({
          year: other.year,
          month: other.month,
          day: other.day,
        }),
      ) > 0;
    },
  });
}

function createInvalidDateTime(fields: DateTimeInput): DateTime {
  const date = Temporal.PlainDate.from({ year: 1970, month: 1, day: 1 });
  return Object.assign(date, {
    year: fields.year ?? date.year,
    month: fields.month ?? date.month,
    day: fields.day ?? date.day,
    hour: fields.hour ?? 0,
    minute: fields.minute ?? 0,
    second: fields.second ?? 0,
    millisecond: fields.millisecond ?? 0,
    isValid: () => false,
    isAfter: () => false,
  });
}

export function datetime(fields?: DateTimeInput): DateTime {
  const normalized = toDateTimeFields(fields);
  try {
    return createValidDateTime(normalized as Required<DateTimeInput>);
  } catch {
    return createInvalidDateTime(normalized);
  }
}
