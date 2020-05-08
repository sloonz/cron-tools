export interface ISyncResult {
  hasSync: boolean;
  success: boolean;
  output: string | null;
  error: string | null;
  syncDate: string;
}

export interface ILogMetadata {
  logId: string;
  time: string;
  startTime: string;
  endTime: string;
  headers: Record<"Command" | "Host" | "User" | "Job-ID", string>;
  footers: Record<"Exit", string>;
}

export interface ILoggedJob {
  jobId: string;
  failedCount: number;
  successCount: number;
  lastLog: ILogMetadata | null;
  lastFail: ILogMetadata | null;
  lastSuccess: ILogMetadata | null;
}

export interface IRegisteredJob {
  description: {
    id: string;
    frequency: string;
    user: string;
    host: string;
    command: string[];
    tz: string;
    extra: Record<string, unknown>;
  };
  previousRunTime: string;
  nextRunTime: string;
}

export interface IJob {
  id: string;
  logs?: ILoggedJob;
  info?: IRegisteredJob;
}

export interface BreadcrumbComponent {
  label: string;
  to?: string;
}

export type SetBreadcrumb = (breadcrumb: BreadcrumbComponent[]) => void;
