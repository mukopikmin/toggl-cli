import { assertEquals, assertRejects, assertThrows } from "@std/assert";
import { ClipboardUnavailableError } from "./clipboard.ts";
import {
  CliUsageError,
  createHelpText,
  HELP_TEXT,
  parseCliArgs,
} from "./cli.ts";
import { createConfigTemplate, createConfigToml } from "./command/init.ts";
import {
  appendMissingProjects,
  formatProjectList,
  formatProjectsJson,
} from "./command/projects.ts";
import {
  formatConfigJson,
  formatConfigValues,
  withoutSensitiveConfig,
} from "./command/config.ts";
import {
  buildWorkTimeTable,
  formatTimeEntriesJson,
  formatWorkTimeTable,
  outputSummaryText,
  resolveSummaryDateRange,
} from "./command/summary.ts";
import {
  ConfigValidationError,
  loadConfigDocument,
  parseConfigToml,
  parseProjectsConfig,
} from "./config.ts";
import { main } from "./main.ts";
import { getTimeEntries } from "./toggl/time_entries.ts";
import { formatTimeEntryDate, resolveTimeZone } from "./model/date.ts";
import { TogglApiError } from "./toggl/error.ts";
  parseConfigToml,
  parseProjectsConfig,
} from "./config.ts";
import { main } from "./main.ts";
import { getProjects } from "./toggl/projects.ts";
import { getSummaryTimeEntries } from "./toggl/summary.ts";
import {
  apiEndpoint,
  getTimeEntriesForDays,
  reportsApiEndpoint,
} from "./toggl/api.ts";
import { formatTimeEntryDate, resolveTimeZone } from "./toggl/date.ts";
import { TogglApiError } from "./toggl/error.ts";

const config = {
  WORKSPACE: "workspace-id",
  TOKEN: "test-token",
};

Deno.test("parseCliArgs returns help for the root command", () => {
  assertEquals(parseCliArgs([]), { name: "help" });
  assertEquals(HELP_TEXT.includes("toggl summary"), true);
  assertEquals(HELP_TEXT.includes("toggl config"), true);
  assertEquals(HELP_TEXT.includes("-h, --help"), true);
  assertEquals(HELP_TEXT.includes("--clipboard"), true);
});

Deno.test("createHelpText describes commands and options", () => {
  assertEquals(
    createHelpText(),
    `Usage:
  toggl summary <start-date> <end-date> [options]
  toggl summary --days <days> [options]
  toggl projects [options]
  toggl projects sync
  toggl config [options]
  toggl init
  toggl update [--channel stable|nightly]

Commands:
  init      Create the configuration file
  projects  List projects
  config    Show configuration values
  summary   Summarize time entries for a range of days
  update    Update the installed Toggl CLI binary

Options:
  -s, --separator <text> Set the output delimiter (default: tab)
  -f, --format <format>  Set the output format: csv or json (default: csv)
  -d, --days <days>      Aggregate from this many days ago through today
      --clipboard        Copy the output to the clipboard as well as stdout
  -h, --help             Show this help
      --no-project       Omit the project column from CSV output
      --no-date          Omit the date header row from CSV output
      --version          Show the version`,
  );
});

Deno.test("parseCliArgs parses the help option", () => {
  assertEquals(parseCliArgs(["--help"]), { name: "help" });
  assertEquals(parseCliArgs(["-h"]), { name: "help" });
});

Deno.test("parseCliArgs parses the version option", () => {
  assertEquals(parseCliArgs(["--version"]), { name: "version" });
});

Deno.test("parseCliArgs parses the explicit summary command", () => {
  const command = parseCliArgs(
    ["summary", "--format", "json", "2026-05-01", "2026-05-31"],
  );

  if (command.name !== "summary" || !("startDay" in command)) {
    throw new Error("expected summary date range");
  }
  assertEquals(command.format, "json");
  assertEquals(command.separator, "\t");
  assertEquals(command.noProject, false);
  assertEquals(command.noDate, false);
  assertEquals(command.clipboard, false);
  assertEquals(
    [command.startDay.year, command.startDay.month, command.startDay.day],
    [2026, 5, 1],
  );
  assertEquals(
    [command.endDay.year, command.endDay.month, command.endDay.day],
    [2026, 5, 31],
  );
});

Deno.test("parseCliArgs accepts a summary range across years", () => {
  const command = parseCliArgs(
    [
      "summary",
      "--separator",
      ",",
      "--clipboard",
      "2025-12-31",
      "2026-01-01",
    ],
  );

  if (command.name !== "summary" || !("startDay" in command)) {
    throw new Error("expected summary date range");
  }
  assertEquals(command.separator, ",");
  assertEquals(command.noProject, false);
  assertEquals(command.noDate, false);
  assertEquals(command.clipboard, true);
  assertEquals(
    [command.startDay.year, command.startDay.month, command.startDay.day],
    [2025, 12, 31],
  );
  assertEquals(command.endDay.toString(), "2026-01-01");
});

Deno.test("parseCliArgs parses summary without the project column", () => {
  const command = parseCliArgs(
    ["summary", "--no-project", "2026-05-01", "2026-05-15"],
  );

  if (command.name !== "summary") throw new Error("expected summary command");
  assertEquals(command.noProject, true);
});

Deno.test("parseCliArgs parses summary without the date header row", () => {
  const command = parseCliArgs(
    ["summary", "--no-date", "2026-05-01", "2026-05-15"],
  );

  if (command.name !== "summary") throw new Error("expected summary command");
  assertEquals(command.noDate, true);
});

