import {
  assertEquals,
  assertRejects,
  assertStringIncludes,
  assertThrows,
} from "@std/assert";
import { CliUsageError, parseCliArgs } from "../cli.ts";
import {
  archiveName,
  checkForUpdate,
  defaultUpdateChannel,
  installUpdate,
  releaseTarget,
  type UpdateDependencies,
  updateIsNewer,
  verifyChecksum,
} from "./update.ts";

const encoder = new TextEncoder();

async function fixture(options: {
  current?: string;
  target?: string;
  os?: string;
  checksum?: string;
  extractionFails?: boolean;
  executableIsFile?: boolean;
  probeOutput?: string;
  renameFails?: boolean;
} = {}) {
  const current = options.current ?? "1.0.0";
  const target = options.target ?? "1.2.3";
  const archiveBytes = encoder.encode("archive");
  const digest = await crypto.subtle.digest("SHA-256", archiveBytes);
  const checksum = [...new Uint8Array(digest)].map((value) =>
    value.toString(16).padStart(2, "0")
  ).join("");
  const removed: string[] = [];
  const renamed: string[] = [];
  const spawned: Array<{ command: string; args: string[] }> = [];
  const written = new Map<string, Uint8Array>();
  let tempFileIndex = 0;
  let fetchCount = 0;
  const deps: UpdateDependencies = {
    fetch: (input) => {
      fetchCount++;
      const url = String(input);
      if (url.endsWith("/releases/latest")) {
        return Promise.resolve(Response.json({ tag_name: `v${target}` }));
      }
      if (url.includes("git/ref")) {
        return Promise.resolve(
          Response.json({ object: { sha: "abcdef1234567890" } }),
        );
      }
      if (url.includes("/commits/")) {
        return Promise.resolve(
          Response.json({
            commit: { committer: { date: "2026-08-24T23:54:06Z" } },
          }),
        );
      }
      if (url.endsWith(".sha256")) {
        return Promise.resolve(new Response(options.checksum ?? checksum));
      }
      return Promise.resolve(new Response(archiveBytes));
    },
    platform: { os: options.os ?? "linux", arch: "x86_64" },
    execPath: () =>
      options.os === "windows" ? "C:\\bin\\toggl.exe" : "/opt/bin/toggl",
    pid: 42,
    stat: (path) =>
      Promise.resolve({
        isFile: path === (options.os === "windows"
            ? "C:\\bin\\toggl.exe"
            : "/opt/bin/toggl")
          ? options.executableIsFile !== false
          : true,
      }),
    makeTempDir: () => Promise.resolve("/tmp/update"),
    makeTempFile: ({ suffix = "" } = {}) =>
      Promise.resolve(`/opt/bin/.toggl-update-${++tempFileIndex}${suffix}`),
    remove: (path) => {
      removed.push(path);
      return Promise.resolve();
    },
    copyFile: () => Promise.resolve(),
    writeFile: (path, data) => {
      written.set(path, data);
      return Promise.resolve();
    },
    chmod: () => Promise.resolve(),
    rename: (from, to) => {
      if (options.renameFails) return Promise.reject(new Error("denied"));
      renamed.push(`${from}->${to}`);
      return Promise.resolve();
    },
    run: (command, args) => {
      if (args[0] === "--version") {
        const installed = command === (options.os === "windows"
          ? "C:\\bin\\toggl.exe"
          : "/opt/bin/toggl");
        return Promise.resolve({
          success: true,
          output: installed ? options.probeOutput ?? current : target,
        });
      }
      return Promise.resolve({
        success: !options.extractionFails,
        output: options.extractionFails ? "broken archive" : "",
      });
    },
    spawn: (command, args) => spawned.push({ command, args }),
  };
  return {
    deps,
    removed,
    renamed,
    spawned,
    written,
    get fetchCount() {
      return fetchCount;
    },
  };
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
  assertThrows(
    () => parseCliArgs(["update", "--channel", "beta"]),
    CliUsageError,
  );
});

Deno.test("update selects channels and names all published archives", () => {
  assertEquals(defaultUpdateChannel("nightly-20260806-abcdef1"), "nightly");
  assertEquals(defaultUpdateChannel("1.2.3"), "stable");
  assertEquals(releaseTarget("linux", "x86_64"), "linux-x64");
  assertEquals(releaseTarget("darwin", "aarch64"), "darwin-arm64");
  assertEquals(releaseTarget("windows", "x86_64"), "windows-x64");
  assertEquals(
    archiveName("stable", "1.2.3", "windows-x64"),
    "toggl-cli-v1.2.3-windows-x64.zip",
  );
  assertEquals(
    archiveName("nightly", "ignored", "windows-x64"),
    "toggl-cli-nightly-windows-x64.zip",
  );
  assertThrows(() => releaseTarget("linux", "aarch64"), Error, "not supported");
});

