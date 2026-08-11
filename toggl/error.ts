export class TogglApiError extends Error {
  constructor(
    public readonly operation: string,
    public readonly status: number,
    public readonly url: string,
    statusText: string,
  ) {
    const statusDescription = statusText ? ` ${statusText}` : "";
    super(`Failed to ${operation}: HTTP ${status}${statusDescription}`);
    this.name = "TogglApiError";
  }
}
