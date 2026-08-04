import {
  useBackupsQuery,
  useCreateBackupMutation,
  useDeleteBackupMutation,
  useRestoreBackupMutation,
} from "@/hooks";
import { operationsService } from "@/services";
import type { TFunction } from "i18next";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";

const formatBytes = (bytes: number, t: TFunction) => {
  if (bytes < 1024) return `${bytes} ${t("admin.operations.units.bytes")}`;
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(1)} ${t("admin.operations.units.kib")}`;
  if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(1)} ${t("admin.operations.units.mib")}`;
  return `${(bytes / 1024 ** 3).toFixed(1)} ${t("admin.operations.units.gib")}`;
};

export function BackupsTab() {
  const { t } = useTranslation();
  const backups = useBackupsQuery();
  const create = useCreateBackupMutation();
  const remove = useDeleteBackupMutation();
  const restore = useRestoreBackupMutation();
  const [includeBooks, setIncludeBooks] = useState(false);

  const createBackup = () =>
    create.mutate(includeBooks, {
      onSuccess: (res) =>
        res.status
          ? toast.success(t("admin.operations.backup_created"))
          : toast.error(res.message),
      onError: () => toast.error(t("admin.operations.action_failed")),
    });
  const restoreBackup = (name: string) => {
    if (!window.confirm(t("admin.operations.confirm_restore"))) return;
    restore.mutate(name, {
      onSuccess: (res) =>
        res.status
          ? toast.warning(
              res.data?.autoRestart
                ? t("admin.operations.restarting")
                : t("admin.operations.restart_required"),
            )
          : toast.error(res.message),
      onError: () => toast.error(t("admin.operations.action_failed")),
    });
  };

  return (
    <div className="space-y-4">
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body p-4 flex-row items-center justify-between flex-wrap gap-3">
          <label className="label cursor-pointer gap-3">
            <input
              type="checkbox"
              className="checkbox checkbox-sm"
              checked={includeBooks}
              onChange={(e) => setIncludeBooks(e.target.checked)}
            />
            <span>{t("admin.operations.include_books")}</span>
          </label>
          <button
            className="btn btn-primary btn-sm"
            disabled={create.isPending}
            onClick={createBackup}
          >
            {create.isPending ? (
              <span className="loading loading-spinner loading-xs" />
            ) : null}
            {t("admin.operations.create_backup")}
          </button>
        </div>
      </div>
      <div className="overflow-x-auto bg-base-100 rounded-box shadow-sm">
        <table className="table table-sm">
          <thead>
            <tr>
              <th>{t("admin.operations.backup")}</th>
              <th>{t("admin.operations.scope")}</th>
              <th>{t("admin.operations.size")}</th>
              <th>{t("admin.operations.created")}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {(backups.data || []).map((item) => (
              <tr key={item.name}>
                <td className="font-mono text-xs">{item.name}</td>
                <td>
                  {item.includeBooks
                    ? t("admin.operations.db_and_books")
                    : t("admin.operations.database_only")}
                </td>
                <td>{formatBytes(item.size_bytes, t)}</td>
                <td>{new Date(item.created_at).toLocaleString()}</td>
                <td className="flex justify-end gap-1">
                  <a
                    className="btn btn-xs"
                    href={operationsService.backupDownloadUrl(item.name)}
                  >
                    {t("common.download")}
                  </a>
                  <button
                    className="btn btn-xs btn-warning btn-outline"
                    disabled={restore.isPending}
                    onClick={() => restoreBackup(item.name)}
                  >
                    {t("admin.operations.restore")}
                  </button>
                  <button
                    className="btn btn-xs btn-error btn-outline"
                    disabled={remove.isPending}
                    onClick={() => {
                      if (window.confirm(t("admin.operations.confirm_delete")))
                        remove.mutate(item.name, {
                          onError: () =>
                            toast.error(t("admin.operations.action_failed")),
                        });
                    }}
                  >
                    {t("common.delete")}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!backups.isLoading && !backups.data?.length && (
          <div className="p-8 text-center opacity-60">
            {t("admin.operations.no_backups")}
          </div>
        )}
      </div>
    </div>
  );
}
