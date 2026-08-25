export const NIGHTLY_SHA_LENGTH = 7;

export function nightlyVersion(
  commitTimestamp: number,
  commitSha: string,
): string {
  if (!Number.isSafeInteger(commitTimestamp) || commitTimestamp < 0) {
    throw new Error("Invalid commit timestamp.");
  }
  if (
    !new RegExp(`^[0-9a-f]{${NIGHTLY_SHA_LENGTH},40}$`, "i").test(commitSha)
  ) {
    throw new Error("Invalid commit SHA.");
  }

  const date = new Date(commitTimestamp * 1000);
  if (Number.isNaN(date.valueOf())) {
    throw new Error("Invalid commit timestamp.");
  }
  const utcDate = date.toISOString().slice(0, 10).replaceAll("-", "");
  return `nightly-${utcDate}-${commitSha.slice(0, NIGHTLY_SHA_LENGTH)}`;
}

export function isNightlyVersion(version: string): boolean {
  return version === "nightly" ||
    new RegExp(
      `^nightly-\\d{8}-[0-9a-f]{${NIGHTLY_SHA_LENGTH}}$`,
      "i",
    ).test(version);
}
