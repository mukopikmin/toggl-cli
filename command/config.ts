import { loadConfig } from "../config.ts";
import type { Config } from "../config.ts";

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

export async function runConfigCommand(format: "csv" | "json") {
  const config = await loadConfig();
  console.log(
    format === "json" ? formatConfigJson(config) : formatConfigValues(config),
  );
}
