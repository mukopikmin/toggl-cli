export interface Project {
  id: number;
  name: string;
  displayName: string;
  active: boolean;
  hidden: boolean;
  displayOrder?: number;
}

export function visibleProjects(projects: Project[]): Project[] {
  return projects.filter((project) => !project.hidden);
}

export function sortProjectsByDisplayOrder(projects: Project[]): Project[] {
  return projects.toSorted((a, b) => {
    if (a.displayOrder === undefined && b.displayOrder === undefined) {
      return 0;
    }
    if (a.displayOrder === undefined) {
      return 1;
    }
    if (b.displayOrder === undefined) {
      return -1;
    }
    return a.displayOrder - b.displayOrder;
  });
}
