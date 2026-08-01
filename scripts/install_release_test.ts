import { assert, assertEquals, assertStringIncludes } from "@std/assert";
import { join } from "@std/path";

const projectRoot = new URL("../", import.meta.url).pathname;

const sha256 = async (path: string): Promise<string> => {
  const digest = await crypto.subtle.digest(
    "SHA-256",
    await Deno.readFile(path),
  );
  return [...new Uint8Array(digest)].map((byte) =>
    byte.toString(16).padStart(2, "0")
  ).join("");
};

const makeFixture = async (root: string, version: string, target: string) => {
  const archiveRoot = version === "nightly"
    ? `toggl-cli-nightly-${target}`
    : `toggl-cli-v${version}-${target}`;
  const sourceDir = join(root, archiveRoot);
  await Deno.mkdir(sourceDir, { recursive: true });
  await Deno.writeTextFile(join(sourceDir, "toggl"), `toggl ${version}\n`);

  const archive = `${archiveRoot}.tar.gz`;
  const archivePath = join(root, archive);
  const result = await new Deno.Command("tar", {
    args: ["-czf", archivePath, "-C", root, archiveRoot],
  }).output();
  assert(result.success);
  await Deno.writeTextFile(
    `${archivePath}.sha256`,
    `${await sha256(archivePath)}\n`,
  );
};

const writeCommands = async (binDir: string) => {
  await Deno.mkdir(binDir);
  const curl = `#!/bin/sh
output=""
write_effective=false
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) output="$2"; shift 2 ;;
    -w) write_effective=true; shift 2 ;;
    -*) shift ;;
    *) url="$1"; shift ;;
  esac
done
if $write_effective; then
  printf '%s' "https://github.com/mukopikmin/toggl-cli/releases/tag/$FAKE_RELEASE_TAG"
  exit 0
fi
cp "$FAKE_RELEASE_DIR/\${url##*/}" "$output"
`;
  const uname = `#!/bin/sh
case "$1" in
  -s) printf '%s\n' "$FAKE_UNAME_S" ;;
  -m) printf '%s\n' "$FAKE_UNAME_M" ;;
esac
`;
  await Deno.writeTextFile(join(binDir, "curl"), curl, { mode: 0o755 });
  await Deno.writeTextFile(join(binDir, "uname"), uname, { mode: 0o755 });
};

const runInstaller = async (
  options: {
    args?: string[];
    corruptChecksum?: boolean;
    existingBinary?: string;
    os?: string;
    arch?: string;
    releaseTag?: string;
  } = {},
) => {
  const root = await Deno.makeTempDir({
    prefix: "toggl-cli-release-install-test-",
  });
  const home = join(root, "home");
  const fixtures = join(root, "fixtures");
  const bin = join(root, "bin");
  await Deno.mkdir(home);
  if (options.existingBinary !== undefined) {
    await Deno.mkdir(join(home, ".local", "bin"), { recursive: true });
    await Deno.writeTextFile(
      join(home, ".local", "bin", "toggl"),
      options.existingBinary,
    );
  }
  await Deno.mkdir(fixtures);
  await writeCommands(bin);
  await makeFixture(fixtures, "1.2.3", "linux-x64");
  await makeFixture(fixtures, "nightly", "darwin-arm64");
  if (options.corruptChecksum) {
    await Deno.writeTextFile(
      join(fixtures, "toggl-cli-v1.2.3-linux-x64.tar.gz.sha256"),
      `${"0".repeat(64)}\n`,
    );
  }

  const command = new Deno.Command("sh", {
    args: [join(projectRoot, "install.sh"), ...(options.args ?? [])],
    env: {
      HOME: home,
      PATH: `${bin}:${Deno.env.get("PATH") ?? ""}`,
      FAKE_RELEASE_DIR: fixtures,
      FAKE_RELEASE_TAG: options.releaseTag ?? "v1.2.3",
      FAKE_UNAME_S: options.os ?? "Linux",
      FAKE_UNAME_M: options.arch ?? "x86_64",
    },
    stdout: "piped",
    stderr: "piped",
  });
  const output = await command.output();
  return {
    root,
    home,
    output,
    stdout: new TextDecoder().decode(output.stdout),
    stderr: new TextDecoder().decode(output.stderr),
  };
};

Deno.test("installs the latest release for Linux x64", async () => {
  const result = await runInstaller();
  try {
    assert(result.output.success, result.stderr);
    const installed = join(result.home, ".local", "bin", "toggl");
    assertEquals(await Deno.readTextFile(installed), "toggl 1.2.3\n");
    assertEquals((await Deno.stat(installed)).mode! & 0o777, 0o755);
    assertStringIncludes(result.stdout, "Installed toggl-cli 1.2.3");
  } finally {
    await Deno.remove(result.root, { recursive: true });
  }
});

Deno.test("accepts a stable tag without a leading v", async () => {
  const result = await runInstaller({ releaseTag: "1.2.3" });
  try {
    assert(result.output.success, result.stderr);
    assertEquals(
      await Deno.readTextFile(join(result.home, ".local", "bin", "toggl")),
      "toggl 1.2.3\n",
    );
  } finally {
    await Deno.remove(result.root, { recursive: true });
  }
});

Deno.test("installs the nightly release for macOS arm64", async () => {
  const result = await runInstaller({
    args: ["--nightly"],
    os: "Darwin",
    arch: "arm64",
  });
  try {
    assert(result.output.success, result.stderr);
    assertEquals(
      await Deno.readTextFile(join(result.home, ".local", "bin", "toggl")),
      "toggl nightly\n",
    );
  } finally {
    await Deno.remove(result.root, { recursive: true });
  }
});

Deno.test("does not replace an existing binary when checksum verification fails", async () => {
  const result = await runInstaller({
    corruptChecksum: true,
    existingBinary: "existing toggl\n",
  });
  try {
    assertEquals(result.output.success, false);
    assertStringIncludes(result.stderr, "SHA-256 checksum mismatch");
    assertEquals(
      await Deno.readTextFile(join(result.home, ".local", "bin", "toggl")),
      "existing toggl\n",
    );
  } finally {
    await Deno.remove(result.root, { recursive: true });
  }
});

Deno.test("rejects unsupported platforms and unknown arguments", async () => {
  const unsupported = await runInstaller({ os: "Linux", arch: "aarch64" });
  const badArgument = await runInstaller({ args: ["--version"] });
  try {
    assertEquals(unsupported.output.success, false);
    assertStringIncludes(
      unsupported.stderr,
      "unsupported platform: Linux aarch64",
    );
    assertEquals(badArgument.output.success, false);
    assertStringIncludes(badArgument.stderr, "Usage: install.sh [--nightly]");
  } finally {
    await Promise.all([
      Deno.remove(unsupported.root, { recursive: true }),
      Deno.remove(badArgument.root, { recursive: true }),
    ]);
  }
});
