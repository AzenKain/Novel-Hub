import { operationsService } from "@/services";
import type { UpsertJobScheduleInput } from "@/types";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export function useJobsQuery(status = "", type = "") {
  return useQuery({
    queryKey: ["operations", "jobs", status, type],
    queryFn: async () => {
      const res = await operationsService.listJobs({ status: status || undefined, type: type || undefined, limit: 100 });
      if (!res.status) throw new Error(res.message || "jobs_failed");
      return { items: res.data || [], total: res.pagination?.total_records || 0 };
    },
    refetchInterval: (query) => query.state.data?.items.some((job) => job.status === "pending" || job.status === "running") ? 3000 : false,
  });
}

export function useJobTasksQuery() {
  return useQuery({
    queryKey: ["operations", "tasks"],
    queryFn: async () => {
      const res = await operationsService.listTasks();
      if (!res.status) throw new Error(res.message || "tasks_failed");
      return res.data || [];
    },
  });
}

export function useTriggerJobMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (type: string) => operationsService.triggerJob(type),
    onSuccess: () => void client.invalidateQueries({ queryKey: ["operations", "jobs"] }),
  });
}

export function useSchedulesQuery() {
  return useQuery({
    queryKey: ["operations", "schedules"],
    queryFn: async () => {
      const res = await operationsService.listSchedules();
      if (!res.status) throw new Error(res.message || "schedules_failed");
      return res.data || [];
    },
  });
}

export function useSaveScheduleMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id?: string; input: UpsertJobScheduleInput }) => id ? operationsService.updateSchedule(id, input) : operationsService.createSchedule(input),
    onSuccess: () => void client.invalidateQueries({ queryKey: ["operations", "schedules"] }),
  });
}

export function useDeleteScheduleMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => operationsService.deleteSchedule(id),
    onSuccess: () => void client.invalidateQueries({ queryKey: ["operations", "schedules"] }),
  });
}

export function useRunScheduleMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => operationsService.runSchedule(id),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ["operations", "jobs"] });
      void client.invalidateQueries({ queryKey: ["operations", "schedules"] });
    },
  });
}

export function useLogFilesQuery() {
  return useQuery({
    queryKey: ["operations", "logs"],
    queryFn: async () => {
      const res = await operationsService.listLogs();
      if (!res.status) throw new Error(res.message || "logs_failed");
      return res.data || [];
    },
  });
}

export function useLogTailQuery(file: string, level: string, search: string) {
  return useQuery({
    queryKey: ["operations", "log-tail", file, level, search],
    enabled: Boolean(file),
    queryFn: async () => {
      const res = await operationsService.tailLogs({ file, lines: 500, level: level || undefined, search: search || undefined });
      if (!res.status) throw new Error(res.message || "log_tail_failed");
      return res.data;
    },
  });
}

export function useBackupsQuery() {
  return useQuery({
    queryKey: ["operations", "backups"],
    queryFn: async () => {
      const res = await operationsService.listBackups();
      if (!res.status) throw new Error(res.message || "backups_failed");
      return res.data || [];
    },
  });
}

export function useCreateBackupMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (includeBooks: boolean) => operationsService.createBackup(includeBooks),
    onSuccess: () => void client.invalidateQueries({ queryKey: ["operations", "backups"] }),
  });
}

export function useDeleteBackupMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => operationsService.deleteBackup(name),
    onSuccess: () => void client.invalidateQueries({ queryKey: ["operations", "backups"] }),
  });
}

export function useRestoreBackupMutation() {
  return useMutation({ mutationFn: (name: string) => operationsService.restoreBackup(name) });
}
