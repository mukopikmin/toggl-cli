if (Deno.build.os !== "darwin") {
  console.error("This installer is intended for macOS.");
  Deno.exit(1);
}

const home = Deno.env.get("HOME");
if (!home) {
  console.error("HOME is not set.");
  Deno.exit(1);
}

const run = async (args: string[]): Promise<void> => {
  const result = await new Deno.Command(Deno.execPath(), {
    args,
    stdout: "inherit",
    stderr: "inherit",
  }).output();

  if (!result.success) {
    Deno.exit(result.code);
  }
};

const installDir = `${home}/.local/bin`;
const installPath = `${installDir}/toggl`;
const buildDir = await Deno.makeTempDir({ prefix: "toggl-" });
const buildPath = `${buildDir}/toggl`;

try {
  await run([
    "compile",
    "--allow-net",
    "--allow-read",
    "--allow-env",
    ...Deno.args,
    "--output",
    buildPath,
    "main.ts",
  ]);

  await Deno.mkdir(installDir, { recursive: true });
  await Deno.copyFile(buildPath, installPath);
  await Deno.chmod(installPath, 0o755);

  console.log(`Installed toggl to ${installPath}`);
} finally {
  await Deno.remove(buildDir, { recursive: true }).catch(() => undefined);
}
