import { assertEquals, assertRejects, assertThrows } from "@std/assert";
import { CliUsageError, parseCliArgs } from "../cli.ts";
import {
  archiveName,
  defaultUpdateChannel,
  releaseTarget,
  runUpdateCommand,
  type UpdateDependencies,
} from "./update.ts";

const encoder = new TextEncoder();
const ok = (stdout = "") => ({
  success: true,
  stdout: encoder.encode(stdout),
  stderr: new Uint8Array(),
});

async function fixture(
  overrides: Partial<UpdateDependencies> = {},
  options: { version?: string; archive?: Uint8Array } = {},
) {
  const archive = options.archive ?? encoder.encode("archive");
  const digest = await crypto.subtle.digest("SHA-256", archive);
  const checksum = [...new Uint8Array(digest)].map((value) =>
    value.toString(16).padStart(2, "0")
  ).join("");
  const files = new Map<string, Uint8Array>();
  const removed: string[] = [];
  const renamed: string[] = [];
  const version = options.version ?? "1.2.3";
  const deps: UpdateDependencies = {
    fetch: (input) => {
      const url = String(input);
      if (url.endsWith("/releases/latest")) {
        return Promise.resolve(Response.json({ tag_name: `v${version}` }));
      }
      if (url.endsWith(".sha256")) {
        return Promise.resolve(new Response(`${checksum}\n`));
      }
      return Promise.resolve(new Response(archive));
    },
    os: "linux",
    arch: "x86_64",
    execPath: () => "/opt/bin/toggl",
    makeTempDir: () => Promise.resolve("/tmp/update"),
    remove: (path) => {
      removed.push(path);
      files.delete(path);
      return Promise.resolve();
    },
    readFile: (path) =>
      Promise.resolve(files.get(path) ?? encoder.encode("binary")),
    writeFile: (path, data) => {
      files.set(path, data);
      return Promise.resolve();
    },
    chmod: () => Promise.resolve(),
    rename: (from, to) => {
      renamed.push(`${from}->${to}`);
      return Promise.resolve();
    },
    run: (command, args) =>
      args[0] === "--version"
        ? Promise.resolve(ok(version))
        : Promise.resolve(ok()),
    randomId: () => "id",
    ...overrides,
  };
  return { deps, removed, renamed, files, archive, checksum };
}

Deno.test("update CLI parses channels and rejects invalid arguments", () => {
  assertEquals(parseCliArgs(["update"]), {
    name: "update",
    channel: undefined,
  });
  assertEquals(parseCliArgs(["update", "--channel", "nightly"]), {
    name: "update",
    channel: "nightly",
  });
  for (
    const args of [["update", "extra"], ["update", "--channel"], [
      "update",
      "--channel",
      "beta",
    ]]
  ) {
    assertThrows(() => parseCliArgs(args), CliUsageError);
  }
});

Deno.test("update selects channels, artifact names, and supported targets", () => {
  assertEquals(defaultUpdateChannel("nightly-20260806-abcdef1"), "nightly");
  assertEquals(defaultUpdateChannel("1.2.3"), "stable");
  assertEquals(
    archiveName("stable", "1.2.3", "linux-x64"),
    "toggl-cli-v1.2.3-linux-x64.tar.gz",
  );
  assertEquals(
    archiveName("nightly", "ignored", "darwin-arm64"),
    "toggl-cli-nightly-darwin-arm64.tar.gz",
  );
  assertEquals(releaseTarget("linux", "x86_64"), "linux-x64");
  assertEquals(releaseTarget("darwin", "aarch64"), "darwin-arm64");
  assertThrows(
    () => releaseTarget("windows", "x86_64"),
    Error,
    "not supported",
  );
  assertThrows(() => releaseTarget("linux", "aarch64"), Error, "not supported");
});

Deno.test("update reports current and performs a verified atomic update", async () => {
  const current = await fixture();
  assertEquals(
    await runUpdateCommand({ currentVersion: "1.2.3" }, current.deps),
    { status: "current", version: "1.2.3" },
  );
  assertEquals(current.renamed, []);
  const update = await fixture();
  assertEquals(
    await runUpdateCommand({ currentVersion: "1.0.0" }, update.deps),
    { status: "updated", version: "1.2.3" },
  );
  assertEquals(update.renamed, ["/opt/bin/.toggl-update-id->/opt/bin/toggl"]);
  assertEquals(update.removed.includes("/tmp/update"), true);
});

Deno.test("nightly version comes from tag commit date and SHA", async () => {
  const item = await fixture({
    fetch: (input) => {
      const url = String(input);
      if (url.includes("git/ref")) {
        return Promise.resolve(
          Response.json({ object: { sha: "abcdef1234567890" } }),
        );
      }
      if (url.includes("/commits/")) {
        return Promise.resolve(
          Response.json({
            commit: { committer: { date: "2026-08-06T12:00:00Z" } },
          }),
        );
      }
      throw new Error(`unexpected ${url}`);
    },
  });
  assertEquals(
    await runUpdateCommand({
      channel: "nightly",
      currentVersion: "nightly-20260806-abcdef1",
    }, item.deps),
    { status: "current", version: "nightly-20260806-abcdef1" },
  );
});

Deno.test("update rejects checksum mismatch and malformed GitHub data", async () => {
  const mismatch = await fixture({
    fetch: (input) =>
      String(input).endsWith("/releases/latest")
        ? Promise.resolve(Response.json({ tag_name: "v1.2.3" }))
        : Promise.resolve(new Response("bad")),
  });
  await assertRejects(
    () => runUpdateCommand({ currentVersion: "1.0.0" }, mismatch.deps),
    Error,
    "checksum mismatch",
  );
  assertEquals(mismatch.renamed, []);
  const malformed = await fixture({
    fetch: () => Promise.resolve(Response.json({ nope: true })),
  });
  await assertRejects(
    () => runUpdateCommand({ currentVersion: "1.0.0" }, malformed.deps),
    Error,
    "invalid latest release",
  );
});

Deno.test("update preserves binary for extraction, version, and rename failures", async () => {
  for (const failure of ["extract", "version", "rename"] as const) {
    const dependencies: Partial<UpdateDependencies> = {
      run: (_command, args) => {
        if (args[0] === "--version") {
          return Promise.resolve(
            failure === "version" ? ok("9.9.9") : ok("1.2.3"),
          );
        }
        return Promise.resolve(
          failure === "extract" ? { ...ok(), success: false } : ok(),
        );
      },
    };
    if (failure === "rename") {
      dependencies.rename = () =>
        Promise.reject(new Error("permission denied"));
    }
    const item = await fixture(dependencies);
    await assertRejects(() =>
      runUpdateCommand({ currentVersion: "1.0.0" }, item.deps)
    );
    assertEquals(item.removed.includes("/tmp/update"), true);
    if (failure === "rename") {
      assertEquals(item.removed.includes("/opt/bin/.toggl-update-id"), true);
    }
  }
});

Deno.test("update refuses source execution and unsupported OS before downloads", async () => {
  const source = await fixture({ execPath: () => "/usr/bin/deno" });
  await assertRejects(
    () => runUpdateCommand({ currentVersion: "1.0.0" }, source.deps),
    Error,
    "running from source",
  );
  const windows = await fixture({ os: "windows" });
  await assertRejects(
    () => runUpdateCommand({ currentVersion: "1.0.0" }, windows.deps),
    Error,
    "not supported",
  );
  assertEquals(windows.renamed, []);
});
