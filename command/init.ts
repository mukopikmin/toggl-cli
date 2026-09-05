import { dirname } from "node:path";
import { stringify } from "@std/toml";
import { CONFIG_FILE_DISPLAY, getConfigFile } from "../config.ts";

const DEFAULT_TIMEZONE = "Asia/Tokyo";

export type InitialConfig = {
  workspace: string;
  token: string;
  timezone?: string;
};

export interface InitInput {
  readonly interactive: boolean;
  readLine(secret?: boolean): Promise<string | null>;
  write(text: string): Promise<void>;
}

export class InitInputError extends Error {
  override name = "InitInputError";
}

export function createConfigToml(config: InitialConfig): string {
  return stringify({
    workspace: config.workspace,
    token: config.token,
    timezone: config.timezone?.trim() || DEFAULT_TIMEZONE,
  });
}

export function createConfigTemplate(): string {
  return `${
    createConfigToml({
      workspace: "your_workspace_id",
      token: "your_api_token",
      timezone: DEFAULT_TIMEZONE,
    })
  }
[projects.123456]
display_name = "Client A"
hidden = false
display_order = 10

[projects.234567]
hidden = true
display_order = 20
`;
}

async function assertConfigFileDoesNotExist(configFile: string): Promise<void> {
  try {
    await Deno.stat(configFile);
    console.error(`Error: ${CONFIG_FILE_DISPLAY} already exists`);
    Deno.exit(1);
  } catch (error) {
    if (!(error instanceof Deno.errors.NotFound)) {
      throw error;
    }
  }
}

async function writeAll(bytes: Uint8Array): Promise<void> {
  const writer = Deno.stdout.writable.getWriter();
  try {
    await writer.write(bytes);
  } finally {
    writer.releaseLock();
  }
}

async function writeText(text: string): Promise<void> {
  await writeAll(new TextEncoder().encode(text));
}

async function readInputLine(secret = false): Promise<string | null> {
  if (secret && Deno.stdin.isTerminal()) {
    Deno.stdin.setRaw(true);
  }

  const buffer = new Uint8Array(1);
  const chunks: number[] = [];
  let reachedEof = false;

  try {
    while (true) {
      const bytesRead = await Deno.stdin.read(buffer);
      if (bytesRead === null) {
        reachedEof = true;
        break;
      }

      const byte = buffer[0];
      if (byte === 3) {
        await writeText("\n");
        Deno.exit(130);
      }
      if (byte === 10 || byte === 13) {
        if (secret && Deno.stdin.isTerminal()) {
          await writeText("\n");
        }
        break;
      }
      if (byte === 127 || byte === 8) {
        chunks.pop();
        continue;
      }

      chunks.push(byte);
    }
  } finally {
    if (secret && Deno.stdin.isTerminal()) {
      Deno.stdin.setRaw(false);
    }
  }

  if (reachedEof && chunks.length === 0) return null;
  return new TextDecoder().decode(new Uint8Array(chunks)).trim();
}

const denoInput: InitInput = {
  interactive: Deno.stdin.isTerminal(),
  readLine: readInputLine,
  write: writeText,
};

async function promptForValue(
  input: InitInput,
  label: string,
  options: { defaultValue?: string; required?: boolean; secret?: boolean } = {},
): Promise<string> {
  while (true) {
    const suffix = options.defaultValue ? ` [${options.defaultValue}]` : "";
    await input.write(`${label}${suffix}: `);
    const value = await input.readLine(options.secret);
    if (value === null) {
      throw new InitInputError(`Input ended before ${label} was provided`);
    }
    if (value.trim()) return value.trim();
    if (options.defaultValue) return options.defaultValue;
    if (!options.required) return "";
    if (!input.interactive) {
      throw new InitInputError(`${label} is required`);
    }
    await input.write(`${label} is required. Please enter a value.\n`);
  }
}

export async function promptForInitialConfig(
  input: InitInput,
): Promise<InitialConfig> {
  return {
    workspace: await promptForValue(input, "Workspace", { required: true }),
    token: await promptForValue(input, "API token", {
      required: true,
      secret: true,
    }),
    timezone: await promptForValue(input, "Timezone", {
      defaultValue: DEFAULT_TIMEZONE,
    }),
  };
}

export async function writeInitialConfig(
  configFile: string,
  configToml = createConfigTemplate(),
): Promise<void> {
  await assertConfigFileDoesNotExist(configFile);
  await Deno.mkdir(dirname(configFile), { recursive: true });
  await Deno.writeTextFile(configFile, configToml);
}

export async function runInitCommand(
  input: InitInput = denoInput,
): Promise<void> {
  const home = Deno.env.get("HOME");
  if (!home) {
    console.error("Error: HOME environment variable not set");
    Deno.exit(1);
  }

  const configFile = getConfigFile(home);
  await assertConfigFileDoesNotExist(configFile);
  const config = await promptForInitialConfig(input);
  await Deno.mkdir(dirname(configFile), { recursive: true });
  await Deno.writeTextFile(configFile, createConfigToml(config));
  console.log(`Wrote ${CONFIG_FILE_DISPLAY}`);
}
