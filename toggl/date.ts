export function resolveTimeZone(
  configuredTimeZone?: string,
  systemTimeZone = Intl.DateTimeFormat().resolvedOptions().timeZone,
): string {
  return configuredTimeZone ?? systemTimeZone;
}

export function formatTimeEntryDate(
  start: string,
  timeZone?: string,
): string {
  const startDate = new Date(start);

  const parts = new Intl.DateTimeFormat("en-US", {
    timeZone: resolveTimeZone(timeZone),
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(startDate);
  const dateParts = Object.fromEntries(
    parts
      .filter((part) => ["year", "month", "day"].includes(part.type))
      .map((part) => [part.type, part.value]),
  );

  return `${dateParts.year}-${dateParts.month}-${dateParts.day}`;
}
