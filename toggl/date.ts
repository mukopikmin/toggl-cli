export function formatTimeEntryDate(
  start: string,
  timeZone?: string,
): string {
  const startDate = new Date(start);

  if (!timeZone) {
    const yStr = String(startDate.getFullYear());
    const mStr = String(startDate.getMonth() + 1).padStart(2, "0");
    const dStr = String(startDate.getDate()).padStart(2, "0");
    return `${yStr}-${mStr}-${dStr}`;
  }

  const parts = new Intl.DateTimeFormat("en-US", {
    timeZone,
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