Deno.test("parseCliArgs parses long and short summary days options", () => {
  for (const option of ["--days", "-d"]) {
    const command = parseCliArgs([
      "summary",
      option,
      "7",
      "--format",
      "json",
    ]);

    if (command.name !== "summary" || !("days" in command)) {
      throw new Error("expected relative summary range");
    }
    assertEquals(command.days, 7);
    assertEquals(command.format, "json");
  }
});

Deno.test("parseCliArgs accepts zero summary days", () => {
  const command = parseCliArgs(["summary", "--days", "0"]);
  if (command.name !== "summary" || !("days" in command)) {
    throw new Error("expected relative summary range");
  }
  assertEquals(command.days, 0);
});

Deno.test("parseCliArgs rejects invalid or ambiguous summary days", () => {
  assertThrows(
    () => parseCliArgs(["summary", "--days", "-1"]),
    CliUsageError,
  );
  for (const days of ["1.5", "seven", "9007199254740992"]) {
    assertThrows(
      () => parseCliArgs(["summary", "--days", days]),
      CliUsageError,
      "days must be a non-negative integer",
    );
  }
  assertThrows(
    () =>
      parseCliArgs([
        "summary",
        "--days",
        "7",
        "2026-05-01",
        "2026-05-31",
      ]),
    CliUsageError,
    "summary accepts either start and end date or --days, not both",
  );
  assertThrows(
    () => parseCliArgs(["summary"]),
    CliUsageError,
    "summary requires start and end date or --days",
  );
  assertThrows(
    () => parseCliArgs(["summary", "--days"]),
    CliUsageError,
    "argument missing",
  );
});

Deno.test("parseCliArgs validates summary format and dates", () => {
  assertThrows(
    () =>
      parseCliArgs(["summary", "--format", "xml", "2026-05-01", "2026-05-02"]),
    CliUsageError,
    "format must be csv or json",
  );
  assertThrows(
    () => parseCliArgs(["summary", "2026-05-31", "2026-05-01"]),
    CliUsageError,
    "start date must not be after end date",
  );
  assertThrows(
    () => parseCliArgs(["summary", "2026-02-29", "2026-03-01"]),
    CliUsageError,
    "start and end date must be valid dates",
  );
  assertThrows(
    () => parseCliArgs(["summary", "1", "15"]),
    CliUsageError,
    "start and end date must use YYYY-MM-DD",
  );
  assertThrows(
    () => parseCliArgs(["summary", "--lastMonth", "2026-05-01", "2026-05-31"]),
    CliUsageError,
    "Unknown option",
  );
});

Deno.test("parseCliArgs accepts leap-day summary boundaries", () => {
  const command = parseCliArgs(["summary", "2024-02-01", "2024-02-29"]);
  if (command.name !== "summary" || !("startDay" in command)) {
    throw new Error("expected summary date range");
  }
  assertEquals(command.startDay.toString(), "2024-02-01");
  assertEquals(command.endDay.toString(), "2024-02-29");
});

Deno.test("resolveSummaryDateRange uses the configured timezone", () => {
  const command = {
    days: 7,
    separator: "\t",
    format: "csv" as const,
    noProject: false,
    noDate: false,
    clipboard: false,
  };
  const now = Temporal.Instant.from("2026-01-01T00:30:00Z");

  assertEquals(resolveSummaryDateRange(command, "UTC", now), {
    startDay: Temporal.PlainDate.from("2025-12-25"),
    endDay: Temporal.PlainDate.from("2026-01-01"),
  });
  assertEquals(resolveSummaryDateRange(command, "America/Los_Angeles", now), {
    startDay: Temporal.PlainDate.from("2025-12-24"),
    endDay: Temporal.PlainDate.from("2025-12-31"),
  });
});

Deno.test("resolveSummaryDateRange uses the system timezone and accepts zero days", () => {
  const range = resolveSummaryDateRange(
    {
      days: 0,
      separator: "\t",
      format: "csv",
      noProject: false,
      noDate: false,
      clipboard: false,
    },
    undefined,
    Temporal.Instant.from("2026-05-10T23:30:00Z"),
    "Asia/Tokyo",
  );

  assertEquals(range.startDay, Temporal.PlainDate.from("2026-05-11"));
  assertEquals(range.endDay, Temporal.PlainDate.from("2026-05-11"));
});

Deno.test("outputSummaryText writes to stdout only by default", async () => {
  const stdout: string[] = [];
  const clipboard: string[] = [];

  await outputSummaryText("summary output", false, {
    writeStdout(text) {
      stdout.push(text);
    },
    writeClipboard(text) {
      clipboard.push(text);
      return Promise.resolve();
    },
  });

  assertEquals(stdout, ["summary output"]);
  assertEquals(clipboard, []);
});

Deno.test("outputSummaryText writes to stdout and clipboard", async () => {
  const stdout: string[] = [];
  const clipboard: string[] = [];

  await outputSummaryText("summary output", true, {
    writeStdout(text) {
      stdout.push(text);
    },
    writeClipboard(text) {
      clipboard.push(text);
      return Promise.resolve();
    },
  });

  assertEquals(stdout, ["summary output"]);
  assertEquals(clipboard, ["summary output"]);
});

