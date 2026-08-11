import { assertEquals } from "@std/assert";
import { sortProjectsByDisplayOrder, visibleProjects } from "./project.ts";
import type { Project } from "./project.ts";

const projects: Project[] = [
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
    displayName: "Project Beta",
    active: true,
    hidden: true,
    displayOrder: 20,
  },
  {
    id: 3,
    name: "Project Gamma",
    displayName: "Project Gamma",
    active: true,
    hidden: false,
    displayOrder: 10,
  },
];

Deno.test("sortProjectsByDisplayOrder puts ordered projects first", () => {
  assertEquals(
    sortProjectsByDisplayOrder(projects).map((project) => project.id),
    [3, 2, 1],
  );
});

Deno.test("sortProjectsByDisplayOrder does not mutate its input", () => {
  sortProjectsByDisplayOrder(projects);
  assertEquals(projects.map((project) => project.id), [1, 2, 3]);
});

Deno.test("visibleProjects excludes hidden projects", () => {
  assertEquals(
    visibleProjects(projects).map((project) => project.id),
    [1, 3],
  );
});
