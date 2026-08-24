export interface TimeEntry {
  id: number;
  projectId: number | null;
  start: string;
  stop: string | null;
  durationSeconds: number;
  description: string;
}
