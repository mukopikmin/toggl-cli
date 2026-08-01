import { assertEquals } from "@std/assert";
import { buildTimeEntriesDateRange } from "./date_range.ts";

Deno.test("buildTimeEntriesDateRange applies UTC when configured", () => {
  const fromDay = Temporal.PlainDate.from("2026-05-01");
  const toDay = Temporal.PlainDate.from("2026-05-02");

  assertEquals(buildTimeEntriesDateRange(fromDay, toDay, "UTC"), {
    startDate: "2026-05-01T00:00:00Z",
    endDate: "2026-05-03T00:00:00Z",
  });
});

Deno.test("buildTimeEntriesDateRange applies configured timezone to range", () => {
  const fromDay = Temporal.PlainDate.from("2026-05-01");
  const toDay = Temporal.PlainDate.from("2026-05-02");

  assertEquals(buildTimeEntriesDateRange(fromDay, toDay, "Asia/Tokyo"), {
    startDate: "2026-04-30T15:00:00Z",
    endDate: "2026-05-02T15:00:00Z",
  });
});
