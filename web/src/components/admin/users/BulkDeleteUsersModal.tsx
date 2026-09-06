import React, { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { AlertTriangle, Loader2, Trash2, ShieldAlert } from "lucide-react";
import { toast } from "react-toastify";
import { useBulkDeleteUsersMutation } from "@/hooks/useAdminQueries";
import type { User } from "@/types";
import { isAdminUser } from "@/utils/permission";

interface BulkDeleteUsersModalProps {
  isOpen: boolean;
  selectedUsers: User[];
  currentUserId?: string;
  isCallerOwner?: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export const BulkDeleteUsersModal: React.FC<BulkDeleteUsersModalProps> = ({
  isOpen,
  selectedUsers,
  currentUserId,
  isCallerOwner = false,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation();
  const deleteMutation = useBulkDeleteUsersMutation();

  const { eligibleUsers, skippedList } = useMemo(() => {
    const eligible: User[] = [];
    const skipped: { user: User; reasonKey: string }[] = [];

    for (const user of selectedUsers) {
      if (user.is_deleted) {
        skipped.push({ user, reasonKey: "admin.bulk_skip_already_deleted" });
        continue;
      }
      if (user.is_owner) {
        skipped.push({ user, reasonKey: "admin.bulk_skip_owner" });
        continue;
      }
      if (user.id === currentUserId) {
        skipped.push({ user, reasonKey: "admin.bulk_skip_self" });
        continue;
      }
      if (isAdminUser(user) && !isCallerOwner) {
        skipped.push({ user, reasonKey: "admin.bulk_skip_admin" });
        continue;
      }
      eligible.push(user);
    }

    return { eligibleUsers: eligible, skippedList: skipped };
  }, [selectedUsers, currentUserId, isCallerOwner]);

  if (!isOpen) return null;

  const handleDelete = async () => {
    if (eligibleUsers.length === 0) return;
    try {
      const res = await deleteMutation.mutateAsync({
        user_ids: eligibleUsers.map((u) => u.id),
      });

      if (res.affected_count > 0) {
        toast.success(
          t("admin.bulk_delete_success", "Successfully deleted {{count}} users", {
            count: res.affected_count,
          }),
        );
      }
      if (res.skipped_count > 0 && res.skipped_reasons?.length) {
        toast.info(
          t("admin.bulk_skip_notice", "Skipped {{count}} users due to safety constraints", {
            count: res.skipped_count,
          }),
        );
      }

      onSuccess();
      onClose();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : t("admin.delete_failed", "Failed to delete users");
      toast.error(msg);
    }
  };

  return (
    <div className="modal modal-open">
      <div className="modal-box relative border border-error/30 shadow-2xl max-w-lg max-h-[80dvh] max-h-[80svh] sm:max-h-[85vh] overflow-y-auto p-4 sm:p-6 pb-6">
        <h3 className="font-bold text-lg text-error flex items-center gap-2">
          <AlertTriangle className="h-5 w-5" />
          {t("admin.bulk_delete_title", "Delete Selected Users")}
        </h3>

        <div className="py-4 space-y-3 text-sm text-base-content/80">
          <p>
            {t(
              "admin.bulk_delete_confirm",
              "Are you sure you want to delete {{count}} eligible users? Their active sessions will be revoked.",
              { count: eligibleUsers.length },
            )}
          </p>

          {skippedList.length > 0 && (
            <div className="p-3 bg-warning/10 border border-warning/30 rounded-xl space-y-1.5 text-xs text-base-content/80">
              <div className="font-bold flex items-center gap-1.5 text-warning">
                <ShieldAlert className="w-4 h-4 shrink-0" />
                <span>
                  {t(
                    "admin.bulk_skipped_accounts_warning",
                    "{{count}} accounts will be automatically skipped for safety:",
                    { count: skippedList.length },
                  )}
                </span>
              </div>
              <ul className="list-disc list-inside space-y-0.5 text-base-content/80">
                {skippedList.map(({ user, reasonKey }) => (
                  <li key={user.id} className="truncate">
                    <span className="font-semibold text-base-content">{user.full_name || user.email}</span>:{" "}
                    <span>{t(reasonKey, "Protected account")}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {eligibleUsers.length === 0 && (
            <div className="alert alert-error text-xs">
              <span>{t("admin.bulk_none_eligible", "None of the selected accounts can be deleted.")}</span>
            </div>
          )}
        </div>

        <div className="modal-action sticky bottom-0 bg-base-100/95 backdrop-blur-xs py-2.5 -mx-4 px-4 sm:-mx-6 sm:px-6 border-t border-base-200 mt-4 z-10">
          <button
            type="button"
            className="btn btn-ghost btn-sm"
            onClick={onClose}
            disabled={deleteMutation.isPending}
          >
            {t("common.cancel", "Cancel")}
          </button>
          <button
            type="button"
            className="btn btn-error btn-sm gap-2"
            onClick={handleDelete}
            disabled={deleteMutation.isPending || eligibleUsers.length === 0}
          >
            {deleteMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Trash2 className="h-4 w-4" />
            )}
            {t("admin.delete_count", "Delete ({{count}})", { count: eligibleUsers.length })}
          </button>
        </div>
      </div>
      <div className="modal-backdrop" onClick={onClose} />
    </div>
  );
};
