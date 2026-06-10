import { join } from "@std/path";
import { stringify } from "@std/toml";
import { CONFIG_FILE_DISPLAY, getConfigFile } from "../config.ts";

interface LegacyConfig {
  workspace?: string;
  token?: string;
  projects: Record<string, { display_name: string }>;
}

function parseLegacyConfig(text: string): LegacyConfig {
  const config: LegacyConfig = {
    projects: {},
  };

  for (const line of text.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#") || !trimmed.includes("=")) {
      continue;
    }

    const [key, value] = trimmed.split("=", 2);
    const normalizedKey = key.trim();
    const normalizedValue = value.trim();

    if (normalizedKey === "WORKSPACE") {
      config.workspace = normalizedValue;
      continue;
    }

    if (normalizedKey === "TOKEN") {
      config.token = normalizedValue;
      continue;
    }

    if (normalizedKey.startsWith("PROJECT_NAME_")) {
      const projectId = normalizedKey.slice("PROJECT_NAME_".length);
      if (projectId) {
        config.projects[projectId] = {
          display_name: normalizedValue,
        };
      }
    }
  }

  return config;
}

function configToToml(config: LegacyConfig): string {
  const output: Record<string, unknown> = {
    workspace: config.workspace ?? "",
    token: config.token ?? "",
  };

  if (Object.keys(config.projects).length > 0) {
    output.projects = config.projects;
  }

  return stringify(output);
}

const home = Deno.env.get("HOME");
if (!home) {
  console.error("Error: HOME environment variable not set");
  Deno.exit(1);
}

const legacyFile = join(home, ".toggl_config");
const tomlFile = getConfigFile(home);

try {
  await Deno.stat(tomlFile);
  console.error(`Error: ${CONFIG_FILE_DISPLAY} already exists`);
  console.error("Remove it first if you want to overwrite it.");
  Deno.exit(1);
} catch (error) {
  if (!(error instanceof Deno.errors.NotFound)) {
    throw error;
  }
}

try {
  const legacyText = await Deno.readTextFile(legacyFile);
  const config = parseLegacyConfig(legacyText);
  await Deno.mkdir(join(home, ".config", "toggl-cli"), { recursive: true });
  await Deno.writeTextFile(tomlFile, configToToml(config));
  console.log(`Wrote ${tomlFile}`);
} catch (error) {
  if (error instanceof Deno.errors.NotFound) {
    console.error("Error: ~/.toggl_config file not found");
    Deno.exit(1);
  }
  throw error;
}
