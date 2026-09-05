import { API_BASE, api, toQuery } from "@/config/api";
import type {
  AuditLogEntry,
  BackgroundJob,
  BackupInfo,
  CacheStats,
  CommonResponse,
  JobSchedule,
  JobTask,
  LogFileInfo,
  LogTail,
  PaginatedResponse,
  RestoreResult,
  UpsertJobScheduleInput,
} from "@/types";
import axios from "axios";

export const operationsService = {
  async listAuditLogs(params: {
    action?: string;
    cursor?: string;
    limit?: number;
  }): Promise<PaginatedResponse<AuditLogEntry>> {
    try {
      const res = await api.get(`/admin/audit${toQuery(params)}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as PaginatedResponse<AuditLogEntry>;
      }
      throw error;
    }
  },
  async listAuditActions(): Promise<CommonResponse<string[]>> {
    try {
      const res = await api.get("/admin/audit/actions");
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<string[]>;
      }
      throw error;
    }
  },
  async listJobs(params: {
    status?: string;
    type?: string;
    limit?: number;
    offset?: number;
  }): Promise<PaginatedResponse<BackgroundJob>> {
    try {
      const res = await api.get(`/jobs${toQuery(params)}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as PaginatedResponse<BackgroundJob>;
      }
      throw error;
    }
  },
  async listTasks(): Promise<CommonResponse<JobTask[]>> {
    try {
      const res = await api.get("/jobs/tasks");
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<JobTask[]>;
      }
      throw error;
    }
  },
  async triggerJob(type: string): Promise<CommonResponse<BackgroundJob>> {
    try {
      const res = await api.post("/jobs", { type });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<BackgroundJob>;
      }
      throw error;
    }
  },
  async listSchedules(): Promise<CommonResponse<JobSchedule[]>> {
    try {
      const res = await api.get("/jobs/schedules");
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<JobSchedule[]>;
      }
      throw error;
    }
  },
  async createSchedule(
    input: UpsertJobScheduleInput,
  ): Promise<CommonResponse<JobSchedule>> {
    try {
      const res = await api.post("/jobs/schedules", input);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<JobSchedule>;
      }
      throw error;
    }
  },
  async updateSchedule(
    id: string,
    input: UpsertJobScheduleInput,
  ): Promise<CommonResponse<JobSchedule>> {
    try {
      const res = await api.put(
        `/jobs/schedules/${encodeURIComponent(id)}`,
        input,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<JobSchedule>;
      }
      throw error;
    }
  },
  async deleteSchedule(id: string): Promise<CommonResponse<unknown>> {
    try {
      const res = await api.delete(`/jobs/schedules/${encodeURIComponent(id)}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<unknown>;
      }
      throw error;
    }
  },
  async runSchedule(id: string): Promise<CommonResponse<BackgroundJob>> {
    try {
      const res = await api.post(
        `/jobs/schedules/${encodeURIComponent(id)}/run`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<BackgroundJob>;
      }
      throw error;
    }
  },
  async listLogs(): Promise<CommonResponse<LogFileInfo[]>> {
    try {
      const res = await api.get("/system/logs");
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<LogFileInfo[]>;
      }
      throw error;
    }
  },
  async tailLogs(params: {
    file: string;
    lines?: number;
    level?: string;
    search?: string;
  }): Promise<CommonResponse<LogTail>> {
    try {
      const res = await api.get(`/system/logs/tail${toQuery(params)}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<LogTail>;
      }
      throw error;
    }
  },
  logDownloadUrl(name: string) {
    return `${API_BASE}/system/logs/${encodeURIComponent(name)}/download`;
  },
  async listBackups(): Promise<CommonResponse<BackupInfo[]>> {
    try {
      const res = await api.get("/system/backups");
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<BackupInfo[]>;
      }
      throw error;
    }
  },
  async createBackup(
    includeBooks: boolean,
  ): Promise<CommonResponse<BackupInfo>> {
    try {
      const res = await api.post("/system/backups", {
        include_books: includeBooks,
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<BackupInfo>;
      }
      throw error;
    }
  },
  async deleteBackup(name: string): Promise<CommonResponse<unknown>> {
    try {
      const res = await api.delete(
        `/system/backups/${encodeURIComponent(name)}`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<unknown>;
      }
      throw error;
    }
  },
  async restoreBackup(name: string): Promise<CommonResponse<RestoreResult>> {
    try {
      const res = await api.post(
        `/system/backups/${encodeURIComponent(name)}/restore`,
        { confirmation: "RESTORE" },
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<RestoreResult>;
      }
      throw error;
    }
  },
  backupDownloadUrl(name: string) {
    return `${API_BASE}/system/backups/${encodeURIComponent(name)}/download`;
  },
  async getCacheStats(): Promise<CommonResponse<CacheStats>> {
    try {
      const res = await api.get("/system/metrics/cache");
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<CacheStats>;
      }
      throw error;
    }
  },
};