Deno.test("outputSummaryText reports clipboard failure without command details", async () => {
  const stdout: string[] = [];

  await assertRejects(
    () =>
      outputSummaryText("summary output", true, {
        writeStdout(text) {
          stdout.push(text);
        },
        writeClipboard() {
          throw new ClipboardUnavailableError();
        },
      }),
    ClipboardUnavailableError,
    "Could not copy output to the clipboard.",
  );

  assertEquals(stdout, ["summary output"]);
  assertEquals(
    new ClipboardUnavailableError().message.includes("not found"),
    false,
  );
});

Deno.test("parseCliArgs rejects the removed root summary syntax", () => {
  const error = assertThrows(
    () => parseCliArgs(["1", "31"]),
    CliUsageError,
  );
  assertEquals(error.message, "unknown command: 1");
});

Deno.test("parseCliArgs rejects unknown commands", () => {
  const error = assertThrows(
    () => parseCliArgs(["foo", "1", "31"]),
    CliUsageError,
  );
  assertEquals(error.message, "unknown command: foo");
});

Deno.test("parseCliArgs preserves init and projects routing", () => {
  assertEquals(parseCliArgs(["init"]), { name: "init" });
  assertEquals(parseCliArgs(["projects", "--format", "json"]), {
    name: "projects",
    format: "json",
  });
  assertEquals(parseCliArgs(["projects", "sync"]), {
    name: "projects-sync",
  });
  assertEquals(parseCliArgs(["config"]), {
    name: "config",
    format: "csv",
  });
  assertEquals(parseCliArgs(["config", "--format", "json"]), {
    name: "config",
    format: "json",
  });
});

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

Deno.test("createConfigTemplate returns TOML config template", () => {
  assertEquals(
    createConfigTemplate(),
    `workspace = "your_workspace_id"
token = "your_api_token"
timezone = "Asia/Tokyo"

[projects.123456]
display_name = "Client A"
hidden = false
display_order = 10

[projects.234567]
hidden = true
display_order = 20
`,
  );
});

Deno.test("createConfigToml returns TOML from entered values", () => {
  assertEquals(
    createConfigToml({
      workspace: "workspace-id",
      token: "test-token",
      timezone: "America/New_York",
    }),
    `workspace = "workspace-id"
token = "test-token"
timezone = "America/New_York"
`,
  );
});

Deno.test("createConfigToml uses the default timezone when omitted", () => {
  assertEquals(
    createConfigToml({ workspace: "workspace-id", token: "test-token" }),
    `workspace = "workspace-id"
token = "test-token"
timezone = "Asia/Tokyo"
`,
  );
});

Deno.test("parseProjectsConfig returns per-project settings", () => {
  assertEquals(
    parseProjectsConfig({
      "123456": { display_name: "Client A", display_order: 10 },
      "789012": { hidden: true },
      invalid: { display_name: "Ignored" },
      345678: "ignored",
Deno.test("parseConfigToml reports missing required keys", () => {
  const error = assertThrows(
    () => parseConfigToml('timezone = "UTC"'),
    ConfigValidationError,
  );
  assertEquals(error.missingKeys, ["workspace", "token"]);
  assertEquals(error.invalidProjects, []);
});

Deno.test("parseConfigToml reports invalid project settings", () => {
  const error = assertThrows(
    () =>
      parseConfigToml(`
workspace = "workspace-id"
token = "test-token"

[projects.invalid]
hidden = "yes"
`),
    ConfigValidationError,
  );
  assertEquals(error.missingKeys, []);
  assertEquals(error.invalidProjects, ["projects.invalid"]);
});

Deno.test("loadConfigDocument uses injected environment and file access", async () => {
  const requestedPaths: string[] = [];
  const document = await loadConfigDocument({
    getHome: () => "/home/tester",
    readTextFile: (path) => {
      requestedPaths.push(path);
      return Promise.resolve('workspace = "w"\ntoken = "t"\n');
    },
    isNotFound: () => false,
  });

  assertEquals(requestedPaths, ["/home/tester/.config/toggl-cli/config.toml"]);
  assertEquals(document.config.WORKSPACE, "w");
});

async function captureConfigBoundaryError(
  adapter: Parameters<typeof loadConfigDocument>[0],
) {
  const messages: string[] = [];
  const originalError = console.error;
  console.error = (...values: unknown[]) => messages.push(values.join(" "));
  try {
    const exitCode = await main(["config"], {
      runConfigCommand: async () => {
        await loadConfigDocument(adapter);
      },
    });
    return { exitCode, messages };
  } finally {
    console.error = originalError;
  }
}

Deno.test("main reports a missing config file and returns exit code 1", async () => {
  const result = await captureConfigBoundaryError({
    getHome: () => "/home/tester",
    readTextFile: () => Promise.reject(new Error("missing")),
    isNotFound: () => true,
  });
  assertEquals(result.exitCode, 1);
  assertEquals(result.messages, [
    "Error: ~/.config/toggl-cli/config.toml file not found",
    "Please create ~/.config/toggl-cli/config.toml with the following format:",
    'workspace = "your_workspace_id"',
    'token = "your_api_token"',
  ]);
});

Deno.test("main reports an unset HOME and returns exit code 1", async () => {
  const result = await captureConfigBoundaryError({
    getHome: () => undefined,
    readTextFile: () => Promise.reject(new Error("must not read")),
    isNotFound: () => false,
  });
  assertEquals(result, {
    exitCode: 1,
    messages: ["Error: HOME environment variable not set"],
  });
});

Deno.test("main reports config read errors and returns exit code 1", async () => {
  const result = await captureConfigBoundaryError({
    getHome: () => "/home/tester",
    readTextFile: () => Promise.reject(new Error("permission denied")),
    isNotFound: () => false,
  });
  assertEquals(result, {
    exitCode: 1,
    messages: ["Error: Unable to read ~/.config/toggl-cli/config.toml"],
  });
});

    }),
    {
      123456: { displayName: "Client A", hidden: false, displayOrder: 10 },
      789012: { displayName: undefined, hidden: true },
    },
  );
});

