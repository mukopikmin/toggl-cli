export interface TimeEntry {
  id: number;
  projectId: number;
  start: string;
  stop: string;
  durationSeconds: number;
  description: string;
}
