import { loadConfig } from "../config.ts";
import type { Config } from "../config.ts";
import type { OutputFormat } from "./output_format.ts";
import { formatTable } from "./table.ts";

export function withoutSensitiveConfig(config: Config): Record<string, string> {
  const visibleConfig: Record<string, string> = {};

  for (const [key, value] of Object.entries(config)) {
    if (key === "TOKEN" || typeof value !== "string") continue;
    visibleConfig[key] = value;
  }

  return visibleConfig;
}

export function formatConfigValues(config: Config): string {
  return Object.entries(withoutSensitiveConfig(config))
    .map(([key, value]) => `${key}=${value}`)
    .join("\n");
}

export function formatConfigJson(config: Config): string {
  return JSON.stringify(withoutSensitiveConfig(config), null, 2);
}

export function formatConfigTable(config: Config): string {
  return formatTable(
    ["Setting", "Value"],
    Object.entries(withoutSensitiveConfig(config)),
  );
}

export async function runConfigCommand(format: OutputFormat) {
  const config = await loadConfig();
  console.log(
    format === "json"
      ? formatConfigJson(config)
      : format === "table"
      ? formatConfigTable(config)
      : formatConfigValues(config),
  );
}
