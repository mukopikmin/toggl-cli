import { assertEquals } from "@std/assert";
import { aggregateTimeEntries } from "./time_entry_aggregation.ts";
import type { TimeEntry } from "./types.ts";

function entry(
  id: number,
  projectId: number,
  start: string,
  duration: number,
): TimeEntry {
  return {
    id,
    project_id: projectId,
    start,
    stop: "",
    duration,
    description: "",
  };
}

Deno.test("aggregateTimeEntries sums normal entries by UTC date and project", () => {
  const entries = [
    entry(1, 100, "2026-05-01T00:00:00Z", 1800),
    entry(2, 100, "2026-05-01T23:59:59Z", 900),
    entry(3, 200, "2026-05-01T12:00:00Z", 3600),
    entry(4, 100, "2026-05-02T00:00:00Z", 600),
  ];

  assertEquals(
    aggregateTimeEntries(
      entries,
      "UTC",
      Temporal.Instant.from("2026-05-03T00:00:00Z"),
    ),
    {
      "2026-05-01": { 100: 45, 200: 60 },
      "2026-05-02": { 100: 10 },
    },
  );
});

Deno.test("aggregateTimeEntries calculates a running entry from the supplied instant", () => {
  const start = Temporal.Instant.from("2026-05-01T10:00:00Z");
  const runningDuration = -Math.floor(start.epochMilliseconds / 1000);

  assertEquals(
    aggregateTimeEntries(
      [entry(1, 100, start.toString(), runningDuration)],
      "UTC",
      Temporal.Instant.from("2026-05-01T10:45:00Z"),
    ),
    { "2026-05-01": { 100: 45 } },
  );
});

Deno.test("aggregateTimeEntries uses the configured timezone at date and month boundaries", () => {
  const entries = [
    entry(1, 100, "2026-01-31T14:59:59Z", 60),
    entry(2, 100, "2026-01-31T15:00:00Z", 120),
    entry(3, 200, "2026-02-01T04:59:59Z", 180),
    entry(4, 200, "2026-02-01T05:00:00Z", 240),
  ];
  const now = Temporal.Instant.from("2026-02-02T00:00:00Z");

  assertEquals(aggregateTimeEntries(entries, "Asia/Tokyo", now), {
    "2026-01-31": { 100: 1 },
    "2026-02-01": { 100: 2, 200: 7 },
  });
  assertEquals(aggregateTimeEntries(entries, "America/New_York", now), {
    "2026-01-31": { 100: 3, 200: 3 },
    "2026-02-01": { 200: 4 },
  });
});
