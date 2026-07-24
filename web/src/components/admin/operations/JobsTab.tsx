import { useJobTasksQuery, useJobsQuery, useTriggerJobMutation } from "@/hooks";
import { useAuthStore } from "@/stores";
import { hasPermission } from "@/utils/permission";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";

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
    <div className="space-y-4">
      {canManage && (
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body p-4">
            <h3 className="font-bold">{t("admin.operations.run_task")}</h3>
            <div className="flex flex-wrap gap-2">
              {(tasks.data || []).map((task) => (
                <button
                  key={task.type}
                  className="btn btn-sm btn-outline"
                  disabled={trigger.isPending}
                  onClick={() => run(task.type)}
                  title={task.description}
                >
                  {t(`admin.operations.tasks.${task.type}`)}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}
      <div className="flex flex-wrap gap-2">
        <select
          className="select select-bordered select-sm"
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
          className="select select-bordered select-sm"
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
        <button className="btn btn-sm" onClick={() => void jobs.refetch()}>
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
                <td>{new Date(job.createdAt).toLocaleString()}</td>
                <td className="max-w-xs truncate text-error">
                  {job.errorMsg || "—"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!jobs.isLoading && !jobs.data?.items.length && (
          <div className="p-8 text-center opacity-60">
            {t("admin.operations.no_jobs")}
          </div>
        )}
      </div>
    </div>
  );
}
