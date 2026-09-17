import { basename, dirname, join } from "@std/path";
import { isNightlyVersion, nightlyVersion } from "../model/version.ts";

export type UpdateChannel = "stable" | "nightly";

export interface UpdatePlan {
  channel: UpdateChannel;
  currentVersion: string;
  targetVersion: string;
  target: string;
  executable: string;
  archive: string;
  downloadUrl: string;
  updateAvailable: boolean;
}

export type UpdateResult = {
  channel: UpdateChannel;
  currentVersion: string;
  targetVersion: string;
  updated: boolean;
};

type CommandResult = { success: boolean; output: string };
type FileInfo = { isFile: boolean };

export interface UpdateDependencies {
  fetch: typeof fetch;
  platform: { os: string; arch: string };
  execPath(): string;
  pid: number;
  stat(path: string): Promise<FileInfo>;
  makeTempDir(options?: Deno.MakeTempOptions): Promise<string>;
  makeTempFile(options?: Deno.MakeTempOptions): Promise<string>;
  remove(path: string, options?: Deno.RemoveOptions): Promise<void>;
  copyFile(from: string, to: string): Promise<void>;
  writeFile(path: string, data: Uint8Array): Promise<void>;
  chmod(path: string, mode: number): Promise<void>;
  rename(oldPath: string, newPath: string): Promise<void>;
  run(command: string, args: string[]): Promise<CommandResult>;
  spawn(command: string, args: string[]): void;
}

const defaultDependencies = (): UpdateDependencies => ({
  fetch,
  platform: Deno.build,
  execPath: () => Deno.execPath(),
  pid: Deno.pid,
  stat: (path) => Deno.stat(path),
  makeTempDir: (options) => Deno.makeTempDir(options),
  makeTempFile: (options) => Deno.makeTempFile(options),
  remove: (path, options) => Deno.remove(path, options),
  copyFile: (from, to) => Deno.copyFile(from, to),
  writeFile: (path, data) => Deno.writeFile(path, data),
  chmod: (path, mode) => Deno.chmod(path, mode),
  rename: (from, to) => Deno.rename(from, to),
  run: async (command, args) => {
    const result = await new Deno.Command(command, {
      args,
      stdout: "piped",
      stderr: "piped",
    }).output();
    return {
      success: result.success,
      output: new TextDecoder().decode(
        result.success ? result.stdout : result.stderr,
      ).trim(),
    };
  },
  spawn: (command, args) => {
    const child = new Deno.Command(command, {
      args,
      stdin: "null",
      stdout: "null",
      stderr: "null",
    }).spawn();
    child.unref();
  },
});

const repository = "https://github.com/mukopikmin/toggl-cli";
const apiRepository = "https://api.github.com/repos/mukopikmin/toggl-cli";

export function defaultUpdateChannel(currentVersion: string): UpdateChannel {
  return isNightlyVersion(currentVersion) ? "nightly" : "stable";
}

export function releaseTarget(os: string, arch: string): string {
  if (os === "linux" && arch === "x86_64") return "linux-x64";
  if (os === "darwin" && arch === "aarch64") return "darwin-arm64";
  if (os === "windows" && arch === "x86_64") return "windows-x64";
  throw new Error(`Self-update is not supported on ${os}/${arch}.`);
}

export function archiveName(
  channel: UpdateChannel,
  version: string,
  target: string,
): string {
  const extension = target === "windows-x64" ? "zip" : "tar.gz";
  return channel === "nightly"
    ? `toggl-cli-nightly-${target}.${extension}`
    : `toggl-cli-v${version}-${target}.${extension}`;
}

async function responseJson(response: Response, description: string) {
  if (!response.ok) {
    throw new Error(
      `GitHub ${description} request failed (${response.status}).`,
    );
  }
  try {
    return await response.json();
  } catch {
    throw new Error(`GitHub returned invalid JSON for ${description}.`);
  }
}

function field(value: unknown, key: string, description: string): string {
  if (
    typeof value !== "object" || value === null ||
    typeof (value as Record<string, unknown>)[key] !== "string"
  ) throw new Error(`GitHub returned an invalid ${description} response.`);
  return (value as Record<string, string>)[key];
}

