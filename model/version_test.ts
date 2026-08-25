import { assertEquals, assertThrows } from "@std/assert";
import {
  isNightlyVersion,
  NIGHTLY_SHA_LENGTH,
  nightlyVersion,
} from "./version.ts";

Deno.test("nightly version uses the UTC commit date and a fixed short SHA", () => {
  const timestamp = Date.parse("2026-08-24T23:54:06Z") / 1000;
  const tokyoCalendarDate = new Date((timestamp + 9 * 60 * 60) * 1000)
    .toISOString().slice(0, 10);

  assertEquals(tokyoCalendarDate, "2026-08-25");
  assertEquals(NIGHTLY_SHA_LENGTH, 7);
  assertEquals(
    nightlyVersion(timestamp, "c1648a3b01234567890123456789012345678901"),
    "nightly-20260824-c1648a3",
  );
});

Deno.test("nightly version recognition includes legacy unversioned builds", () => {
  assertEquals(isNightlyVersion("nightly"), true);
  assertEquals(isNightlyVersion("nightly-20260824-c1648a3"), true);
  assertEquals(isNightlyVersion("nightly-20260824-c1648a3b"), false);
  assertEquals(isNightlyVersion("1.2.3"), false);
});

Deno.test("nightly version rejects invalid timestamps and SHAs", () => {
  assertThrows(() => nightlyVersion(Number.NaN, "c1648a3"));
  assertThrows(() => nightlyVersion(0, "not-a-sha"));
});