Deno.test("parseConfigToml reads token, workspace, and project settings", () => {
  assertEquals(
    parseConfigToml(`
workspace = "workspace-id"
token = "test-token"
timezone = "Asia/Tokyo"

[projects."123456"]
display_name = "Client A"
hidden = true
display_order = 20

[projects."789012"]
display_name = "Internal"
`),
    {
      WORKSPACE: "workspace-id",
      TOKEN: "test-token",
      TIMEZONE: "Asia/Tokyo",
      PROJECTS: {
        123456: { displayName: "Client A", hidden: true, displayOrder: 20 },
        789012: { displayName: "Internal", hidden: false },
      },
    },
  );
});

Deno.test("parseConfigToml reports missing required keys", () => {
  const error = assertThrows(
    () => parseConfigToml('timezone = "UTC"'),
    ConfigValidationError,
  );
  assertEquals(error.missingKeys, ["workspace", "token"]);
  assertEquals(error.invalidProjects, []);
});

Deno.test("parseConfigToml reports invalid project settings", () => {
  const error = assertThrows(
    () =>
      parseConfigToml(`
workspace = "workspace-id"
token = "test-token"

[projects.invalid]
hidden = "yes"
`),
    ConfigValidationError,
  );
  assertEquals(error.missingKeys, []);
  assertEquals(error.invalidProjects, ["projects.invalid"]);
});

Deno.test("loadConfigDocument uses injected environment and file access", async () => {
  const requestedPaths: string[] = [];
  const document = await loadConfigDocument({
    getHome: () => "/home/tester",
    readTextFile: (path) => {
      requestedPaths.push(path);
      return Promise.resolve('workspace = "w"\ntoken = "t"\n');
    },
    isNotFound: () => false,
  });

  assertEquals(requestedPaths, ["/home/tester/.config/toggl-cli/config.toml"]);
  assertEquals(document.config.WORKSPACE, "w");
});

async function captureConfigBoundaryError(
  adapter: Parameters<typeof loadConfigDocument>[0],
) {
  const messages: string[] = [];
  const originalError = console.error;
  console.error = (...values: unknown[]) => messages.push(values.join(" "));
  try {
    const exitCode = await main(["config"], {
      runConfigCommand: async () => {
        await loadConfigDocument(adapter);
      },
    });
    return { exitCode, messages };
  } finally {
    console.error = originalError;
  }
}

Deno.test("main reports a missing config file and returns exit code 1", async () => {
  const result = await captureConfigBoundaryError({
    getHome: () => "/home/tester",
    readTextFile: () => Promise.reject(new Error("missing")),
    isNotFound: () => true,
  });
  assertEquals(result.exitCode, 1);
  assertEquals(result.messages, [
    "Error: ~/.config/toggl-cli/config.toml file not found",
    "Please create ~/.config/toggl-cli/config.toml with the following format:",
    'workspace = "your_workspace_id"',
    'token = "your_api_token"',
  ]);
});

Deno.test("main reports an unset HOME and returns exit code 1", async () => {
  const result = await captureConfigBoundaryError({
    getHome: () => undefined,
    readTextFile: () => Promise.reject(new Error("must not read")),
    isNotFound: () => false,
  });
  assertEquals(result, {
    exitCode: 1,
    messages: ["Error: HOME environment variable not set"],
  });
});

Deno.test("main reports config read errors and returns exit code 1", async () => {
  const result = await captureConfigBoundaryError({
    getHome: () => "/home/tester",
    readTextFile: () => Promise.reject(new Error("permission denied")),
    isNotFound: () => false,
  });
  assertEquals(result, {
    exitCode: 1,
    messages: ["Error: Unable to read ~/.config/toggl-cli/config.toml"],
  });
});

Deno.test("formatProjectList returns one project name per line", () => {
  assertEquals(
    formatProjectList([
      {
        id: 1,
        name: "Project Alpha",
        displayName: "Project Alpha",
        active: true,
        hidden: false,
      },
      {
        id: 2,
        name: "Project Beta",
        displayName: "Custom Beta",
        active: true,
        hidden: false,
      },
    ]),
    "Project Alpha\nCustom Beta",
  );
});

Deno.test("formatProjectList returns an empty string for no projects", () => {
  assertEquals(formatProjectList([]), "");
});

Deno.test("formatProjectsJson returns explicit JSON output for projects", () => {
  assertEquals(
    formatProjectsJson([
      {
        id: 1,
        name: "Project Alpha",
        displayName: "Project Alpha",
        active: true,
        hidden: false,
      },
      {
        id: 2,
        name: "Project Beta",
        displayName: "Custom Beta",
        active: true,
        hidden: true,
      },
    ]),
    `[
  {
    "id": 1,
    "name": "Project Alpha",
    "displayName": "Project Alpha",
    "active": true,
    "hidden": false
  },
  {
    "id": 2,
    "name": "Project Beta",
    "displayName": "Custom Beta",
    "active": true,
    "hidden": true
  }
]`,
  );
});

