import { assertEquals } from "@std/assert";
import { datetime } from "ptera";
import {
  buildWorkTimeTable,
  formatTimeEntriesJson,
} from "./command/summary.ts";
import {
  formatProjectList,
  formatProjectsJson,
  resolveTargetMonth,
} from "./main.ts";
import { getProjects } from "./toggl/projects.ts";
import { getSummaryTimeEntries } from "./toggl/summary.ts";
import { getTimeEntriesForDays } from "./toggl/time_entries.ts";
import { apiEndpoint, reportsApiEndpoint } from "./toggl/api.ts";

const config = {
  WORKSPACE: "workspace-id",
  TOKEN: "test-token",
};

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

Deno.test("formatProjectList returns one project name per line", () => {
  assertEquals(
    formatProjectList([
      { id: 1, name: "Project Alpha", active: true },
      { id: 2, name: "Project Beta", active: true },
    ]),
    "Project Alpha\nProject Beta",
  );
});

Deno.test("formatProjectList returns an empty string for no projects", () => {
  assertEquals(formatProjectList([]), "");
});

Deno.test("formatProjectsJson returns explicit JSON output for projects", () => {
  assertEquals(
    formatProjectsJson([
      { id: 1, name: "Project Alpha", active: true },
      { id: 2, name: "Project Beta", active: true },
    ]),
    `[
  {
    "id": 1,
    "name": "Project Alpha",
    "active": true
  },
  {
    "id": 2,
    "name": "Project Beta",
    "active": true
  }
]`,
  );
});

Deno.test("resolveTargetMonth returns December in previous year for January last month", () => {
  assertEquals(
    resolveTargetMonth(datetime({ year: 2026, month: 1, day: 15 }), true),
    { year: 2025, month: 12 },
  );
});

Deno.test("resolveTargetMonth returns previous month in the same year", () => {
  assertEquals(
    resolveTargetMonth(datetime({ year: 2026, month: 5, day: 15 }), true),
    { year: 2026, month: 4 },
  );
});

Deno.test("resolveTargetMonth returns current month when lastMonth is false", () => {
  assertEquals(
    resolveTargetMonth(datetime({ year: 2026, month: 5, day: 15 }), false),
    { year: 2026, month: 5 },
  );
});

Deno.test("buildWorkTimeTable structures project rows across the requested date range", () => {
  const table = buildWorkTimeTable(
    [
      { id: 100, name: "Client work", active: true },
      { id: 200, name: "Internal", active: true },
    ],
    {
      "2026-05-01": { 100: 45.125 },
      "2026-05-02": { 200: 60 },
      "2026-05-03": { 100: 12 },
    },
    datetime({ year: 2026, month: 5, day: 1 }),
    datetime({ year: 2026, month: 5, day: 3 }),
  );

  assertEquals(table, {
    projectNames: ["Client work", "Internal"],
    headers: ["2026-05-01", "2026-05-02", "2026-05-03"],
    rows: [
      ["45.13", " ", "12"],
      [" ", "60", " "],
    ],
  });
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

  const fromDay = datetime({ year: 2026, month: 5, day: 1 });
  const toDay = datetime({ year: 2026, month: 5, day: 31 });

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

Deno.test("getTimeEntriesForDays fetches range without configured timezone", async () => {
  const originalFetch = globalThis.fetch;
  let requestedUrl = "";

  globalThis.fetch = ((input) => {
    requestedUrl = String(input);

    return Promise.resolve(jsonResponse([]));
  }) as typeof fetch;

  const fromDay = datetime({ year: 2026, month: 5, day: 1 });
  const toDay = datetime({ year: 2026, month: 5, day: 2 });

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

  const fromDay = datetime({ year: 2026, month: 5, day: 1 });
  const toDay = datetime({ year: 2026, month: 5, day: 2 });

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
