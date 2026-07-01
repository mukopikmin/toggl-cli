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
    try {
      await writeClipboardWithCommand(text, command);
      return;
    } catch {
      // Try the next platform-specific clipboard command.
    }
  }

  throw new ClipboardUnavailableError();
}