Deno.test("withoutSensitiveConfig excludes TOKEN from visible settings", () => {
  assertEquals(
    withoutSensitiveConfig({
      WORKSPACE: "workspace-id",
      TOKEN: "test-token",
      TIMEZONE: "Asia/Tokyo",
      PROJECTS: {},
    }),
    {
      WORKSPACE: "workspace-id",
      TIMEZONE: "Asia/Tokyo",
    },
  );
});

Deno.test("formatConfigValues outputs visible settings as key-value lines", () => {
  assertEquals(
    formatConfigValues({
      WORKSPACE: "workspace-id",
      TOKEN: "test-token",
      TIMEZONE: "Asia/Tokyo",
      PROJECTS: {},
    }),
    "WORKSPACE=workspace-id\nTIMEZONE=Asia/Tokyo",
  );
});

Deno.test("formatConfigJson outputs visible settings as JSON", () => {
  assertEquals(
    formatConfigJson({
      WORKSPACE: "workspace-id",
      TOKEN: "test-token",
      TIMEZONE: "Asia/Tokyo",
      PROJECTS: {},
    }),
    `{
  "WORKSPACE": "workspace-id",
  "TIMEZONE": "Asia/Tokyo"
}`,
  );
});

Deno.test("appendMissingProjects preserves config and appends projects by id", () => {
  const configText = `workspace = "workspace-id"
token = "test-token"

# Keep this project setting.
[projects."20"]
display_name = "Custom name"
hidden = true
`;

  assertEquals(
    appendMissingProjects(configText, [20], [
      { id: 30, name: "Project Thirty", active: true },
      { id: 20, name: "Existing Project", active: true },
      { id: 10, name: "Project Ten", active: true },
    ]),
    {
      text: `${configText}
# Project Ten
[projects.10]
hidden = false

# Project Thirty
[projects.30]
hidden = false
`,
      addedCount: 2,
    },
  );
});

Deno.test("appendMissingProjects writes project names as comments", () => {
  const result = appendMissingProjects(
    `workspace = "workspace-id"
token = "test-token"
`,
    [],
    [{ id: 10, name: 'Client "A"\\Internal', active: true }],
  );

  assertEquals(
    result.text,
    `workspace = "workspace-id"
token = "test-token"

# Client "A"\\Internal
[projects.10]
hidden = false
`,
  );
  assertEquals(parseConfigToml(result.text).PROJECTS, {
    10: {
      displayName: undefined,
      hidden: false,
    },
  });
});

Deno.test("appendMissingProjects comments every project name line", () => {
  const result = appendMissingProjects(
    `workspace = "workspace-id"
token = "test-token"
`,
    [],
    [{ id: 10, name: "Client A\r\nInternal\nSupport", active: true }],
  );

  assertEquals(
    result.text,
    `workspace = "workspace-id"
token = "test-token"

# Client A
# Internal
# Support
[projects.10]
hidden = false
`,
  );
});

Deno.test("appendMissingProjects does not change fully configured text", () => {
  const configText = `workspace = "workspace-id"
token = "test-token"

[projects."10"]
hidden = false`;

  assertEquals(
    appendMissingProjects(
      configText,
      [10],
      [{ id: 10, name: "Project Ten", active: true }],
    ),
    { text: configText, addedCount: 0 },
  );
});

Deno.test("buildWorkTimeTable structures project rows across the requested date range", () => {
  const table = buildWorkTimeTable(
    [
      {
        id: 100,
        name: "Client work",
        displayName: "Client A",
        active: true,
        hidden: false,
      },
      {
        id: 200,
        name: "Internal",
        displayName: "Internal",
        active: true,
        hidden: false,
      },
    ],
    {
      "2026-05-01": { 100: 45.125 },
      "2026-05-02": { 200: 60 },
      "2026-05-03": { 100: 12 },
    },
    Temporal.PlainDate.from({ year: 2026, month: 5, day: 1 }),
    Temporal.PlainDate.from({ year: 2026, month: 5, day: 3 }),
  );

  assertEquals(table, {
    projectNames: ["Client A", "Internal"],
    headers: ["2026-05-01", "2026-05-02", "2026-05-03"],
    rows: [
      ["45.13", "", "12"],
      ["", "60", ""],
    ],
  });
});

Deno.test("buildWorkTimeTable enumerates dates across a year boundary", () => {
  const table = buildWorkTimeTable(
    [
      {
        id: 100,
        name: "Client work",
        displayName: "Client work",
        active: true,
        hidden: false,
      },
    ],
    {},
    Temporal.PlainDate.from("2025-12-31"),
    Temporal.PlainDate.from("2026-01-02"),
  );

  assertEquals(table.headers, [
    "2025-12-31",
    "2026-01-01",
    "2026-01-02",
  ]);
});

