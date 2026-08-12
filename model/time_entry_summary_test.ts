import { assertEquals } from "@std/assert";
import { summarizeTimeEntries } from "./time_entry_summary.ts";
import type { TimeEntry } from "./time_entry.ts";

function entry(
  id: number,
  projectId: number,
  start: string,
  duration: number,
): TimeEntry {
  return {
    id,
    projectId,
    start,
    stop: "",
    durationSeconds: duration,
    description: "",
  };
}

Deno.test("summarizeTimeEntries sums normal entries by UTC date and project", () => {
  const entries = [
    entry(1, 100, "2026-05-01T00:00:00Z", 1800),
    entry(2, 100, "2026-05-01T23:59:59Z", 900),
    entry(3, 200, "2026-05-01T12:00:00Z", 3600),
    entry(4, 100, "2026-05-02T00:00:00Z", 600),
  ];

  assertEquals(
    summarizeTimeEntries(
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

Deno.test("summarizeTimeEntries calculates a running entry from the supplied instant", () => {
  const start = Temporal.Instant.from("2026-05-01T10:00:00Z");
  const runningDuration = -Math.floor(start.epochMilliseconds / 1000);

  assertEquals(
    summarizeTimeEntries(
      [entry(1, 100, start.toString(), runningDuration)],
      "UTC",
      Temporal.Instant.from("2026-05-01T10:45:00Z"),
    ),
    { "2026-05-01": { 100: 45 } },
  );
});

Deno.test("summarizeTimeEntries uses the configured timezone at date and month boundaries", () => {
  const entries = [
    entry(1, 100, "2026-01-31T14:59:59Z", 60),
    entry(2, 100, "2026-01-31T15:00:00Z", 120),
    entry(3, 200, "2026-02-01T04:59:59Z", 180),
    entry(4, 200, "2026-02-01T05:00:00Z", 240),
  ];
  const now = Temporal.Instant.from("2026-02-02T00:00:00Z");

  assertEquals(summarizeTimeEntries(entries, "Asia/Tokyo", now), {
    "2026-01-31": { 100: 1 },
    "2026-02-01": { 100: 2, 200: 7 },
  });
  assertEquals(summarizeTimeEntries(entries, "America/New_York", now), {
    "2026-01-31": { 100: 3, 200: 3 },
    "2026-02-01": { 200: 4 },
  });
});