async function desiredVersion(channel: UpdateChannel, fetcher: typeof fetch) {
  const headers = { "User-Agent": "toggl-cli-update" };
  if (channel === "stable") {
    const data = await responseJson(
      await fetcher(`${apiRepository}/releases/latest`, { headers }),
      "latest release",
    );
    const tag = field(data, "tag_name", "latest release");
    if (!/^v\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$/.test(tag)) {
      throw new Error("GitHub returned an invalid latest release tag.");
    }
    return { version: tag.slice(1), tag };
  }
  const ref = await responseJson(
    await fetcher(`${apiRepository}/git/ref/tags/nightly`, { headers }),
    "nightly tag",
  );
  const object = typeof ref === "object" && ref !== null
    ? (ref as Record<string, unknown>).object
    : undefined;
  const sha = field(object, "sha", "nightly tag");
  if (!/^[0-9a-f]{7,40}$/i.test(sha)) {
    throw new Error("GitHub returned an invalid nightly commit SHA.");
  }
  const commit = await responseJson(
    await fetcher(`${apiRepository}/commits/${sha}`, { headers }),
    "nightly commit",
  );
  const commitData = typeof commit === "object" && commit !== null
    ? (commit as Record<string, unknown>).commit
    : undefined;
  const committer = typeof commitData === "object" && commitData !== null
    ? (commitData as Record<string, unknown>).committer
    : undefined;
  const date = field(committer, "date", "nightly commit");
  const timestamp = Date.parse(date) / 1000;
  if (!Number.isInteger(timestamp)) {
    throw new Error("GitHub returned an invalid nightly commit date.");
  }
  return { version: nightlyVersion(timestamp, sha), tag: "nightly" };
}

function stableParts(version: string): number[] | undefined {
  return version.match(/^(\d+)\.(\d+)\.(\d+)(?:[-+].*)?$/)?.slice(1, 4)
    .map(Number);
}

export function updateIsNewer(
  channel: UpdateChannel,
  currentVersion: string,
  targetVersion: string,
): boolean {
  if (currentVersion === targetVersion) return false;
  if (channel === "nightly") {
    const currentDay = currentVersion.match(/^nightly-(\d{8})-[0-9a-f]+$/i)
      ?.[1];
    const targetDay = targetVersion.match(/^nightly-(\d{8})-[0-9a-f]+$/i)?.[1];
    if (currentDay && targetDay && currentDay !== targetDay) {
      return targetDay > currentDay;
    }
    return true;
  }
  const current = stableParts(currentVersion);
  const target = stableParts(targetVersion);
  if (!current || !target) return true;
  for (let index = 0; index < 3; index++) {
    if (current[index] !== target[index]) return target[index] > current[index];
  }
  return currentVersion.includes("-") && !targetVersion.includes("-");
}

function validVersion(value: string): boolean {
  return /^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$/.test(value) ||
    value === "nightly" ||
    /^nightly-\d{8}-[0-9a-f]{7,40}$/i.test(value);
}

export async function checkForUpdate(
  currentVersion: string,
  requestedChannel?: UpdateChannel,
  deps: UpdateDependencies = defaultDependencies(),
): Promise<UpdatePlan> {
  const channel = requestedChannel ?? defaultUpdateChannel(currentVersion);
  const target = releaseTarget(deps.platform.os, deps.platform.arch);
  const executable = deps.execPath();
  if (/^deno(?:\.exe)?$/i.test(basename(executable))) {
    throw new Error(
      "Self-update is unavailable when running from source with Deno. Install a compiled toggl binary first.",
    );
  }
  const info = await deps.stat(executable).catch(() => undefined);
  const probe = info?.isFile
    ? await deps.run(executable, ["--version"]).catch(() => ({
      success: false,
      output: "",
    }))
    : undefined;
  if (
    !info?.isFile || !probe?.success || !validVersion(probe.output) ||
    probe.output !== currentVersion
  ) {
    throw new Error(
      `The running executable is not the expected compiled Toggl CLI binary: ${executable}`,
    );
  }
  const desired = await desiredVersion(channel, deps.fetch);
  const archive = archiveName(channel, desired.version, target);
  return {
    channel,
    currentVersion,
    targetVersion: desired.version,
    target,
    executable,
    archive,
    downloadUrl: `${repository}/releases/download/${desired.tag}/${archive}`,
    updateAvailable: updateIsNewer(channel, currentVersion, desired.version),
  };
}

export async function verifyChecksum(
  archive: Uint8Array,
  checksumText: string,
): Promise<void> {
  const checksum = checksumText.trim();
  if (!/^[0-9a-f]{64}$/i.test(checksum)) {
    throw new Error(
      "Release checksum is not exactly one 64-digit hexadecimal SHA-256 value.",
    );
  }
  const digest = await crypto.subtle.digest("SHA-256", archive);
  const actual = [...new Uint8Array(digest)].map((byte) =>
    byte.toString(16).padStart(2, "0")
  ).join("");
  if (actual !== checksum.toLowerCase()) {
    throw new Error(
      "Update checksum mismatch; the existing binary was not changed.",
    );
  }
}

