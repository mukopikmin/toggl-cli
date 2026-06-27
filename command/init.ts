import { dirname } from "node:path";
import { stringify } from "@std/toml";
import { CONFIG_FILE_DISPLAY, getConfigFile } from "../config.ts";

export function createConfigTemplate(): string {
  return stringify({
    workspace: "your_workspace_id",
    token: "your_api_token",
    timezone: "Asia/Tokyo",
    projects: {
      "123456": {
        display_name: "Client A",
        hidden: false,
        display_order: 10,
      },
      "234567": {
        hidden: true,
        display_order: 20,
      },
    },
  });
}

export async function writeInitialConfig(configFile: string): Promise<void> {
  try {
    await Deno.stat(configFile);
    console.error(`Error: ${CONFIG_FILE_DISPLAY} already exists`);
    Deno.exit(1);
  } catch (error) {
    if (!(error instanceof Deno.errors.NotFound)) {
      throw error;
    }
  }

  await Deno.mkdir(dirname(configFile), { recursive: true });
  await Deno.writeTextFile(configFile, createConfigTemplate());
}

export async function runInitCommand(): Promise<void> {
  const home = Deno.env.get("HOME");
  if (!home) {
    console.error("Error: HOME environment variable not set");
    Deno.exit(1);
  }

  await writeInitialConfig(getConfigFile(home));
  console.log(`Wrote ${CONFIG_FILE_DISPLAY}`);
}