Deno.test("formatWorkTimeTable renders a single TSV table for spreadsheet paste", () => {
  const table = buildWorkTimeTable(
    [
      {
        id: 100,
        name: "Client work",
        displayName: "Client A",
        active: true,
        hidden: false,
      },
      {
        id: 200,
        name: "Internal",
        displayName: "Internal",
        active: true,
        hidden: false,
      },
Deno.test("getProjects throws a typed error for a non-success response", async () => {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = (() =>
    Promise.resolve(
      new Response(null, { status: 401, statusText: "Unauthorized" }),
    )) as typeof fetch;

  try {
    const error = await assertRejects(
      () => getProjects(config),
      TogglApiError,
      "Failed to fetch projects: HTTP 401 Unauthorized",
    );
    assertEquals(error.operation, "fetch projects");
    assertEquals(error.status, 401);
    assertEquals(
      error.url,
      `${apiEndpoint}/workspaces/${config.WORKSPACE}/projects`,
    );
  } finally {
    globalThis.fetch = originalFetch;
  }
});

    ],
    {
      "2026-05-01": { 100: 5 },
      "2026-05-10": { 200: 123.45 },
    },
    Temporal.PlainDate.from("2026-05-01"),
    Temporal.PlainDate.from("2026-05-10"),
  );

  assertEquals(
    formatWorkTimeTable(table, "\t"),
    [
      "Project\t2026-05-01\t2026-05-02\t2026-05-03\t2026-05-04\t2026-05-05\t2026-05-06\t2026-05-07\t2026-05-08\t2026-05-09\t2026-05-10",
      "Client A\t5\t\t\t\t\t\t\t\t\t",
      "Internal\t\t\t\t\t\t\t\t\t\t123.45",
    ].join("\n"),
  );
});

Deno.test("formatWorkTimeTable can omit the project column", () => {
  const table = buildWorkTimeTable(
    [
      {
        id: 100,
        name: "Client work",
        displayName: "Client A",
        active: true,
        hidden: false,
      },
      {
        id: 200,
        name: "Internal",
        displayName: "Internal",
        active: true,
        hidden: false,
      },
    ],
    {
      "2026-05-01": { 100: 5 },
      "2026-05-02": { 200: 30 },
    },
    Temporal.PlainDate.from("2026-05-01"),
    Temporal.PlainDate.from("2026-05-02"),
  );

  assertEquals(
    formatWorkTimeTable(table, "\t", true),
    [
      "2026-05-01\t2026-05-02",
      "5\t",
      "\t30",
    ].join("\n"),
  );
});

