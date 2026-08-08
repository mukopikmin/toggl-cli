import { basename, dirname, join } from "@std/path";

export type UpdateChannel = "stable" | "nightly";
export type UpdateResult =
  | { status: "current"; version: string }
  | { status: "updated"; version: string };

type CommandResult = {
  success: boolean;
  stdout: Uint8Array;
  stderr: Uint8Array;
};

export interface UpdateDependencies {
  fetch: typeof fetch;
  os: string;
  arch: string;
  execPath(): string;
  makeTempDir(): Promise<string>;
  remove(path: string, options?: { recursive?: boolean }): Promise<void>;
  readFile(path: string): Promise<Uint8Array>;
  writeFile(path: string, data: Uint8Array): Promise<void>;
  chmod(path: string, mode: number): Promise<void>;
  rename(oldPath: string, newPath: string): Promise<void>;
  run(command: string, args: string[], cwd?: string): Promise<CommandResult>;
  randomId(): string;
}

const defaultDependencies: UpdateDependencies = {
  fetch,
  os: Deno.build.os,
  arch: Deno.build.arch,
  execPath: () => Deno.execPath(),
  makeTempDir: () => Deno.makeTempDir({ prefix: "toggl-update-" }),
  remove: (path, options) => Deno.remove(path, options),
  readFile: (path) => Deno.readFile(path),
  writeFile: (path, data) => Deno.writeFile(path, data, { createNew: true }),
  chmod: (path, mode) => Deno.chmod(path, mode),
  rename: (oldPath, newPath) => Deno.rename(oldPath, newPath),
  run: async (command, args, cwd) => {
    const output = await new Deno.Command(command, {
      args,
      cwd,
      stdout: "piped",
      stderr: "piped",
    }).output();
    return output;
  },
  randomId: () => crypto.randomUUID(),
};

export function defaultUpdateChannel(currentVersion: string): UpdateChannel {
  return /^nightly-\d{8}-[0-9a-f]+$/i.test(currentVersion)
    ? "nightly"
    : "stable";
}

export function releaseTarget(os: string, arch: string): string {
  if (os === "linux" && arch === "x86_64") return "linux-x64";
  if (os === "darwin" && arch === "aarch64") return "darwin-arm64";
  throw new Error(
    `Self-update is not supported on ${os}/${arch}; the existing binary was not changed.`,
  );
}

export function archiveName(
  channel: UpdateChannel,
  version: string,
  target: string,
): string {
  return channel === "nightly"
    ? `toggl-cli-nightly-${target}.tar.gz`
    : `toggl-cli-v${version}-${target}.tar.gz`;
}

async function json(response: Response, description: string): Promise<unknown> {
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
  ) {
    throw new Error(`GitHub returned an invalid ${description} response.`);
  }
  return (value as Record<string, string>)[key];
}

async function desiredVersion(
  channel: UpdateChannel,
  fetcher: typeof fetch,
): Promise<{ version: string; releaseTag: string }> {
  const headers = {
    Accept: "application/vnd.github+json",
    "X-GitHub-Api-Version": "2022-11-28",
  };
  if (channel === "stable") {
    const data = await json(
      await fetcher(
        "https://api.github.com/repos/mukopikmin/toggl-cli/releases/latest",
        { headers },
      ),
      "latest release",
    );
    const tag = field(data, "tag_name", "latest release");
    const version = tag.replace(/^v/, "");
    if (
      !/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/.test(version)
    ) {
      throw new Error("GitHub returned an invalid latest release tag.");
    }
    return { version, releaseTag: tag };
  }
  const ref = await json(
    await fetcher(
      "https://api.github.com/repos/mukopikmin/toggl-cli/git/ref/tags/nightly",
      { headers },
    ),
    "nightly tag",
  );
  const object = typeof ref === "object" && ref !== null
    ? (ref as Record<string, unknown>).object
    : undefined;
  const sha = field(object, "sha", "nightly tag");
  if (!/^[0-9a-f]{7,40}$/i.test(sha)) {
    throw new Error("GitHub returned an invalid nightly commit SHA.");
  }
  const commit = await json(
    await fetcher(
      `https://api.github.com/repos/mukopikmin/toggl-cli/commits/${sha}`,
      { headers },
    ),
    "nightly commit",
  );
  const commitData = typeof commit === "object" && commit !== null
    ? (commit as Record<string, unknown>).commit
    : undefined;
  const committer = typeof commitData === "object" && commitData !== null
    ? (commitData as Record<string, unknown>).committer
    : undefined;
  const date = field(committer, "date", "nightly commit");
  const parsed = new Date(date);
  if (Number.isNaN(parsed.valueOf())) {
    throw new Error("GitHub returned an invalid nightly commit date.");
  }
  return {
    version: `nightly-${
      parsed.toISOString().slice(0, 10).replaceAll("-", "")
    }-${sha.slice(0, 7)}`,
    releaseTag: "nightly",
  };
}

