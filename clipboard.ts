export interface ClipboardCommand {
  command: string;
  args: string[];
}

export class ClipboardUnavailableError extends Error {
  constructor() {
    super("Could not copy output to the clipboard.");
    this.name = "ClipboardUnavailableError";
  }
}

function clipboardCommands(): ClipboardCommand[] {
  switch (Deno.build.os) {
    case "darwin":
      return [{ command: "pbcopy", args: [] }];
    case "windows":
      return [
        { command: "clip", args: [] },
        {
          command: "powershell.exe",
          args: ["-NoProfile", "-Command", "Set-Clipboard"],
        },
        {
          command: "powershell",
          args: ["-NoProfile", "-Command", "Set-Clipboard"],
        },
      ];
    case "linux":
      return [
        { command: "wl-copy", args: [] },
        { command: "xclip", args: ["-selection", "clipboard"] },
        { command: "xsel", args: ["--clipboard", "--input"] },
      ];
    default:
      return [];
  }
}

async function commandExists(command: string): Promise<boolean> {
  const path = Deno.env.get("PATH");
  if (!path) return false;

  const windows = Deno.build.os === "windows";
  const separator = windows ? "\\" : "/";
  const extensions = windows
    ? ["", ...(Deno.env.get("PATHEXT") ?? ".COM;.EXE;.BAT;.CMD").split(";")]
    : [""];

  for (const directory of path.split(windows ? ";" : ":")) {
    if (!directory) continue;

    for (const extension of extensions) {
      const candidate = `${directory}${separator}${command}${extension}`;
      try {
        const stat = await Deno.stat(candidate);
        if (
          stat.isFile && (windows || stat.mode === null || (stat.mode & 0o111))
        ) {
          return true;
        }
      } catch {
        // Unreadable and missing PATH entries are not usable commands.
      }
    }
  }

  return false;
}

async function writeClipboardWithCommand(
  text: string,
  { command, args }: ClipboardCommand,
): Promise<void> {
  const child = new Deno.Command(command, {
    args,
    stdin: "piped",
    stdout: "null",
    stderr: "piped",
  }).spawn();

  const writer = child.stdin.getWriter();
  await writer.write(new TextEncoder().encode(text));
  await writer.close();

  const result = await child.output();
  if (!result.success) {
    const stderr = new TextDecoder().decode(result.stderr).trim();
    throw new Error(
      `${command} exited with code ${result.code}${
        stderr ? `: ${stderr}` : ""
      }`,
    );
  }
}

export async function writeClipboardText(text: string): Promise<void> {
  for (const command of clipboardCommands()) {
    if (!(await commandExists(command.command))) continue;

    try {
      await writeClipboardWithCommand(text, command);
      return;
    } catch {
      // Try the next platform-specific clipboard command.
    }
  }

  throw new ClipboardUnavailableError();
}
