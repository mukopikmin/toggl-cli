const versionPath = new URL("../version.ts", import.meta.url);

const parseOutputPath = (args: string[]): string => {
  const outputIndex = args.indexOf("--output");
  if (outputIndex === -1) return "toggl";

  const outputPath = args[outputIndex + 1];
  if (!outputPath) {
    console.error("Missing value for --output.");
    Deno.exit(1);
  }

  return outputPath;
};

const parseTarget = (args: string[]): string | undefined => {
  const targetIndex = args.indexOf("--target");
  if (targetIndex === -1) return undefined;

  const target = args[targetIndex + 1];
  if (!target) {
    console.error("Missing value for --target.");
    Deno.exit(1);
  }

  return target;
};

const parseVersion = (args: string[]): string | undefined => {
  const versionIndex = args.indexOf("--version");
  if (versionIndex === -1) return undefined;

  const version = args[versionIndex + 1];
  if (!version) {
    console.error("Missing value for --version.");
    Deno.exit(1);
  }

  return version;
};

const versionSource = (version: string): string =>
  `export const version = ${JSON.stringify(version)};\n`;

const outputPath = parseOutputPath(Deno.args);
const target = parseTarget(Deno.args);
const version = parseVersion(Deno.args);
const targetArgs = target ? ["--target", target] : [];
const originalVersionSource = await Deno.readTextFile(versionPath);

const command = new Deno.Command(Deno.execPath(), {
  args: [
    "compile",
    "--quiet",
    "--node-modules-dir=none",
    "--no-check",
    "--allow-read",
    "--allow-write",
    "--allow-net",
    "--allow-run=pbcopy,wl-copy,xclip,xsel,clip,powershell.exe,powershell",
    "--allow-env",
    ...targetArgs,
    "--output",
    outputPath,
    "main.ts",
  ],
  stdout: "inherit",
  stderr: "inherit",
});

let exitCode = 0;

try {
  if (version) await Deno.writeTextFile(versionPath, versionSource(version));
  const result = await command.output();
  if (!result.success) exitCode = result.code;
} finally {
  if (version) await Deno.writeTextFile(versionPath, originalVersionSource);
}

if (exitCode !== 0) Deno.exit(exitCode);