const hex = (bytes: Uint8Array): string =>
  [...bytes].map((byte) => byte.toString(16).padStart(2, "0")).join("");

export async function runUpdateCommand(
  options: { channel?: UpdateChannel; currentVersion: string },
  deps: UpdateDependencies = defaultDependencies,
): Promise<UpdateResult> {
  const executable = deps.execPath();
  if (/^deno(?:\.exe)?$/i.test(basename(executable))) {
    throw new Error(
      "Self-update is unavailable when running from source with Deno. Install a compiled toggl binary first.",
    );
  }
  const target = releaseTarget(deps.os, deps.arch);
  const channel = options.channel ??
    defaultUpdateChannel(options.currentVersion);
  const desired = await desiredVersion(channel, deps.fetch);
  const expected = desired.version;
  if (expected === options.currentVersion) {
    return { status: "current", version: expected };
  }

  const name = archiveName(channel, expected, target);
  const base =
    `https://github.com/mukopikmin/toggl-cli/releases/download/${desired.releaseTag}`;
  const [archiveResponse, checksumResponse] = await Promise.all([
    deps.fetch(`${base}/${name}`),
    deps.fetch(`${base}/${name}.sha256`),
  ]);
  if (!archiveResponse.ok || !checksumResponse.ok) {
    throw new Error("Failed to download the update archive or checksum.");
  }
  const archiveBytes = new Uint8Array(await archiveResponse.arrayBuffer());
  const checksum = (await checksumResponse.text()).trim().split(/\s+/)[0]
    ?.toLowerCase();
  const actual = hex(
    new Uint8Array(await crypto.subtle.digest("SHA-256", archiveBytes)),
  );
  if (!/^[0-9a-f]{64}$/.test(checksum ?? "") || checksum !== actual) {
    throw new Error(
      "Update checksum mismatch; the existing binary was not changed.",
    );
  }

  const tempDir = await deps.makeTempDir();
  const archivePath = join(tempDir, name);
  const stagedPath = join(
    dirname(executable),
    `.toggl-update-${deps.randomId()}`,
  );
  let staged = false;
  try {
    await deps.writeFile(archivePath, archiveBytes);
    const extraction = await deps.run("tar", [
      "-xzf",
      archivePath,
      "-C",
      tempDir,
    ]);
    if (!extraction.success) {
      throw new Error(
        "Failed to extract the update archive; the existing binary was not changed.",
      );
    }
    const extracted = join(tempDir, name.slice(0, -7), "toggl");
    const verification = await deps.run(extracted, ["--version"]);
    const output = new TextDecoder().decode(verification.stdout).trim();
    if (!verification.success || output !== expected) {
      throw new Error(
        `Downloaded binary version mismatch (expected ${expected}); the existing binary was not changed.`,
      );
    }
    staged = true;
    await deps.writeFile(stagedPath, await deps.readFile(extracted));
    await deps.chmod(stagedPath, 0o755);
    await deps.rename(stagedPath, executable);
    staged = false;
    return { status: "updated", version: expected };
  } finally {
    if (staged) await deps.remove(stagedPath).catch(() => undefined);
    await deps.remove(tempDir, { recursive: true }).catch(() => undefined);
  }
}
