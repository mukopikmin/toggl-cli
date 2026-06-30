export interface ClipboardCommand {
  command: string;
  args: string[];
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
  const errors: string[] = [];

  for (const command of clipboardCommands()) {
    try {
      await writeClipboardWithCommand(text, command);
      return;
    } catch (error) {
      errors.push(error instanceof Error ? error.message : String(error));
    }
  }

  throw new Error(
    `Could not copy output to the clipboard.${
      errors.length > 0 ? ` Tried: ${errors.join("; ")}` : ""
    }`,
  );
}
