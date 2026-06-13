import { assertEquals } from "@std/assert";
import { datetime } from "ptera";
import { buildTimeEntriesDateRange } from "./date_range.ts";

Deno.test("buildTimeEntriesDateRange preserves range without configured timezone", () => {
  const fromDay = datetime({ year: 2026, month: 5, day: 1 });
  const toDay = datetime({ year: 2026, month: 5, day: 2 });

  assertEquals(buildTimeEntriesDateRange(fromDay, toDay), {
    startDate: fromDay.toUTC().toISO(),
    endDate: toDay.add({ day: 1 }).toUTC().toISO(),
  });
});

Deno.test("buildTimeEntriesDateRange applies configured timezone to range", () => {
  const fromDay = datetime({ year: 2026, month: 5, day: 1 });
  const toDay = datetime({ year: 2026, month: 5, day: 2 });

  assertEquals(buildTimeEntriesDateRange(fromDay, toDay, "Asia/Tokyo"), {
    startDate: "2026-04-30T15:00:00Z",
    endDate: "2026-05-02T15:00:00Z",
  });
});
