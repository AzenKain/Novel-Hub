import { API_BASE, api, toQuery } from "@/config/api";
import type {
  BackgroundJob,
  BackupInfo,
  CommonResponse,
  JobSchedule,
  JobTask,
  LogFileInfo,
  LogTail,
  PaginatedResponse,
  RestoreResult,
  UpsertJobScheduleInput,
} from "@/types";

export const operationsService = {
  async listJobs(params: { status?: string; type?: string; limit?: number; offset?: number }): Promise<PaginatedResponse<BackgroundJob>> {
    const res = await api.get(`/jobs${toQuery(params)}`);
    return res.data;
  },
  async listTasks(): Promise<CommonResponse<JobTask[]>> {
    const res = await api.get("/jobs/tasks");
    return res.data;
  },
  async triggerJob(type: string): Promise<CommonResponse<BackgroundJob>> {
    const res = await api.post("/jobs", { type });
    return res.data;
  },
  async listSchedules(): Promise<CommonResponse<JobSchedule[]>> {
    const res = await api.get("/jobs/schedules");
    return res.data;
  },
  async createSchedule(input: UpsertJobScheduleInput): Promise<CommonResponse<JobSchedule>> {
    const res = await api.post("/jobs/schedules", input);
    return res.data;
  },
  async updateSchedule(id: string, input: UpsertJobScheduleInput): Promise<CommonResponse<JobSchedule>> {
    const res = await api.put(`/jobs/schedules/${encodeURIComponent(id)}`, input);
    return res.data;
  },
  async deleteSchedule(id: string): Promise<CommonResponse<unknown>> {
    const res = await api.delete(`/jobs/schedules/${encodeURIComponent(id)}`);
    return res.data;
  },
  async runSchedule(id: string): Promise<CommonResponse<BackgroundJob>> {
    const res = await api.post(`/jobs/schedules/${encodeURIComponent(id)}/run`);
    return res.data;
  },
  async listLogs(): Promise<CommonResponse<LogFileInfo[]>> {
    const res = await api.get("/system/logs");
    return res.data;
  },
  async tailLogs(params: { file: string; lines?: number; level?: string; search?: string }): Promise<CommonResponse<LogTail>> {
    const res = await api.get(`/system/logs/tail${toQuery(params)}`);
    return res.data;
  },
  logDownloadUrl(name: string) {
    return `${API_BASE}/system/logs/${encodeURIComponent(name)}/download`;
  },
  async listBackups(): Promise<CommonResponse<BackupInfo[]>> {
    const res = await api.get("/system/backups");
    return res.data;
  },
  async createBackup(includeBooks: boolean): Promise<CommonResponse<BackupInfo>> {
    const res = await api.post("/system/backups", { include_books: includeBooks });
    return res.data;
  },
  async deleteBackup(name: string): Promise<CommonResponse<unknown>> {
    const res = await api.delete(`/system/backups/${encodeURIComponent(name)}`);
    return res.data;
  },
  async restoreBackup(name: string): Promise<CommonResponse<RestoreResult>> {
    const res = await api.post(`/system/backups/${encodeURIComponent(name)}/restore`, { confirmation: "RESTORE" });
    return res.data;
  },
  backupDownloadUrl(name: string) {
    return `${API_BASE}/system/backups/${encodeURIComponent(name)}/download`;
  },
};
