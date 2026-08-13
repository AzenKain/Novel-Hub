import {
  useDeleteScheduleMutation,
  useJobTasksQuery,
  useRunScheduleMutation,
  useSaveScheduleMutation,
  useSchedulesQuery,
} from "@/hooks";
import { useAuthStore } from "@/stores";
import type { JobSchedule } from "@/types";
import { hasPermission } from "@/utils/permission";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";
import { ConfirmModal } from "@/components/common";

export function SchedulesTab() {
  const { t } = useTranslation();
  const schedules = useSchedulesQuery();
  const tasks = useJobTasksQuery();
  const save = useSaveScheduleMutation();
  const remove = useDeleteScheduleMutation();
  const run = useRunScheduleMutation();
  const [name, setName] = useState("");
  const [taskType, setTaskType] = useState("maintenance");
  const [interval, setInterval] = useState(1440);
  const [editing, setEditing] = useState<JobSchedule | null>(null);
  const [deleteScheduleId, setDeleteScheduleId] = useState<string | null>(null);

  const handleConfirmDelete = () => {
    if (!deleteScheduleId) return;
    remove.mutate(deleteScheduleId, {
      onSuccess: () => setDeleteScheduleId(null),
      onError: () => {
        toast.error(t("admin.operations.action_failed"));
        setDeleteScheduleId(null);
      },
    });
  };
  const user = useAuthStore((state) => state.user);
  const canManage = hasPermission(user, "job.manage");

  const saveForm = () =>
    save.mutate(
      {
        id: editing?.id,
        input: {
          name,
          task_type: taskType,
          payload_json: editing?.payloadJson,
          interval_minutes: interval,
          enabled: editing?.enabled ?? true,
        },
      },
      {
        onSuccess: (res) => {
          if (res.status) {
            setName("");
            setEditing(null);
            toast.success(t("admin.operations.schedule_saved"));
          } else toast.error(res.message);
        },
        onError: () => toast.error(t("admin.operations.action_failed")),
      },
    );

  const toggle = (item: JobSchedule) =>
    save.mutate(
      {
        id: item.id,
        input: {
          name: item.name,
          task_type: item.taskType,
          payload_json: item.payloadJson,
          interval_minutes: item.intervalMinutes,
          enabled: !item.enabled,
        },
      },
      { onError: () => toast.error(t("admin.operations.action_failed")) },
    );

  return (
    <div className="space-y-4">
      {canManage && (
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body p-4">
            <h3 className="font-bold">{t("admin.operations.add_schedule")}</h3>
            <div className="grid gap-3 md:grid-cols-4">
              <input
                className="input input-bordered input-sm"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t("admin.operations.schedule_name")}
              />
              <select
                className="select select-bordered select-sm"
                value={taskType}
                onChange={(e) => setTaskType(e.target.value)}
              >
                {(tasks.data || []).map((task) => (
                  <option key={task.type} value={task.type}>
                    {t(`admin.operations.tasks.${task.type}`)}
                  </option>
                ))}
              </select>
              <label className="input input-bordered input-sm flex items-center gap-2">
                <input
                  className="grow"
                  type="number"
                  min={5}
                  max={525600}
                  value={interval}
                  onChange={(e) => setInterval(Number(e.target.value))}
                />
                <span className="text-xs opacity-60">
                  {t("admin.operations.minutes")}
                </span>
              </label>
              <button
                className="btn btn-primary btn-sm"
                disabled={!name.trim() || save.isPending}
                onClick={saveForm}
              >
                {editing ? t("common.save") : t("admin.operations.add")}
              </button>
            </div>
          </div>
        </div>
      )}
      <div className="overflow-x-auto bg-base-100 rounded-box shadow-sm">
        <table className="table table-sm">
          <thead>
            <tr>
              <th>{t("admin.operations.schedule_name")}</th>
              <th>{t("admin.operations.task")}</th>
              <th>{t("admin.operations.interval")}</th>
              <th>{t("admin.operations.next_run")}</th>
              <th>{t("admin.operations.enabled")}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {(schedules.data || []).map((item) => (
              <tr key={item.id}>
                <td className="font-medium">{item.name}</td>
                <td>{t(`admin.operations.tasks.${item.taskType}`)}</td>
                <td>
                  {item.intervalMinutes} {t("admin.operations.minutes")}
                </td>
                <td>{new Date(item.nextRunAt).toLocaleString()}</td>
                <td>
                  <input
                    type="checkbox"
                    className="toggle toggle-sm"
                    checked={item.enabled}
                    disabled={!canManage}
                    onChange={() => toggle(item)}
                  />
                </td>
                <td className="flex gap-1 justify-end">
                  {canManage && (
                    <>
                      <button
                        className="btn btn-xs"
                        onClick={() => {
                          setEditing(item);
                          setName(item.name);
                          setTaskType(item.taskType);
                          setInterval(item.intervalMinutes);
                        }}
                      >
                        {t("admin.edit")}
                      </button>
                      <button
                        className="btn btn-xs"
                        onClick={() =>
                          run.mutate(item.id, {
                            onError: () =>
                              toast.error(t("admin.operations.action_failed")),
                          })
                        }
                      >
                        {t("admin.operations.run_now")}
                      </button>
                      <button
                        className="btn btn-xs btn-error btn-outline"
                        onClick={() => {
                          setDeleteScheduleId(item.id);
                        }}
                      >
                        {t("common.delete")}
                      </button>
                    </>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!schedules.isLoading && !schedules.data?.length && (
          <div className="p-8 text-center opacity-60">
            {t("admin.operations.no_schedules")}
          </div>
        )}
      </div>

      <ConfirmModal
        open={deleteScheduleId !== null}
        title={t("admin.operations.confirm_delete_title", "Delete Schedule")}
        message={t("admin.operations.confirm_delete", "Are you sure you want to delete this schedule?")}
        onClose={() => setDeleteScheduleId(null)}
        onConfirm={handleConfirmDelete}
        variant="danger"
        loading={remove.isPending}
      />
    </div>
  );
}
