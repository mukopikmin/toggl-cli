type CommandResult = {
  code: number;
  stdout: string;
  stderr: string;
};

export type CommandRunner = (
  command: "git" | "gh",
  args: string[],
) => Promise<CommandResult>;

type Release = {
  tagName: string;
  isDraft: boolean;
  isPrerelease: boolean;
};

type ReleaseContext = {
  version: string;
  tag: string;
  targetSha: string;
  previousTag: string;
  repository: string;
};

const stableVersion = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/;
const semanticVersion =
  /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$|^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$/;

export const validateVersion = (version: string): void => {
  if (!semanticVersion.test(version)) {
    throw new Error(`Invalid semantic version: ${version}`);
  }
  const prerelease = version.split("-", 2)[1]?.split("+", 1)[0];
  if (
    prerelease?.split(".").some((part) =>
      /^\d+$/.test(part) &&
      (part.length > 1 && part.startsWith("0"))
    )
  ) {
    throw new Error(`Invalid semantic version: ${version}`);
  }
};

export const systemRunner: CommandRunner = async (command, args) => {
  try {
    const result = await new Deno.Command(command, {
      args,
      stdout: "piped",
      stderr: "piped",
    }).output();
    return {
      code: result.code,
      stdout: new TextDecoder().decode(result.stdout),
      stderr: new TextDecoder().decode(result.stderr),
    };
  } catch (error) {
    throw new Error(
      `Could not run ${command}: ${
        error instanceof Error ? error.message : String(error)
      }`,
    );
  }
};

const run = async (
  runner: CommandRunner,
  command: "git" | "gh",
  args: string[],
): Promise<string> => {
  const result = await runner(command, args);
  if (result.code !== 0) {
    const detail = result.stderr.trim() || result.stdout.trim() ||
      `exit code ${result.code}`;
    throw new Error(`${command} ${args.join(" ")} failed: ${detail}`);
  }
  return result.stdout.trim();
};

const readJson = <T>(value: string, source: string): T => {
  try {
    return JSON.parse(value) as T;
  } catch {
    throw new Error(`${source} returned invalid JSON.`);
  }
};

const isNightly = (tag: string): boolean =>
  /^v?\d+\.\d+\.\d+-nightly(?:\.|$)/i.test(tag);

const findPreviousTag = async (
  runner: CommandRunner,
  releases: Release[],
): Promise<string> => {
  const previous = releases.find((release) =>
    !release.isDraft && !release.isPrerelease && !isNightly(release.tagName) &&
    stableVersion.test(release.tagName.replace(/^v/, ""))
  );
  if (previous) return previous.tagName;

  const baseline = "release-notes-baseline";
  const baselineSha = await run(runner, "git", [
    "rev-list",
    "-n",
    "1",
    baseline,
  ]);
  const roots = (await run(runner, "git", [
    "rev-list",
    "--max-parents=0",
    "HEAD",
  ])).split("\n").filter(Boolean);
  if (roots.length !== 1 || baselineSha !== roots[0]) {
    throw new Error(
      `${baseline} must point to the repository root commit (${
        roots.join(", ")
      }).`,
    );
  }
  return baseline;
};