Deno.test("semantic comparison prevents downgrades and upgrades prereleases", () => {
  assertEquals(updateIsNewer("stable", "2.0.0", "1.9.9"), false);
  assertEquals(updateIsNewer("stable", "1.2.3-beta.1", "1.2.3"), true);
  assertEquals(updateIsNewer("stable", "1.2.3", "1.2.3-beta.1"), false);
  assertEquals(
    updateIsNewer(
      "nightly",
      "nightly-20260824-aaaaaaa",
      "nightly-20260824-bbbbbbb",
    ),
    true,
  );
  assertEquals(
    updateIsNewer(
      "nightly",
      "nightly-20260825-aaaaaaa",
      "nightly-20260824-bbbbbbb",
    ),
    false,
  );
});

Deno.test("planning verifies the current executable before querying GitHub", async () => {
  for (
    const options of [{ executableIsFile: false }, { probeOutput: "not toggl" }]
  ) {
    const item = await fixture(options);
    await assertRejects(
      () => checkForUpdate("1.0.0", undefined, item.deps),
      Error,
      "expected compiled Toggl CLI",
    );
    assertEquals(item.fetchCount, 0);
  }
});

Deno.test("planning reports downgrade as unavailable and same-day nightly SHA as available", async () => {
  const stable = await fixture({ current: "2.0.0", target: "1.2.3" });
  assertEquals(
    (await checkForUpdate("2.0.0", undefined, stable.deps)).updateAvailable,
    false,
  );
  const nightly = await fixture({
    current: "nightly-20260824-1111111",
    probeOutput: "nightly-20260824-1111111",
  });
  const plan = await checkForUpdate(
    "nightly-20260824-1111111",
    "nightly",
    nightly.deps,
  );
  assertEquals(plan.targetVersion, "nightly-20260824-abcdef1");
  assertEquals(plan.updateAvailable, true);
});

Deno.test("checksum requires exactly one hexadecimal digest", async () => {
  const bytes = encoder.encode("archive");
  const digest = await crypto.subtle.digest("SHA-256", bytes);
  const checksum = [...new Uint8Array(digest)].map((value) =>
    value.toString(16).padStart(2, "0")
  ).join("");
  await verifyChecksum(bytes, ` ${checksum}\n`);
  await assertRejects(
    () => verifyChecksum(bytes, `${checksum}  archive.tar.gz`),
    Error,
    "exactly one",
  );
});

Deno.test("Linux installation atomically replaces and cleans temporary files", async () => {
  const item = await fixture();
  const plan = await checkForUpdate("1.0.0", undefined, item.deps);
  assertEquals((await installUpdate(plan, item.deps)).updated, true);
  assertEquals(item.renamed, ["/opt/bin/.toggl-update-1->/opt/bin/toggl"]);
  assertEquals(item.removed, ["/tmp/update"]);
});

Deno.test("Windows installation defers replacement to a detached helper", async () => {
  const item = await fixture({ os: "windows" });
  const plan = await checkForUpdate("1.0.0", undefined, item.deps);
  await installUpdate(plan, item.deps);
  assertEquals(item.renamed, []);
  assertEquals(item.spawned.length, 1);
  assertEquals(item.spawned[0].command, "powershell.exe");
  const script = new TextDecoder().decode(
    item.written.get("/opt/bin/.toggl-update-2.ps1"),
  );
  assertStringIncludes(script, "Wait-Process -Id 42");
  assertStringIncludes(script, "Move-Item");
  assertEquals(item.removed, ["/tmp/update"]);
});

Deno.test("extraction failures preserve the executable and clean work files", async () => {
  const item = await fixture({ extractionFails: true });
  const plan = await checkForUpdate("1.0.0", undefined, item.deps);
  await assertRejects(
    () => installUpdate(plan, item.deps),
    Error,
    "Failed to extract",
  );
  assertEquals(item.renamed, []);
  assertEquals(item.removed, ["/tmp/update"]);
});

Deno.test("failed atomic replacement removes staged and work files", async () => {
  const item = await fixture({ renameFails: true });
  const plan = await checkForUpdate("1.0.0", undefined, item.deps);
  await assertRejects(() => installUpdate(plan, item.deps), Error, "denied");
  assertEquals(item.removed, ["/opt/bin/.toggl-update-1", "/tmp/update"]);
});