Deno.test("formatWorkTimeTable can omit dates and projects", () => {
  const table = {
Deno.test("getSummaryTimeEntries throws a typed error for a non-success response", async () => {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = (() =>
    Promise.resolve(
      new Response(null, { status: 429, statusText: "Too Many Requests" }),
    )) as typeof fetch;
  const fromDay = Temporal.PlainDate.from("2026-05-01");
  const toDay = Temporal.PlainDate.from("2026-05-31");

  try {
    const error = await assertRejects(
      () => getSummaryTimeEntries(config, fromDay, toDay),
      TogglApiError,
      "Failed to fetch summary time entries: HTTP 429 Too Many Requests",
    );
    assertEquals(error.operation, "fetch summary time entries");
    assertEquals(error.status, 429);
    assertEquals(
      error.url,
      `${reportsApiEndpoint}/workspace/${config.WORKSPACE}/summary/time_entries`,
    );
  } finally {
    globalThis.fetch = originalFetch;
  }
});

Deno.test("getTimeEntries fetches range in configured UTC timezone", async () => {
    const entries = await getTimeEntries(
    assertEquals(entries, []);
Deno.test("getTimeEntries throws a typed error for a non-success response", async () => {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = (() =>
    Promise.resolve(
      new Response(null, { status: 503, statusText: "Service Unavailable" }),
    )) as typeof fetch;
  const fromDay = Temporal.PlainDate.from("2026-05-01");
  const toDay = Temporal.PlainDate.from("2026-05-02");

  try {
    const error = await assertRejects(
      () => getTimeEntries({ ...config, TIMEZONE: "UTC" }, fromDay, toDay),
      TogglApiError,
      "Failed to fetch time entries: HTTP 503 Service Unavailable",
    );
    assertEquals(error.operation, "fetch time entries");
    assertEquals(error.status, 503);
    assertEquals(
      new URL(error.url).origin + new URL(error.url).pathname,
      `${apiEndpoint}/me/time_entries`,
    );
  } finally {
    globalThis.fetch = originalFetch;
  }
});

Deno.test("getTimeEntries maps response DTOs to domain models", async () => {

    const entries = await getTimeEntries(
    assertEquals(entries, [
      {
        id: 10,
        projectId: 100,
        start: "2026-05-01T12:00:00Z",
        stop: "2026-05-01T12:30:00Z",
        durationSeconds: 1800,
        description: "first block",
      },
      {
        id: 11,
        projectId: 100,
        start: "2026-05-01T13:00:00Z",
        stop: "2026-05-01T13:15:00Z",
        durationSeconds: 900,
        description: "second block",
      },
      {
        id: 12,
        projectId: 200,
        start: "2026-05-02T12:00:00Z",
        stop: "2026-05-02T13:00:00Z",
        durationSeconds: 3600,
        description: "legacy project id",
      },
    ]);
Deno.test("getTimeEntries returns domain entries without aggregating them", async () => {
    const entries = await getTimeEntries(
    assertEquals(entries, [{
      id: 20,
      projectId: 300,
      start: "2026-05-01T15:30:00Z",
      stop: "2026-05-01T16:00:00Z",
      durationSeconds: 1800,
      description: "crosses configured timezone date",
    }]);
Deno.test("formatTimeEntriesJson returns explicit JSON output for time entry data", () => {
  const json = formatTimeEntriesJson({
    "2026-05-07": {
      188325278: 60,
      188325289: 180,
      202971208: 30,
    },
  });

  assertEquals(
    json,
    `{
  "2026-05-07": {
    "188325278": 60,
    "188325289": 180,
    "202971208": 30
  }
}`,
  );
});

Deno.test("getProjects fetches active projects with Toggl auth", async () => {
  const originalFetch = globalThis.fetch;
  let requestedUrl = "";
  let requestedHeaders = new Headers();

  globalThis.fetch = ((input, init) => {
    requestedUrl = String(input);
    requestedHeaders = new Headers(
      (init as { headers?: HeadersInit } | undefined)?.headers,
    );

    return Promise.resolve(jsonResponse([
      { id: 1, name: "Client work", active: true },
      { id: 2, name: "Archived", active: false },
      { id: 3, project_name: "Legacy shape", project_active: true },
    ]));
  }) as typeof fetch;

  try {
    const projects = await getProjects(config);

    assertEquals(
      requestedUrl,
      `${apiEndpoint}/workspaces/${config.WORKSPACE}/projects`,
    );
    assertEquals(requestedHeaders.get("Content-Type"), "application/json");
    assertEquals(
      requestedHeaders.get("Authorization"),
      `Basic ${btoa(`${config.TOKEN}:api_token`)}`,
    );
    assertEquals(projects, [
      { id: 1, name: "Client work", active: true },
      { id: 3, name: "Legacy shape", active: true },
    ]);
  } finally {
    globalThis.fetch = originalFetch;
  }
});

Deno.test("getProjects throws a typed error for a non-success response", async () => {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = (() =>
    Promise.resolve(
      new Response(null, { status: 401, statusText: "Unauthorized" }),
    )) as typeof fetch;

  try {
    const error = await assertRejects(
      () => getProjects(config),
      TogglApiError,
      "Failed to fetch projects: HTTP 401 Unauthorized",
    );
    assertEquals(error.operation, "fetch projects");
    assertEquals(error.status, 401);
    assertEquals(
      error.url,
      `${apiEndpoint}/workspaces/${config.WORKSPACE}/projects`,
    );
  } finally {
    globalThis.fetch = originalFetch;
  }
});

Deno.test("getSummaryTimeEntries posts summary request with Toggl auth", async () => {
  const originalFetch = globalThis.fetch;
  let requestedUrl = "";
  let requestedMethod = "";
  let requestedHeaders = new Headers();
  let requestedBody: unknown;
  const summary = {
    groups: [
      {
        id: 100,
        title: { project: "Client work" },
        seconds: 5400,
      },
    ],
    seconds: 5400,
  };

  globalThis.fetch = ((input, init) => {
    const requestInit = init as
      | { body?: BodyInit | null; headers?: HeadersInit; method?: string }
      | undefined;

    requestedUrl = String(input);
    requestedMethod = requestInit?.method ?? "GET";
    requestedHeaders = new Headers(requestInit?.headers);
    requestedBody = JSON.parse(String(requestInit?.body));

    return Promise.resolve(jsonResponse(summary));
  }) as typeof fetch;

  const fromDay = Temporal.PlainDate.from("2026-05-01");
  const toDay = Temporal.PlainDate.from("2026-05-31");

  try {
    const response = await getSummaryTimeEntries(config, fromDay, toDay);

    assertEquals(
      requestedUrl,
      `${reportsApiEndpoint}/workspace/${config.WORKSPACE}/summary/time_entries`,
    );
    assertEquals(requestedMethod, "POST");
    assertEquals(requestedHeaders.get("Content-Type"), "application/json");
    assertEquals(
      requestedHeaders.get("Authorization"),
      `Basic ${btoa(`${config.TOKEN}:api_token`)}`,
    );
    assertEquals(requestedBody, {
      start_date: "2026-05-01",
      end_date: "2026-05-31",
      grouping: "projects",
    });
    assertEquals(response, summary);
  } finally {
    globalThis.fetch = originalFetch;
  }
});

Deno.test("getSummaryTimeEntries throws a typed error for a non-success response", async () => {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = (() =>
    Promise.resolve(
      new Response(null, { status: 429, statusText: "Too Many Requests" }),
    )) as typeof fetch;
  const fromDay = Temporal.PlainDate.from("2026-05-01");
  const toDay = Temporal.PlainDate.from("2026-05-31");

  try {
    const error = await assertRejects(
      () => getSummaryTimeEntries(config, fromDay, toDay),
      TogglApiError,
      "Failed to fetch summary time entries: HTTP 429 Too Many Requests",
    );
    assertEquals(error.operation, "fetch summary time entries");
    assertEquals(error.status, 429);
    assertEquals(
      error.url,
      `${reportsApiEndpoint}/workspace/${config.WORKSPACE}/summary/time_entries`,
    );
  } finally {
    globalThis.fetch = originalFetch;
  }
});

Deno.test("formatTimeEntryDate converts the same instant to configured timezone dates", () => {
  const start = "2026-05-01T15:30:00Z";

  assertEquals(formatTimeEntryDate(start, "Asia/Tokyo"), "2026-05-02");
  assertEquals(formatTimeEntryDate(start, "America/New_York"), "2026-05-01");
});

Deno.test("resolveTimeZone falls back to the system timezone", () => {
  assertEquals(
    resolveTimeZone("America/New_York", "Asia/Tokyo"),
    "America/New_York",
  );
  assertEquals(resolveTimeZone(undefined, "Asia/Tokyo"), "Asia/Tokyo");
});

Deno.test("getTimeEntriesForDays fetches range in configured UTC timezone", async () => {
  const originalFetch = globalThis.fetch;
  let requestedUrl = "";

  globalThis.fetch = ((input) => {
    requestedUrl = String(input);

    return Promise.resolve(jsonResponse([]));
  }) as typeof fetch;

  const fromDay = Temporal.PlainDate.from("2026-05-01");
  const toDay = Temporal.PlainDate.from("2026-05-02");

  try {
    const entries = await getTimeEntriesForDays(
      { ...config, TIMEZONE: "UTC" },
      fromDay,
      toDay,
    );
    const url = new URL(requestedUrl);

    assertEquals(url.origin + url.pathname, `${apiEndpoint}/me/time_entries`);
    assertEquals(
      url.searchParams.get("start_date"),
      "2026-05-01T00:00:00Z",
    );
    assertEquals(
      url.searchParams.get("end_date"),
      "2026-05-03T00:00:00Z",
    );
    assertEquals(entries, {});
  } finally {
    globalThis.fetch = originalFetch;
  }
});

Deno.test("getTimeEntriesForDays throws a typed error for a non-success response", async () => {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = (() =>
    Promise.resolve(
      new Response(null, { status: 503, statusText: "Service Unavailable" }),
    )) as typeof fetch;
  const fromDay = Temporal.PlainDate.from("2026-05-01");
  const toDay = Temporal.PlainDate.from("2026-05-02");

  try {
    const error = await assertRejects(
      () =>
        getTimeEntriesForDays({ ...config, TIMEZONE: "UTC" }, fromDay, toDay),
      TogglApiError,
      "Failed to fetch time entries: HTTP 503 Service Unavailable",
    );
    assertEquals(error.operation, "fetch time entries");
    assertEquals(error.status, 503);
    assertEquals(
      new URL(error.url).origin + new URL(error.url).pathname,
      `${apiEndpoint}/me/time_entries`,
    );
  } finally {
    globalThis.fetch = originalFetch;
  }
});

Deno.test("getTimeEntriesForDays fetches range and aggregates minutes by date and project", async () => {
  const configWithTimezone = {
    ...config,
    TIMEZONE: "Asia/Tokyo",
  };
  const originalFetch = globalThis.fetch;
  let requestedUrl = "";
  let requestedHeaders = new Headers();

  globalThis.fetch = ((input, init) => {
    requestedUrl = String(input);
    requestedHeaders = new Headers(
      (init as { headers?: HeadersInit } | undefined)?.headers,
    );

    return Promise.resolve(jsonResponse([
      {
        id: 10,
        project_id: 100,
        start: "2026-05-01T12:00:00Z",
        stop: "2026-05-01T12:30:00Z",
        duration: 1800,
        description: "first block",
      },
      {
        id: 11,
        project_id: 100,
        start: "2026-05-01T13:00:00Z",
        stop: "2026-05-01T13:15:00Z",
        duration: 900,
        description: "second block",
      },
      {
        id: 12,
        project_id: null,
        pid: 200,
        start: "2026-05-02T12:00:00Z",
        stop: "2026-05-02T13:00:00Z",
        duration: 3600,
        description: "legacy project id",
      },
    ]));
  }) as typeof fetch;

  const fromDay = Temporal.PlainDate.from("2026-05-01");
  const toDay = Temporal.PlainDate.from("2026-05-02");

  try {
    const entries = await getTimeEntriesForDays(
      configWithTimezone,
      fromDay,
      toDay,
    );
    const url = new URL(requestedUrl);

    assertEquals(url.origin + url.pathname, `${apiEndpoint}/me/time_entries`);
    assertEquals(url.searchParams.get("start_date"), "2026-04-30T15:00:00Z");
    assertEquals(url.searchParams.get("end_date"), "2026-05-02T15:00:00Z");
    assertEquals(url.searchParams.get("meta"), "true");
    assertEquals(requestedHeaders.get("Content-Type"), "application/json");
    assertEquals(
      requestedHeaders.get("Authorization"),
      `Basic ${btoa(`${config.TOKEN}:api_token`)}`,
    );
    assertEquals(entries, {
      "2026-05-01": { 100: 45 },
      "2026-05-02": { 200: 60 },
    });
  } finally {
    globalThis.fetch = originalFetch;
  }
});

Deno.test("getTimeEntriesForDays aggregates entries by date in configured timezone", async () => {
  const configWithTimezone = {
    ...config,
    TIMEZONE: "Asia/Tokyo",
  };
  const originalFetch = globalThis.fetch;

  globalThis.fetch = (() => {
    return Promise.resolve(jsonResponse([
      {
        id: 20,
        project_id: 300,
        start: "2026-05-01T15:30:00Z",
        stop: "2026-05-01T16:00:00Z",
        duration: 1800,
        description: "crosses configured timezone date",
      },
    ]));
  }) as typeof fetch;

  const fromDay = Temporal.PlainDate.from("2026-05-01");
  const toDay = Temporal.PlainDate.from("2026-05-02");

  try {
    const entries = await getTimeEntriesForDays(
      configWithTimezone,
      fromDay,
      toDay,
    );

    assertEquals(entries, {
      "2026-05-02": { 300: 30 },
    });
  } finally {
    globalThis.fetch = originalFetch;
  }
});
