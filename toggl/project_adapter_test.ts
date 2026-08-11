import { assertEquals } from "@std/assert";
import { mapTogglProject, mapTogglProjects } from "./project_adapter.ts";

Deno.test("mapTogglProject combines an API project with display settings", () => {
  assertEquals(
    mapTogglProject(
      { id: 2, name: "Project Beta", active: true },
      { displayName: "Custom Beta", hidden: true, displayOrder: 5 },
    ),
    {
      id: 2,
      name: "Project Beta",
      displayName: "Custom Beta",
      active: true,
      hidden: true,
      displayOrder: 5,
    },
  );
});

Deno.test("mapTogglProject uses domain defaults without display settings", () => {
  assertEquals(
    mapTogglProject({ id: 1, name: "Project Alpha", active: true }),
    {
      id: 1,
      name: "Project Alpha",
      displayName: "Project Alpha",
      active: true,
      hidden: false,
    },
  );
});

Deno.test("mapTogglProjects applies settings to the matching API project", () => {
  assertEquals(
    mapTogglProjects(
      [
        { id: 1, name: "Project Alpha", active: true },
        { id: 2, name: "Project Beta", active: false },
      ],
      { 2: { displayName: "Beta", hidden: true, displayOrder: 10 } },
    ),
    [
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
        displayName: "Beta",
        active: false,
        hidden: true,
        displayOrder: 10,
      },
    ],
  );
});
