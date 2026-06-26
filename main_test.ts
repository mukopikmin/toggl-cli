import { assertEquals } from "@std/assert";
import { createConfigTemplate } from "./command/init.ts";
import {
  appendMissingProjects,
  formatProjectList,
  formatProjectsJson,
} from "./command/projects.ts";
import {
  buildWorkTimeTable,
  formatTimeEntriesJson,
} from "./command/summary.ts";
import { parseConfigToml, parseProjectsConfig } from "./config.ts";
import { createHelpText, resolveTargetMonth } from "./main.ts";
import { createProject, visibleProjects } from "./model/project.ts";
import { getProjects } from "./toggl/projects.ts";
import { getSummaryTimeEntries } from "./toggl/summary.ts";
import { getTimeEntriesForDays } from "./toggl/time_entries.ts";
import { apiEndpoint, reportsApiEndpoint } from "./toggl/api.ts";
import { formatTimeEntryDate } from "./toggl/date.ts";

const config = {
  WORKSPACE: "workspace-id",
  TOKEN: "test-token",
};

Deno.test("createHelpText describes commands and options", () => {
  assertEquals(
    createHelpText(),
    `Usage:
  toggl summary <start-day> <end-day> [options]
  toggl <start-day> <end-day> [options]
  toggl projects [options]
  toggl projects sync
  toggl init

Options:
  -l, --lastMonth        Aggregate the previous month
  -s, --separator <text> Set the output delimiter (default: tab)
  -f, --format <format>  Set the output format: csv or json (default: csv)
  -h, --help             Show this help
      --version          Show the version`,
  );
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

[projects.234567]
hidden = true
`,
  );
});

Deno.test("parseProjectsConfig returns per-project settings", () => {
  assertEquals(
    parseProjectsConfig({
      "123456": { display_name: "Client A" },
      "789012": { hidden: true },
      invalid: { display_name: "Ignored" },
      345678: "ignored",
    }),
    {
      123456: { displayName: "Client A", hidden: false },
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

[projects."789012"]
display_name = "Internal"
`),
    {
      WORKSPACE: "workspace-id",
      TOKEN: "test-token",
      TIMEZONE: "Asia/Tokyo",
      PROJECTS: {
        123456: { displayName: "Client A", hidden: true },
        789012: { displayName: "Internal", hidden: false },
      },
    },
  );
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

Deno.test("createProject stores original and display project names", () => {
  assertEquals(
    createProject(
      { id: 2, name: "Project Beta", active: true },
      { 2: { displayName: "Custom Beta", hidden: true } },
    ),
    {
      id: 2,
      name: "Project Beta",
      displayName: "Custom Beta",
      active: true,
      hidden: true,
    },
  );
});

Deno.test("visibleProjects excludes hidden projects", () => {
  assertEquals(
    visibleProjects([
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
    [
      {
        id: 1,
        name: "Project Alpha",
        displayName: "Project Alpha",
        active: true,
        hidden: false,
      },
    ],
  );
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

Deno.test("resolveTargetMonth returns December in previous year for January last month", () => {
  assertEquals(
    resolveTargetMonth({ year: 2026, month: 1 }, true),
    { year: 2025, month: 12 },
  );
});

Deno.test("resolveTargetMonth returns previous month in the same year", () => {
  assertEquals(
    resolveTargetMonth({ year: 2026, month: 5 }, true),
    { year: 2026, month: 4 },
  );
});

Deno.test("resolveTargetMonth returns current month when lastMonth is false", () => {
  assertEquals(
    resolveTargetMonth({ year: 2026, month: 5 }, false),
    { year: 2026, month: 5 },
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
      ["45.13", " ", "12"],
      [" ", "60", " "],
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

Deno.test("formatTimeEntryDate converts the same instant to configured timezone dates", () => {
  const start = "2026-05-01T15:30:00Z";

  assertEquals(formatTimeEntryDate(start, "Asia/Tokyo"), "2026-05-02");
  assertEquals(formatTimeEntryDate(start, "America/New_York"), "2026-05-01");
});

Deno.test("getTimeEntriesForDays fetches range without configured timezone", async () => {
  const originalFetch = globalThis.fetch;
  let requestedUrl = "";

  globalThis.fetch = ((input) => {
    requestedUrl = String(input);

    return Promise.resolve(jsonResponse([]));
  }) as typeof fetch;

  const fromDay = Temporal.PlainDate.from("2026-05-01");
  const toDay = Temporal.PlainDate.from("2026-05-02");

  try {
    const entries = await getTimeEntriesForDays(config, fromDay, toDay);
    const url = new URL(requestedUrl);

    assertEquals(url.origin + url.pathname, `${apiEndpoint}/me/time_entries`);
    assertEquals(
      url.searchParams.get("start_date"),
      "2026-05-01T00:00:00.000Z",
    );
    assertEquals(
      url.searchParams.get("end_date"),
      "2026-05-03T00:00:00.000Z",
    );
    assertEquals(entries, {});
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
