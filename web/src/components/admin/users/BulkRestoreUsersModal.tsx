import React, { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { RotateCcw, Loader2, Info } from "lucide-react";
import { toast } from "react-toastify";
import { useBulkRestoreUsersMutation } from "@/hooks/useAdminQueries";
import type { User } from "@/types";
import { isAdminUser } from "@/utils/permission";

interface BulkRestoreUsersModalProps {
  isOpen: boolean;
  selectedUsers: User[];
  isCallerOwner?: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export const BulkRestoreUsersModal: React.FC<BulkRestoreUsersModalProps> = ({
  isOpen,
  selectedUsers,
  isCallerOwner = false,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation();
  const restoreMutation = useBulkRestoreUsersMutation();

  const { eligibleUsers, skippedList } = useMemo(() => {
    const eligible: User[] = [];
    const skipped: { user: User; reasonKey: string }[] = [];

    for (const user of selectedUsers) {
      if (!user.is_deleted) {
        skipped.push({ user, reasonKey: "admin.bulk_skip_not_deleted" });
        continue;
      }
      if (user.is_owner) {
        skipped.push({ user, reasonKey: "admin.bulk_skip_owner_role" });
        continue;
      }
      if (isAdminUser(user) && !isCallerOwner) {
        skipped.push({ user, reasonKey: "admin.bulk_skip_admin_restore" });
        continue;
      }
      eligible.push(user);
    }

    return { eligibleUsers: eligible, skippedList: skipped };
  }, [selectedUsers, isCallerOwner]);

  if (!isOpen) return null;

  const handleRestore = async () => {
    if (eligibleUsers.length === 0) return;
    try {
      const res = await restoreMutation.mutateAsync({
        user_ids: eligibleUsers.map((u) => u.id),
      });

      if (res.affected_count > 0) {
        toast.success(
          t("admin.bulk_restore_success", "Successfully restored {{count}} users", {
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
      const msg = err instanceof Error ? err.message : t("admin.restore_failed", "Failed to restore users");
      toast.error(msg);
    }
  };

  return (
    <div className="modal modal-open">
      <div className="modal-box relative border border-success/30 shadow-2xl max-w-lg max-h-[80dvh] sm:max-h-[85vh] overflow-y-auto p-4 sm:p-6 pb-6">
        <h3 className="font-bold text-lg text-success flex items-center gap-2">
          <RotateCcw className="h-5 w-5" />
          {t("admin.bulk_restore_title", "Restore Selected Users")}
        </h3>

        <div className="py-4 space-y-3 text-sm text-base-content/80">
          <p>
            {t(
              "admin.bulk_restore_confirm",
              "Are you sure you want to restore {{count}} deleted users back to active status?",
              { count: eligibleUsers.length },
            )}
          </p>

          {skippedList.length > 0 && (
            <div className="p-3 bg-base-200/60 rounded-xl space-y-1.5 text-xs text-base-content/80">
              <div className="font-bold flex items-center gap-1.5 text-base-content">
                <Info className="w-4 h-4 text-info shrink-0" />
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
                    <span>{t(reasonKey, "Already active")}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {eligibleUsers.length === 0 && (
            <div className="alert alert-warning text-xs">
              <span>{t("admin.bulk_none_eligible_restore", "None of the selected accounts are in deleted state.")}</span>
            </div>
          )}
        </div>

        <div className="modal-action sticky bottom-0 bg-base-100/95 backdrop-blur-xs py-2.5 -mx-4 px-4 sm:-mx-6 sm:px-6 border-t border-base-200 mt-4 z-10">
          <button
            type="button"
            className="btn btn-ghost btn-sm"
            onClick={onClose}
            disabled={restoreMutation.isPending}
          >
            {t("common.cancel", "Cancel")}
          </button>
          <button
            type="button"
            className="btn btn-success btn-sm gap-2"
            onClick={handleRestore}
            disabled={restoreMutation.isPending || eligibleUsers.length === 0}
          >
            {restoreMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <RotateCcw className="h-4 w-4" />
            )}
            {t("admin.restore_count", "Restore ({{count}})", { count: eligibleUsers.length })}
          </button>
        </div>
      </div>
      <div className="modal-backdrop" onClick={onClose} />
    </div>
  );
};
