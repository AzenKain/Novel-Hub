import { useJobTasksQuery, useJobsQuery, useTriggerJobMutation } from "@/hooks";
import { useAuthStore } from "@/stores";
import { hasPermission } from "@/utils/permission";
import {
  Activity,
  Archive,
  CheckCheck,
  ChevronDown,
  Database,
  FileText,
  FolderMinus,
  FolderSync,
  ListTodo,
  RefreshCw,
  Trash2,
  Wrench,
  Zap,
} from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";

function getTaskIcon(type: string) {
  switch (type) {
    case "full_maintenance":
      return <Wrench className="w-4 h-4 text-primary shrink-0" />;
    case "scan_library_inbox":
      return <FolderSync className="w-4 h-4 text-info shrink-0" />;
    case "delete_empty_book_dirs":
      return <FolderMinus className="w-4 h-4 text-warning shrink-0" />;
    case "delete_old_uploads":
      return <Trash2 className="w-4 h-4 text-error shrink-0" />;
    case "cleanup_finished_jobs":
      return <CheckCheck className="w-4 h-4 text-success shrink-0" />;
    case "cleanup_audit_logs":
      return <FileText className="w-4 h-4 text-secondary shrink-0" />;
    case "database_health_check":
      return <Activity className="w-4 h-4 text-accent shrink-0" />;
    case "backup_database":
      return <Database className="w-4 h-4 text-primary shrink-0" />;
    case "backup_database_books":
      return <Archive className="w-4 h-4 text-primary shrink-0" />;
    default:
      return <Zap className="w-4 h-4 text-base-content/70 shrink-0" />;
  }
}

export function JobsTab() {
  const { t } = useTranslation();
  const [status, setStatus] = useState("");
  const [type, setType] = useState("");
  const jobs = useJobsQuery(status, type);
  const tasks = useJobTasksQuery();
  const trigger = useTriggerJobMutation();
  const user = useAuthStore((state) => state.user);
  const canManage = hasPermission(user, "job.manage");

  const run = (taskType: string) =>
    trigger.mutate(taskType, {
      onSuccess: (res) =>
        res.status
          ? toast.success(t("admin.operations.job_queued"))
          : toast.error(res.message),
      onError: () => toast.error(t("admin.operations.action_failed")),
    });

  return (
    <div className="space-y-3">
      {/* Maintenance Action Buttons Box - Compact & Visible with Icons */}
      {canManage && (
        <div className="border border-base-200 bg-base-100 rounded-2xl p-3 sm:p-3.5 shadow-2xs space-y-2">
          <div className="flex items-center justify-between border-b border-base-200/60 pb-2">
            <div className="flex items-center gap-2">
              <Wrench className="w-4 h-4 text-primary" />
              <h3 className="font-bold text-xs uppercase tracking-wider text-base-content/80">
                {t("admin.operations.run_task", "Run Maintenance Task")}
              </h3>
            </div>
            <span className="text-[11px] text-base-content/40 font-medium hidden sm:inline">
              Click to execute task
            </span>
          </div>

          <div className="flex flex-wrap items-center gap-1.5 sm:gap-2">
            {(tasks.data || []).map((task) => (
              <button
                key={task.type}
                className="btn btn-sm bg-base-100 border border-base-200 hover:border-primary/40 hover:bg-primary/5 hover:text-primary rounded-xl gap-1.5 font-medium text-xs text-base-content shadow-2xs transition-all"
                disabled={trigger.isPending}
                onClick={() => run(task.type)}
                title={task.description}
              >
                {getTaskIcon(task.type)}
                <span>{t(`admin.operations.tasks.${task.type}`)}</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* 1-Row Filter Toolbar: Mọi trạng thái, Mọi tác vụ & Làm mới */}
      <div className="flex items-center justify-between gap-2 w-full">
        <div className="flex items-center gap-2 flex-nowrap min-w-0">
          <select
            className="select select-bordered select-sm bg-base-100 rounded-xl text-xs h-8 min-w-[120px] max-w-[160px]"
            value={status}
            onChange={(e) => setStatus(e.target.value)}
            aria-label={t("admin.operations.status")}
          >
            <option value="">{t("admin.operations.all_statuses")}</option>
            {["pending", "running", "completed", "failed"].map((item) => (
              <option key={item} value={item}>
                {t(`admin.operations.statuses.${item}`)}
              </option>
            ))}
          </select>

          <select
            className="select select-bordered select-sm bg-base-100 rounded-xl text-xs h-8 min-w-[130px] max-w-[180px]"
            value={type}
            onChange={(e) => setType(e.target.value)}
            aria-label={t("admin.operations.task")}
          >
            <option value="">{t("admin.operations.all_tasks")}</option>
            {(tasks.data || []).map((task) => (
              <option key={task.type} value={task.type}>
                {t(`admin.operations.tasks.${task.type}`)}
              </option>
            ))}
          </select>
        </div>

        <button className="btn btn-sm btn-ghost gap-1.5 text-xs h-8 shrink-0" onClick={() => void jobs.refetch()}>
          <RefreshCw className={`w-3.5 h-3.5 ${jobs.isLoading ? "animate-spin" : ""}`} />
          {t("admin.operations.refresh")}
        </button>
      </div>
      <div className="overflow-x-auto bg-base-100 rounded-box shadow-sm">
        <table className="table table-sm">
          <thead>
            <tr>
              <th>{t("admin.operations.task")}</th>
              <th>{t("admin.operations.status")}</th>
              <th>{t("admin.operations.created")}</th>
              <th>{t("admin.operations.error")}</th>
            </tr>
          </thead>
          <tbody>
            {(jobs.data?.items || []).map((job) => (
              <tr key={job.id}>
                <td>
                  <div className="font-medium">
                    {t(`admin.operations.tasks.${job.type}`)}
                  </div>
                  <div className="text-xs opacity-50 font-mono">{job.id}</div>
                </td>
                <td>
                  <span
                    className={`badge badge-sm ${job.status === "completed" ? "badge-success" : job.status === "failed" ? "badge-error" : job.status === "running" ? "badge-info" : "badge-warning"}`}
                  >
                    {t(`admin.operations.statuses.${job.status || "pending"}`)}
                  </span>
                </td>
                <td>{new Date(job.created_at).toLocaleString()}</td>
                <td className="max-w-xs truncate text-error">
                  {job.errorMsg || "—"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!jobs.isLoading && !jobs.data?.items.length && (
          <div className="rounded-2xl border border-dashed border-base-300 bg-base-100 p-12 text-center flex flex-col items-center justify-center gap-3 my-4 mx-4 shadow-xs">
            <div className="grid h-12 w-12 place-items-center rounded-2xl bg-primary/10 text-primary">
              <ListTodo className="h-6 w-6" />
            </div>
            <div>
              <p className="font-bold text-base text-base-content/80">{t("admin.operations.no_jobs", "No Background Tasks Found")}</p>
              <p className="text-xs text-base-content/50 mt-1">{t("admin.operations.no_jobs_hint", "Tasks triggered manually or scheduled automatically will appear here.")}</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