export async function installUpdate(
  plan: UpdatePlan,
  deps: UpdateDependencies = defaultDependencies(),
): Promise<UpdateResult> {
  const result = {
    channel: plan.channel,
    currentVersion: plan.currentVersion,
    targetVersion: plan.targetVersion,
  };
  if (!plan.updateAvailable) return { ...result, updated: false };
  const [archiveResponse, checksumResponse] = await Promise.all([
    deps.fetch(plan.downloadUrl),
    deps.fetch(`${plan.downloadUrl}.sha256`),
  ]);
  if (!archiveResponse.ok || !checksumResponse.ok) {
    throw new Error("Failed to download the update archive or checksum.");
  }
  const archiveBytes = new Uint8Array(await archiveResponse.arrayBuffer());
  await verifyChecksum(archiveBytes, await checksumResponse.text());

  const workDir = await deps.makeTempDir({ prefix: "toggl-update-" });
  let stagedPath: string | undefined;
  let helperPath: string | undefined;
  try {
    const archivePath = join(workDir, plan.archive);
    await deps.writeFile(archivePath, archiveBytes);
    const windows = plan.target === "windows-x64";
    const extraction = windows
      ? await deps.run("powershell.exe", [
        "-NoProfile",
        "-NonInteractive",
        "-Command",
        "Expand-Archive -LiteralPath $args[0] -DestinationPath $args[1] -Force",
        archivePath,
        workDir,
      ])
      : await deps.run("tar", ["-xzf", archivePath, "-C", workDir]);
    if (!extraction.success) {
      throw new Error(
        `Failed to extract the update archive; the existing binary was not changed. ${extraction.output}`
          .trim(),
      );
    }
    const root = plan.archive.replace(/\.(?:tar\.gz|zip)$/, "");
    const extracted = join(workDir, root, windows ? "toggl.exe" : "toggl");
    const extractedInfo = await deps.stat(extracted).catch(() => undefined);
    if (!extractedInfo?.isFile) {
      throw new Error(
        "Release archive does not contain the expected Toggl CLI binary.",
      );
    }
    const verification = await deps.run(extracted, ["--version"]);
    if (!verification.success || verification.output !== plan.targetVersion) {
      throw new Error(
        `Downloaded binary version mismatch (expected ${plan.targetVersion}); the existing binary was not changed.`,
      );
    }
    stagedPath = await deps.makeTempFile({
      dir: dirname(plan.executable),
      prefix: ".toggl-update-",
    });
    await deps.copyFile(extracted, stagedPath);
    if (windows) {
      helperPath = await deps.makeTempFile({
        dir: dirname(plan.executable),
        prefix: ".toggl-update-",
        suffix: ".ps1",
      });
      const quote = (value: string) => value.replaceAll("'", "''");
      const script = `$ErrorActionPreference = "Stop"
$updated = $false
try {
  Wait-Process -Id ${deps.pid} -ErrorAction SilentlyContinue
  Move-Item -LiteralPath '${quote(stagedPath)}' -Destination '${
        quote(plan.executable)
      }' -Force
  $updated = $true
} finally {
  if (-not $updated) { Remove-Item -LiteralPath '${
        quote(stagedPath)
      }' -Force -ErrorAction SilentlyContinue }
  Remove-Item -LiteralPath $PSCommandPath -Force -ErrorAction SilentlyContinue
}
`;
      await deps.writeFile(helperPath, new TextEncoder().encode(script));
      deps.spawn("powershell.exe", [
        "-NoProfile",
        "-NonInteractive",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        helperPath,
      ]);
      stagedPath = undefined;
      helperPath = undefined;
    } else {
      await deps.chmod(stagedPath, 0o755);
      await deps.rename(stagedPath, plan.executable);
      stagedPath = undefined;
    }
    return { ...result, updated: true };
  } finally {
    if (stagedPath) await deps.remove(stagedPath).catch(() => undefined);
    if (helperPath) await deps.remove(helperPath).catch(() => undefined);
    await deps.remove(workDir, { recursive: true }).catch(() => undefined);
  }
}

export async function runUpdateCommand(
  options: { channel?: UpdateChannel; currentVersion: string },
  deps: UpdateDependencies = defaultDependencies(),
): Promise<UpdateResult> {
  return await installUpdate(
    await checkForUpdate(options.currentVersion, options.channel, deps),
    deps,
  );
}
