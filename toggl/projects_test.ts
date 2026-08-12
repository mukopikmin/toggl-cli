import { assertEquals } from "@std/assert";
import { mapProjectResponse } from "./projects.ts";

Deno.test("mapProjectResponse maps a Toggl DTO and display settings", () => {
  assertEquals(
    mapProjectResponse(
      {
        id: 2,
        project_name: "Project Beta",
        project_active: true,
      },
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

Deno.test("mapProjectResponse applies domain defaults", () => {
  assertEquals(
    mapProjectResponse({
      id: 1,
      project_name: "Project Alpha",
      project_active: true,
    }),
    {
      id: 1,
      name: "Project Alpha",
      displayName: "Project Alpha",
      active: true,
      hidden: false,
    },
  );
});

Deno.test("mapProjectResponse supports the API fallback fields", () => {
  assertEquals(
    mapProjectResponse({
      id: 3,
      project_name: "Report Project",
      project_active: false,
      name: "API Project",
      active: true,
    }),
    {
      id: 3,
      name: "API Project",
      displayName: "API Project",
      active: true,
      hidden: false,
    },
  );
});
