import { parseArgs } from "@std/cli/parse-args";
import { join } from "@std/path";
import { getProjects } from "./toggl/projects.ts";
import { getTimeEntriesForDays } from "./toggl/time_entries.ts";
import type { TogglConfig } from "./toggl/types.ts";

const PREVIOUS_MONTH_FLAG = "-p";
const SEPARATOR = ",";

interface Config extends TogglConfig {
  [key: string]: string;
}

async function loadConfig(): Promise<Config> {
  const home = Deno.env.get("HOME");
  if (!home) {
    console.error("Error: HOME environment variable not set");
    Deno.exit(1);
  }
  const configFile = join(home, ".toggl_config");

  try {
    const text = await Deno.readTextFile(configFile);
    const config: Partial<Config> = {};

    for (const line of text.split("\n")) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith("#")) continue;

      if (trimmed.includes("=")) {
        const [key, value] = trimmed.split("=", 2);
        config[key.trim()] = value.trim();
      }
    }

    const requiredKeys = ["WORKSPACE", "TOKEN"];
    const missingKeys = requiredKeys.filter((key) => !config[key]);

    if (missingKeys.length > 0) {
      console.error(
        `Error: Missing required configuration in ~/.toggl_config: ${
          missingKeys.join(", ")
        }`,
      );
      Deno.exit(1);
    }

    return config as Config;
  } catch (error) {
    if (error instanceof Deno.errors.NotFound) {
      console.error("Error: ~/.toggl_config file not found");
      console.error("Please create ~/.toggl_config with the following format:");
      console.error("WORKSPACE=your_workspace_id");
      console.error("TOKEN=your_api_token");
      Deno.exit(1);
    }
    throw error;
  }
}

if (import.meta.main) {
  const args = Deno.args;

  if (args.length < 2) {
    console.log(
      `Usage: deno run --allow-net --allow-read --allow-env main.ts <start_day> <end_day> [${PREVIOUS_MONTH_FLAG}]`,
    );
    console.log(
      "Example: deno run --allow-net --allow-read --allow-env main.ts 1 15",
    );
    console.log(
      `Example: deno run --allow-net --allow-read --allow-env main.ts 1 15 ${PREVIOUS_MONTH_FLAG}`,
    );
    Deno.exit(1);
  }

  const startDay = parseInt(args[0], 10);
  const endDay = parseInt(args[1], 10);
  const usePreviousMonth = args.includes(PREVIOUS_MONTH_FLAG);

  const now = new Date();
  let targetYear = now.getFullYear();
  let targetMonth = now.getMonth() + 1; // 1-12

  if (usePreviousMonth) {
    targetMonth -= 1;
    if (targetMonth === 0) {
      targetMonth = 12;
      targetYear -= 1;
    }
  }

  const config = await loadConfig();
  const projects = await getProjects(config);

  // Generate days array
  const days: Date[] = [];

  const current = new Date(targetYear, targetMonth - 1, startDay);
  const end = new Date(targetYear, targetMonth - 1, endDay);

  for (let d = new Date(current); d <= end; d.setDate(d.getDate() + 1)) {
    days.push(new Date(d));
  }

  const dateEntries = await getTimeEntriesForDays(
    config,
    startDay,
    endDay,
    targetYear,
    targetMonth,
  );

  console.log("--- Project list ---");
  for (const p of projects) {
    console.log(p.name);
  }
  console.log();
  console.log(
    "--- Work time table (The order of the rows follows the projects list) ---",
  );

  const header = days.map((d) => {
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, "0");
    const day = String(d.getDate()).padStart(2, "0");
    return `${y}-${m}-${day}`;
  }).join(SEPARATOR);
  console.log(header);

  for (const project of projects) {
    const row = days.map((d) => {
      const dayNum = d.getDate();
      const duration = dateEntries[dayNum]?.[project.id];
      return duration ? duration.toString() : "";
    }).join(SEPARATOR);
    console.log(row);
  }
}
