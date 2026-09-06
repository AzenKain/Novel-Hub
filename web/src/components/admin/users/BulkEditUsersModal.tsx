import React, { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { UserCog, Loader2, ShieldAlert } from "lucide-react";
import { toast } from "react-toastify";
import { useBulkUpdateUserInfoMutation } from "@/hooks/useAdminQueries";
import type { User } from "@/types";
import { isAdminUser } from "@/utils/permission";

interface BulkEditUsersModalProps {
  isOpen: boolean;
  selectedUsers: User[];
  currentUserId?: string;
  isCallerOwner?: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

const AGE_RATINGS = ["G", "PG", "PG-13", "R15+", "R18+"] as const;

export const BulkEditUsersModal: React.FC<BulkEditUsersModalProps> = ({
  isOpen,
  selectedUsers,
  currentUserId,
  isCallerOwner = false,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation();
  const updateInfoMutation = useBulkUpdateUserInfoMutation();

  // Field toggles
  const [changeAgeRating, setChangeAgeRating] = useState(false);
  const [ageRating, setAgeRating] = useState<string>("R18+");

  const [changeKidsMode, setChangeKidsMode] = useState(false);
  const [kidsModeValue, setKidsModeValue] = useState(false);

  const [resetAvatar, setResetAvatar] = useState(false);
  const [revokeSessions, setRevokeSessions] = useState(false);

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

  const hasAnyChange =
    changeAgeRating || changeKidsMode || resetAvatar || revokeSessions;

  const handleSave = async () => {
    if (!hasAnyChange) {
      toast.warn(
        t("admin.select_at_least_one_field", "Please enable at least one field to update"),
      );
      return;
    }
    if (eligibleUsers.length === 0) return;

    try {
      const res = await updateInfoMutation.mutateAsync({
        user_ids: eligibleUsers.map((u) => u.id),
        max_allowed_age_rating: changeAgeRating ? ageRating : undefined,
        is_kids_mode: changeKidsMode ? kidsModeValue : undefined,
        reset_avatar: resetAvatar ? true : undefined,
        revoke_sessions: revokeSessions ? true : undefined,
      });

      if (res.affected_count > 0) {
        toast.success(
          t("admin.bulk_info_success", "Successfully updated info for {{count}} users", {
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
      const msg = err instanceof Error ? err.message : t("admin.update_failed", "Failed to update users");
      toast.error(msg);
    }
  };

  return (
    <div className="modal modal-open">
      <div className="modal-box relative border border-primary/30 shadow-2xl max-w-lg">
        <h3 className="font-bold text-lg text-primary flex items-center gap-2">
          <UserCog className="h-5 w-5" />
          {t("admin.bulk_edit_info_title", "Bulk Edit User Information")}
        </h3>

        <div className="py-4 space-y-4 text-sm">
          <p className="text-xs text-base-content/70">
            {t(
              "admin.bulk_edit_info_hint",
              "Toggle on the specific fields you wish to modify for the {{count}} selected users. Untoggled fields remain unchanged.",
              { count: eligibleUsers.length },
            )}
          </p>

          {/* Age Rating Toggle & Select */}
          <div className="p-3 bg-base-200/40 rounded-xl border border-base-200 space-y-2">
            <label className="flex items-center justify-between cursor-pointer select-none">
              <span className="font-semibold text-xs flex items-center gap-2">
                <span>{t("admin.bulk_field_age_rating", "Max Allowed Age Rating")}</span>
              </span>
              <input
                type="checkbox"
                className="toggle toggle-primary toggle-sm"
                checked={changeAgeRating}
                onChange={(e) => setChangeAgeRating(e.target.checked)}
              />
            </label>
            {changeAgeRating && (
              <div className="pt-1">
                <select
                  value={ageRating}
                  onChange={(e) => setAgeRating(e.target.value)}
                  className="select select-sm select-bordered w-full"
                >
                  {AGE_RATINGS.map((rating) => (
                    <option key={rating} value={rating}>
                      {rating}
                    </option>
                  ))}
                </select>
              </div>
            )}
          </div>

          {/* Kids Mode Toggle & Select */}
          <div className="p-3 bg-base-200/40 rounded-xl border border-base-200 space-y-2">
            <label className="flex items-center justify-between cursor-pointer select-none">
              <span className="font-semibold text-xs">
                {t("admin.bulk_field_kids_mode", "Kids Mode")}
              </span>
              <input
                type="checkbox"
                className="toggle toggle-primary toggle-sm"
                checked={changeKidsMode}
                onChange={(e) => setChangeKidsMode(e.target.checked)}
              />
            </label>
            {changeKidsMode && (
              <div className="pt-1 flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => setKidsModeValue(true)}
                  className={`btn btn-xs flex-1 ${kidsModeValue ? "btn-primary" : "btn-outline text-base-content/70"}`}
                >
                  {t("admin.kids_mode_enable", "Enable Kids Mode")}
                </button>
                <button
                  type="button"
                  onClick={() => setKidsModeValue(false)}
                  className={`btn btn-xs flex-1 ${!kidsModeValue ? "btn-neutral" : "btn-outline text-base-content/70"}`}
                >
                  {t("admin.kids_mode_disable", "Disable Kids Mode")}
                </button>
              </div>
            )}
          </div>

          {/* Reset Avatar Checkbox */}
          <div className="p-3 bg-base-200/40 rounded-xl border border-base-200">
            <label className="flex items-center justify-between cursor-pointer select-none">
              <div>
                <div className="font-semibold text-xs">
                  {t("admin.bulk_field_reset_avatar", "Reset Profile Avatar")}
                </div>
                <div className="text-[11px] text-base-content/60">
                  {t("admin.bulk_field_reset_avatar_desc", "Clears custom uploaded avatars and reverts to default initials.")}
                </div>
              </div>
              <input
                type="checkbox"
                className="checkbox checkbox-primary checkbox-sm"
                checked={resetAvatar}
                onChange={(e) => setResetAvatar(e.target.checked)}
              />
            </label>
          </div>

          {/* Revoke Sessions Checkbox */}
          <div className="p-3 bg-base-200/40 rounded-xl border border-base-200">
            <label className="flex items-center justify-between cursor-pointer select-none">
              <div>
                <div className="font-semibold text-xs text-warning flex items-center gap-1.5">
                  <ShieldAlert className="w-3.5 h-3.5" />
                  <span>{t("admin.bulk_field_revoke_sessions", "Force Session Revocation")}</span>
                </div>
                <div className="text-[11px] text-base-content/60">
                  {t("admin.bulk_field_revoke_sessions_desc", "Instantly logs out the users across all active devices and browsers.")}
                </div>
              </div>
              <input
                type="checkbox"
                className="checkbox checkbox-warning checkbox-sm"
                checked={revokeSessions}
                onChange={(e) => setRevokeSessions(e.target.checked)}
              />
            </label>
          </div>

          {/* Safety warnings */}
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
              <span>{t("admin.bulk_none_eligible_info", "None of the selected accounts can be modified.")}</span>
            </div>
          )}
        </div>

        <div className="modal-action">
          <button
            type="button"
            className="btn btn-ghost btn-sm"
            onClick={onClose}
            disabled={updateInfoMutation.isPending}
          >
            {t("common.cancel", "Cancel")}
          </button>
          <button
            type="button"
            className="btn btn-primary btn-sm gap-2"
            onClick={handleSave}
            disabled={
              updateInfoMutation.isPending ||
              !hasAnyChange ||
              eligibleUsers.length === 0
            }
          >
            {updateInfoMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <UserCog className="h-4 w-4" />
            )}
            {t("admin.update_count", "Update ({{count}})", { count: eligibleUsers.length })}
          </button>
        </div>
      </div>
      <div className="modal-backdrop" onClick={onClose} />
    </div>
  );
};
