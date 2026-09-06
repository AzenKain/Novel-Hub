import React, { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Shield, Loader2, ShieldCheck, ShieldX, ShieldAlert } from "lucide-react";
import { toast } from "react-toastify";
import { useBulkChangeUserRolesMutation } from "@/hooks/useAdminQueries";
import type { Role, User } from "@/types";
import { isAdminUser } from "@/utils/permission";

interface BulkChangeRolesModalProps {
  isOpen: boolean;
  selectedUsers: User[];
  roles: Role[];
  currentUserId?: string;
  isCallerOwner?: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export const BulkChangeRolesModal: React.FC<BulkChangeRolesModalProps> = ({
  isOpen,
  selectedUsers,
  roles,
  currentUserId,
  isCallerOwner = false,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation();
  const changeRolesMutation = useBulkChangeUserRolesMutation();

  const [actionMode, setActionMode] = useState<"assign" | "unassign" | "replace">("assign");
  const [selectedRoleIds, setSelectedRoleIds] = useState<string[]>([]);

  // Eligible users: must be active (not deleted), not owner, not caller self, and authorized
  const { eligibleUsers, skippedList } = useMemo(() => {
    const eligible: User[] = [];
    const skipped: { user: User; reasonKey: string }[] = [];

    for (const user of selectedUsers) {
      if (user.is_deleted) {
        skipped.push({ user, reasonKey: "admin.bulk_skip_already_deleted" });
        continue;
      }
      if (user.is_owner) {
        skipped.push({ user, reasonKey: "admin.bulk_skip_owner_role" });
        continue;
      }
      if (user.id === currentUserId) {
        skipped.push({ user, reasonKey: "admin.bulk_skip_self_role" });
        continue;
      }
      if (isAdminUser(user) && !isCallerOwner) {
        skipped.push({ user, reasonKey: "admin.bulk_skip_admin" });
        continue;
      }
      eligible.push(user);
    }

    return {
      eligibleUsers: eligible,
      skippedList: skipped,
    };
  }, [selectedUsers, currentUserId, isCallerOwner]);

  if (!isOpen) return null;

  const toggleRole = (roleId: string) => {
    setSelectedRoleIds((prev) =>
      prev.includes(roleId) ? prev.filter((id) => id !== roleId) : [...prev, roleId],
    );
  };

  const handleSave = async () => {
    if (selectedRoleIds.length === 0) {
      toast.warn(t("admin.select_at_least_one_role", "Please select at least one role"));
      return;
    }
    if (eligibleUsers.length === 0) return;

    try {
      const res = await changeRolesMutation.mutateAsync({
        user_ids: eligibleUsers.map((u) => u.id),
        action: actionMode,
        role_ids: selectedRoleIds,
      });

      if (res.affected_count > 0) {
        toast.success(
          t("admin.bulk_roles_success", "Successfully updated roles for {{count}} users", {
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
      const msg = err instanceof Error ? err.message : t("admin.role_change_failed", "Failed to change roles");
      toast.error(msg);
    }
  };

  return (
    <div className="modal modal-open">
      <div className="modal-box relative border border-primary/30 shadow-2xl max-w-lg max-h-[80dvh] sm:max-h-[85vh] overflow-y-auto p-4 sm:p-6 pb-6">
        <h3 className="font-bold text-lg text-primary flex items-center gap-2">
          <Shield className="h-5 w-5" />
          {t("admin.bulk_change_roles_title", "Bulk Change Roles")}
        </h3>

        <div className="py-4 space-y-4 text-sm">
          {/* Action mode selector */}
          <div className="form-control">
            <label className="label">
              <span className="label-text font-semibold">
                {t("admin.bulk_role_action_mode", "Operation Mode")}
              </span>
            </label>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
              <button
                type="button"
                onClick={() => setActionMode("assign")}
                className={`btn btn-sm h-auto min-h-9 py-2 px-3 flex items-center justify-center gap-2 ${
                  actionMode === "assign"
                    ? "btn-primary font-bold shadow-xs"
                    : "btn-outline text-base-content/70 hover:bg-base-200"
                }`}
              >
                <ShieldCheck className="w-4 h-4 shrink-0" />
                <span className="text-xs font-semibold whitespace-nowrap">
                  {t("admin.role_action_assign", "Add Roles")}
                </span>
              </button>
              <button
                type="button"
                onClick={() => setActionMode("unassign")}
                className={`btn btn-sm h-auto min-h-9 py-2 px-3 flex items-center justify-center gap-2 ${
                  actionMode === "unassign"
                    ? "btn-warning font-bold shadow-xs"
                    : "btn-outline text-base-content/70 hover:bg-base-200"
                }`}
              >
                <ShieldX className="w-4 h-4 shrink-0" />
                <span className="text-xs font-semibold whitespace-nowrap">
                  {t("admin.role_action_unassign", "Remove Roles")}
                </span>
              </button>
              <button
                type="button"
                onClick={() => setActionMode("replace")}
                className={`btn btn-sm h-auto min-h-9 py-2 px-3 flex items-center justify-center gap-2 ${
                  actionMode === "replace"
                    ? "btn-secondary font-bold shadow-xs"
                    : "btn-outline text-base-content/70 hover:bg-base-200"
                }`}
              >
                <Shield className="w-4 h-4 shrink-0" />
                <span className="text-xs font-semibold whitespace-nowrap">
                  {t("admin.role_action_replace", "Replace All")}
                </span>
              </button>
            </div>
            <p className="text-[11px] text-base-content/60 mt-1.5 px-1">
              {actionMode === "assign" && t("admin.role_mode_assign_desc", "Append selected roles without removing current roles.")}
              {actionMode === "unassign" && t("admin.role_mode_unassign_desc", "Remove selected roles from users if they possess them.")}
              {actionMode === "replace" && t("admin.role_mode_replace_desc", "Overwrite and set the exact role list for all selected users.")}
            </p>
          </div>

          {/* Roles checklist */}
          <div className="form-control">
            <label className="label">
              <span className="label-text font-semibold">
                {t("admin.roles", "Roles")} ({selectedRoleIds.length} {t("admin.selected", "selected")})
              </span>
            </label>
            <div className="space-y-1.5 max-h-52 overflow-y-auto pr-1">
              {roles.map((role) => {
                const isAdminRole = role.is_admin || role.name === "ADMIN";
                const isBannedRole = role.is_banned || role.name === "BANNED";
                const isDisabled = isAdminRole && !isCallerOwner;

                return (
                  <label
                    key={role.id}
                    className={`flex items-center justify-between p-2.5 rounded-lg border transition-colors cursor-pointer select-none ${
                      selectedRoleIds.includes(role.id)
                        ? "border-primary/40 bg-primary/5"
                        : "border-base-200 bg-base-200/20 hover:bg-base-200/40"
                    } ${isDisabled ? "opacity-40 cursor-not-allowed" : ""}`}
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <input
                        type="checkbox"
                        checked={selectedRoleIds.includes(role.id)}
                        disabled={isDisabled}
                        onChange={() => toggleRole(role.id)}
                        className="checkbox checkbox-primary checkbox-xs"
                      />
                      <div className="min-w-0">
                        <div className="font-semibold text-xs flex items-center gap-1.5">
                          <span className="truncate">{role.name}</span>
                          {isAdminRole && (
                            <span className="badge badge-xs badge-error font-mono">ADMIN</span>
                          )}
                          {isBannedRole && (
                            <span className="badge badge-xs badge-neutral font-mono">BANNED</span>
                          )}
                        </div>
                        {role.description && (
                          <p className="text-[10px] text-base-content/60 truncate max-w-xs">
                            {role.description}
                          </p>
                        )}
                      </div>
                    </div>
                  </label>
                );
              })}
            </div>
          </div>

          {/* Skipped users warning */}
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
            <div className="alert alert-warning text-xs">
              <span>{t("admin.bulk_none_eligible_roles", "None of the selected accounts can be modified.")}</span>
            </div>
          )}
        </div>

        <div className="modal-action sticky bottom-0 bg-base-100/95 backdrop-blur-xs py-2.5 -mx-4 px-4 sm:-mx-6 sm:px-6 border-t border-base-200 mt-4 z-10">
          <button
            type="button"
            className="btn btn-ghost btn-sm"
            onClick={onClose}
            disabled={changeRolesMutation.isPending}
          >
            {t("common.cancel", "Cancel")}
          </button>
          <button
            type="button"
            className="btn btn-primary btn-sm gap-2"
            onClick={handleSave}
            disabled={
              changeRolesMutation.isPending ||
              selectedRoleIds.length === 0 ||
              eligibleUsers.length === 0
            }
          >
            {changeRolesMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Shield className="h-4 w-4" />
            )}
            {t("admin.apply_count", "Apply ({{count}})", { count: eligibleUsers.length })}
          </button>
        </div>
      </div>
      <div className="modal-backdrop" onClick={onClose} />
    </div>
  );
};