const prepare = async (
  version: string,
  runner: CommandRunner,
): Promise<ReleaseContext> => {
  validateVersion(version);
  const tag = `v${version}`;
  const branch = await run(runner, "git", ["branch", "--show-current"]);
  if (branch !== "main") {
    throw new Error(`Current branch is '${branch}', not 'main'.`);
  }

  const status = await run(runner, "git", ["status", "--porcelain"]);
  if (status) throw new Error("Working tree is not clean.");

  const targetSha = await run(runner, "git", ["rev-parse", "HEAD"]);
  const originSha = await run(runner, "git", ["rev-parse", "origin/main"]);
  if (targetSha !== originSha) {
    throw new Error(
      `HEAD (${targetSha}) does not match origin/main (${originSha}).`,
    );
  }
  const divergence = await run(runner, "git", [
    "rev-list",
    "--left-right",
    "--count",
    "HEAD...origin/main",
  ]);
  const [ahead, behind] = divergence.split(/\s+/).map(Number);
  if (!Number.isInteger(ahead) || !Number.isInteger(behind)) {
    throw new Error(
      `git returned an invalid ahead/behind count: ${divergence}`,
    );
  }
  if (ahead !== 0 || behind !== 0) {
    throw new Error(
      `main is ${ahead} commit(s) ahead and ${behind} commit(s) behind origin/main.`,
    );
  }

  const localTags = (await run(runner, "git", ["tag", "--list", tag])).split(
    "\n",
  );
  if (localTags.includes(tag)) throw new Error(`Tag ${tag} already exists.`);
  const remoteTags = await run(runner, "git", [
    "ls-remote",
    "--tags",
    "origin",
    `refs/tags/${tag}`,
    `refs/tags/${tag}^{}`,
  ]);
  if (remoteTags) throw new Error(`Tag ${tag} already exists on origin.`);

  const releases = readJson<Release[]>(
    await run(runner, "gh", [
      "release",
      "list",
      "--limit",
      "100",
      "--json",
      "tagName,isDraft,isPrerelease",
    ]),
    "gh release list",
  );
  if (releases.some((release) => release.tagName === tag)) {
    throw new Error(`Release ${tag} already exists.`);
  }

  const runs = readJson<
    Array<{ headSha: string; status: string; conclusion: string }>
  >(
    await run(runner, "gh", [
      "run",
      "list",
      "--workflow",
      ".github/workflows/test.yml",
      "--commit",
      targetSha,
      "--limit",
      "100",
      "--json",
      "headSha,status,conclusion",
    ]),
    "gh run list",
  );
  if (
    !runs.some((item) =>
      item.headSha === targetSha && item.status === "completed" &&
      item.conclusion === "success"
    )
  ) {
    throw new Error(`The Test workflow has not succeeded for ${targetSha}.`);
  }

  const repository = readJson<{ nameWithOwner: string }>(
    await run(runner, "gh", ["repo", "view", "--json", "nameWithOwner"]),
    "gh repo view",
  ).nameWithOwner;
  if (!repository) {
    throw new Error("gh repo view did not return a repository name.");
  }
  const previousTag = await findPreviousTag(runner, releases);
  return { version, tag, targetSha, previousTag, repository };
};

export const checkRelease = async (
  version: string,
  runner: CommandRunner = systemRunner,
): Promise<Record<string, unknown>> => {
  const context = await prepare(version, runner);
  return { ok: true, ...context };
};

export const generateNotes = async (
  version: string,
  runner: CommandRunner = systemRunner,
): Promise<Record<string, unknown>> => {
  const context = await prepare(version, runner);
  const generated = readJson<{ name: string; body: string }>(
    await run(runner, "gh", [
      "api",
      "--method",
      "POST",
      `repos/${context.repository}/releases/generate-notes`,
      "-f",
      `tag_name=${context.tag}`,
      "-f",
      `target_commitish=${context.targetSha}`,
      "-f",
      `previous_tag_name=${context.previousTag}`,
    ]),
    "gh release notes generation",
  );
  if (
    typeof generated.name !== "string" || typeof generated.body !== "string"
  ) {
    throw new Error("GitHub returned incomplete generated release notes.");
  }
  return {
    ok: true,
    version: context.version,
    tag: context.tag,
    title: generated.name,
    body: generated.body,
    previousTag: context.previousTag,
    targetSha: context.targetSha,
  };
};

const parseArgs = (
  args: string[],
): { command: string; version: string; output?: string } => {
  const command = args[0];
  if (command !== "check" && command !== "notes") {
    throw new Error(
      "Usage: release.ts <check|notes> --version <version> [--output <path>]",
    );
  }
  const versionIndex = args.indexOf("--version");
  const version = versionIndex >= 0 ? args[versionIndex + 1] : undefined;
  if (!version || version.startsWith("--")) {
    throw new Error("Missing required --version <version> option.");
  }
  const outputIndex = args.indexOf("--output");
  const output = outputIndex >= 0 ? args[outputIndex + 1] : undefined;
  if (outputIndex >= 0 && (!output || output.startsWith("--"))) {
    throw new Error("Missing value for --output.");
  }
  if (command === "check" && output) {
    throw new Error("--output is only supported by notes.");
  }
  return { command, version, output };
};

if (import.meta.main) {
  try {
    const options = parseArgs(Deno.args);
    const result = options.command === "check"
      ? await checkRelease(options.version)
      : await generateNotes(options.version);
    const json = `${JSON.stringify(result, null, 2)}\n`;
    if (options.output) await Deno.writeTextFile(options.output, json);
    else await Deno.stdout.write(new TextEncoder().encode(json));
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    Deno.exit(1);
  }
}
