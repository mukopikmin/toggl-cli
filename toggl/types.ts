export interface TogglConfig {
  WORKSPACE: string;
  TOKEN: string;
}

export interface Project {
  id: number;
  name: string;
  active: boolean;
}

export interface TimeEntry {
  id: number;
  project_id: number;
  start: string;
  stop: string;
  duration: number;
  description: string;
}
