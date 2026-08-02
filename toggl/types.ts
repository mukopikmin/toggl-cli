export interface TogglConfig {
  WORKSPACE: string;
  TOKEN: string;
  TIMEZONE?: string;
}

export interface TogglProject {
  id: number;
  name: string;
  active: boolean;
}

export interface TimeEntry {
  id: number;
  project_id: number | null;
  start: string;
  stop: string | null;
  duration: number;
  description: string;
}

export type SummaryTimeEntriesResponse = Record<string, unknown>;
