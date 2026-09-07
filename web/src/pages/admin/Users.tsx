import {
  UserTable,
  SendEmailModal,
  BulkDeleteUsersModal,
  BulkRestoreUsersModal,
  BulkChangeRolesModal,
  BulkEditUsersModal,
  BulkSendEmailModal,
} from "@/components/admin";
import { PasswordStrength, ImageCropperModal, ConfirmModal } from "@/components/common";
import { getMediaUrl } from "@/config/api";
import {
  useAdminUploadAvatarMutation,
  useChangeUserRolesMutation,
  useCreateUserMutation,
  useDeleteUserMutation,
  useResetUserPasswordMutation,
  useRevokeUserSessionsMutation,
  useRolesQuery,
  useSendUserEmailMutation,
  useUpdateUserMutation,
  useUsersQuery,
} from "@/hooks";
import { adminService } from "@/services";
import { useUserAdminStore, useAuthStore } from "@/stores";
import type { CreateUserRequest, User } from "@/types";
import {
  AlertCircle,
  Lock,
  LogOut,
  Mail,
  RefreshCw,
  RotateCcw,
  Search,
  Shield,
  ShieldAlert,
  Trash2,
  UserCog,
  UserPlus,
  X,
} from "lucide-react";
import { SyntheticEvent, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useDebounce } from "@/hooks/useDebounce";
import { toast } from "react-toastify";
import { useQueryClient } from "@tanstack/react-query";
import { useShallow } from "zustand/react/shallow";

const AGE_RATINGS = ["G", "PG", "PG-13", "R15+", "R18+"] as const;

const emptyCreate: CreateUserRequest = {
  email: "",
  password: "",
  full_name: "",
  avatar_url: "",
  role_ids: [],
};

export function Users() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const currentUser = useAuthStore((state) => state.user);

  const {
    selectedUser: selected,
    setSelectedUser: setSelected,
    query,
    setQuery,
    showDeleted,
    setShowDeleted,
    error,
    setError,
    modal,
    setModal,
    form,
    setForm,
    newPassword,
    setNewPassword,
    roleIDs,
    setRoleIDs,
    userToDelete,
    setUserToDelete,
    selectedUserIds,
    setSelectedUserIds,
    clearSelection,
    bulkModal,
    setBulkModal,
  } = useUserAdminStore(
    useShallow((state) => ({
      selectedUser: state.selectedUser,
      setSelectedUser: state.setSelectedUser,
      query: state.query,
      setQuery: state.setQuery,
      showDeleted: state.showDeleted,
      setShowDeleted: state.setShowDeleted,
      error: state.error,
      setError: state.setError,
      modal: state.modal,
      setModal: state.setModal,
      form: state.form,
      setForm: state.setForm,
      newPassword: state.newPassword,
      setNewPassword: state.setNewPassword,
      roleIDs: state.roleIDs,
      setRoleIDs: state.setRoleIDs,
      userToDelete: state.userToDelete,
      setUserToDelete: state.setUserToDelete,
      selectedUserIds: state.selectedUserIds,
      setSelectedUserIds: state.setSelectedUserIds,
      clearSelection: state.clearSelection,
      bulkModal: state.bulkModal,
      setBulkModal: state.setBulkModal,
    })),
  );
  const debouncedSearch = useDebounce(query || "", 400);

  const [cursor, setCursor] = useState("");
  const [cursorHistory, setCursorHistory] = useState<string[]>([]);
  const resetPaging = () => {
    setCursor("");
    setCursorHistory([]);
  };
  const {
    data: usersData,
    isLoading: usersLoading,
    isFetching: usersFetching,
    refetch: refetchUsers,
  } = useUsersQuery({
    cursor: cursor || undefined,
    limit: 50,
    search: debouncedSearch || undefined,
    is_deleted: showDeleted ? undefined : false,
    sort: "created_at",
    order: "desc",
  });

  const {
    data: roles = [],
    isLoading: rolesLoading,
    isFetching: rolesFetching,
    refetch: refetchRoles,
  } = useRolesQuery();

  const createUserMutation = useCreateUserMutation();
  const updateUserMutation = useUpdateUserMutation();
  const resetPasswordMutation = useResetUserPasswordMutation();
  const changeRolesMutation = useChangeUserRolesMutation();
  const deleteUserMutation = useDeleteUserMutation();
  const sendEmailMutation = useSendUserEmailMutation();
  const revokeUserSessionsMutation = useRevokeUserSessionsMutation();

  const users = usersData?.users || [];
  const loading = usersLoading || rolesLoading;
  const saving =
    createUserMutation.isPending ||
    updateUserMutation.isPending ||
    resetPasswordMutation.isPending ||
    changeRolesMutation.isPending ||
    deleteUserMutation.isPending ||
    sendEmailMutation.isPending ||
    revokeUserSessionsMutation.isPending;

  const activeUsers = useMemo(
    () => users.filter((item) => !item.is_deleted).length,
    [users],
  );

  const selectedUsers = useMemo(() => {
    return users.filter((u) => selectedUserIds.includes(u.id));
  }, [users, selectedUserIds]);

  const isAllSelected =
    users.length > 0 && users.every((u) => selectedUserIds.includes(u.id));

  const handleToggleSelectAll = () => {
    if (isAllSelected) {
      clearSelection();
    } else {
      setSelectedUserIds(users.map((u) => u.id));
    }
  };

  const handleToggleSelectUser = (id: string) => {
    setSelectedUserIds((prev) =>
      prev.includes(id) ? prev.filter((item) => item !== id) : [...prev, id],
    );
  };

  useEffect(() => {
    clearSelection();
  }, [debouncedSearch, showDeleted, cursor, clearSelection]);
  const [emailForm, setEmailForm] = useState({ subject: "", body: "" });
  const [confirmPassword, setConfirmPassword] = useState("");
  const [confirmCreatePassword, setConfirmCreatePassword] = useState("");
  const [userToRestore, setUserToRestore] = useState<User | null>(null);
  const [userToRevoke, setUserToRevoke] = useState<User | null>(null);
  const [urlInputOpen, setUrlInputOpen] = useState(false);
  const [tempUrl, setTempUrl] = useState("");
  const [selectedImage, setSelectedImage] = useState<string | null>(null);
  const [uploadingAvatar, setUploadingAvatar] = useState(false);
  const adminUploadAvatarMutation = useAdminUploadAvatarMutation();

  const base64ToBlob = (base64: string): Blob => {
    const parts = base64.split(";base64,");
    const contentType = parts[0].split(":")[1];
    const raw = window.atob(parts[1]);
    const rawLength = raw.length;
    const uInt8Array = new Uint8Array(rawLength);
    for (let i = 0; i < rawLength; ++i) {
      uInt8Array[i] = raw.charCodeAt(i);
    }
    return new Blob([uInt8Array], { type: contentType });
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (event) => {
        if (event.target?.result) {
          setSelectedImage(event.target.result as string);
          setUrlInputOpen(false);
        }
      };
      reader.readAsDataURL(file);
    }
    e.target.value = "";
  };

  const handleUrlSubmit = async () => {
    const trimmed = tempUrl.trim();
    if (!trimmed) return;
    try {
      const res = await fetch(trimmed);
      if (!res.ok) throw new Error("Fetch failed");
      const blob = await res.blob();
      const reader = new FileReader();
      reader.onload = (event) => {
        if (event.target?.result) {
          setSelectedImage(event.target.result as string);
          setUrlInputOpen(false);
        }
      };
      reader.readAsDataURL(blob);
    } catch {
      setSelectedImage(trimmed);
      setUrlInputOpen(false);
    }
  };

  const handleCropApply = async (base64: string) => {
    if (!selected) return;
    setSelectedImage(null);
    setUploadingAvatar(true);
    try {
      const blob = base64ToBlob(base64);
      const file = new File([blob], "avatar.png", { type: blob.type });
      const uploadedUrl = await adminUploadAvatarMutation.mutateAsync({
        id: selected.id,
        file,
      });
      setForm((prev) => ({ ...prev, avatar_url: uploadedUrl }));
      toast.success(
        t("user.avatar_updated_success", "Avatar updated successfully!"),
      );
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : t("user.avatar_upload_failed", "Failed to upload avatar"),
      );
    } finally {
      setUploadingAvatar(false);
    }
  };

  const handleRemoveAvatar = () => {
    setForm((prev) => ({ ...prev, avatar_url: "" }));
  };

  function closeModal() {
    setModal(null);
    setSelectedImage(null);
    setUrlInputOpen(false);
    setTempUrl("");
  }

  function openCreate() {
    setForm({
      ...emptyCreate,
      role_ids: roles.filter((role) => role.auto_assign).map((role) => role.id),
    });
    setConfirmCreatePassword("");
    setModal("create");
    setError("");
  }

  function openEdit(target: User) {
    setSelected(target);
    setForm({
      email: target.email,
      password: "",
      full_name: target.full_name,
      avatar_url: target.avatar_url,
      role_ids: target.roles.map((role) => role.id),
      is_kids_mode: Boolean(target.is_kids_mode),
      max_allowed_age_rating: target.max_allowed_age_rating || "R18+",
    });
    setSelectedImage(null);
    setUrlInputOpen(false);
    setTempUrl("");
    setModal("edit");
    setError("");
  }

  function openPassword(target: User) {
    setSelected(target);
    setNewPassword("");
    setConfirmPassword("");
    setModal("password");
    setError("");
  }

  function openRoles(target: User) {
    setSelected(target);
    setRoleIDs(target.roles.map((role) => role.id));
    setModal("roles");
    setError("");
  }

  function openEmail(target: User) {
    setSelected(target);
    setEmailForm({ subject: "", body: "" });
    setModal("email");
    setError("");
  }

  function handleSendEmail(event: SyntheticEvent) {
    event.preventDefault();
    if (!selected) return;
    setError("");
    sendEmailMutation.mutate(
      { id: selected.id, data: emailForm },
      {
        onSuccess: () => {
          toast.success(t("admin.email_sent", "Email sent"));
          setModal(null);
        },
        onError: (err) =>
          setError(err instanceof Error ? err.message : String(err)),
      },
    );
  }

  function handleCreate(event: SyntheticEvent) {
    event.preventDefault();
    setError("");
    if (form.password.length < 8) {
      setError(t("auth.password_min", "Minimum 8 characters"));
      return;
    }
    if (form.password !== confirmCreatePassword) {
      setError(t("auth.passwords_do_not_match", "New passwords do not match"));
      return;
    }
    createUserMutation.mutate(form, {
      onSuccess: () => {
        toast.success(t("common.success", "Success"));
        setModal(null);
        setConfirmCreatePassword("");
      },
      onError: (err) =>
        setError(err instanceof Error ? err.message : String(err)),
    });
  }

  function handleEdit(event: SyntheticEvent) {
    event.preventDefault();
    if (!selected) return;
    setError("");
    updateUserMutation.mutate(
      {
        id: selected.id,
        data: {
          full_name: form.full_name,
          avatar_url: form.avatar_url,
          is_kids_mode: form.is_kids_mode,
          max_allowed_age_rating: form.max_allowed_age_rating,
        },
      },
      {
        onSuccess: () => {
          toast.success(t("common.success", "Success"));
          closeModal();
        },
        onError: (err) =>
          setError(err instanceof Error ? err.message : String(err)),
      },
    );
  }

  function handleRevokeSessions(targetUser: User) {
    setUserToRevoke(targetUser);
  }

  function confirmRevokeSessions() {
    if (!userToRevoke) return;
    const targetUser = userToRevoke;
    revokeUserSessionsMutation.mutate(targetUser.id, {
      onSuccess: () => {
        toast.success(
          t(
            "admin.revoke_sessions_success",
            "User sessions revoked successfully across all devices",
          ),
        );
        setUserToRevoke(null);
      },
      onError: (err) => {
        toast.error(err instanceof Error ? err.message : String(err));
        setUserToRevoke(null);
      },
    });
  }

  function handlePassword(event: SyntheticEvent) {
    event.preventDefault();
    if (!selected) return;
    setError("");
    if (newPassword.length < 8) {
      setError(t("auth.password_min", "Minimum 8 characters"));
      return;
    }
    if (newPassword !== confirmPassword) {
      setError(t("auth.passwords_do_not_match", "New passwords do not match"));
      return;
    }
    resetPasswordMutation.mutate(
      { id: selected.id, password: newPassword },
      {
        onSuccess: () => {
          toast.success(t("common.success", "Success"));
          setModal(null);
          setNewPassword("");
          setConfirmPassword("");
        },
        onError: (err) =>
          setError(err instanceof Error ? err.message : String(err)),
      },
    );
  }

  function handleRoles(event: SyntheticEvent) {
    event.preventDefault();
    if (!selected) return;
    setError("");
    changeRolesMutation.mutate(
      { id: selected.id, roleIDs },
      {
        onSuccess: () => {
          toast.success(t("common.success", "Success"));
          setModal(null);
        },
        onError: (err) =>
          setError(err instanceof Error ? err.message : String(err)),
      },
    );
  }

  function confirmDeleteUser() {
    if (!userToDelete) return;
    setError("");
    deleteUserMutation.mutate(userToDelete.id, {
      onSuccess: () => {
        toast.success(t("common.success", "Success"));
        setUserToDelete(null);
      },
      onError: (err) =>
        setError(err instanceof Error ? err.message : String(err)),
    });
  }

  function handleRestore(target: User) {
    setUserToRestore(target);
  }

  async function confirmRestoreUser() {
    if (!userToRestore) return;
    setError("");
    try {
      await adminService.restoreUser(userToRestore.id);
      toast.success(t("common.success", "Success"));
      setUserToRestore(null);
      void refetchUsers();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <div className="flex flex-col h-full bg-base-100">
      <header className="px-4 py-4 sm:px-6 lg:px-8 lg:py-6 border-b border-base-200 flex items-center justify-between gap-3 sm:gap-4 bg-base-100/50 backdrop-blur-xl sticky top-0 z-10">
        <div className="min-w-0 flex-1">
          <h1 className="text-xl sm:text-2xl font-bold tracking-tight">
            {t("admin.user_management", "User Management")}
          </h1>
          <p className="text-xs sm:text-sm text-base-content/60 mt-0.5 sm:mt-1 line-clamp-1 sm:line-clamp-none">
            {t(
              "admin.user_subtitle",
              "Manage accounts, roles, access levels, and security credentials.",
            )}
          </p>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <button
            onClick={async () => {
              await queryClient.invalidateQueries({
                queryKey: ["admin", "users"],
              });
              await queryClient.invalidateQueries({
                queryKey: ["admin", "roles"],
              });
              await Promise.all([refetchUsers(), refetchRoles()]);
              toast.info(t("common.refreshed", "Data refreshed"));
            }}
            className="btn btn-square btn-ghost btn-sm sm:btn-md"
            title={t("common.refresh")}
            disabled={usersFetching || rolesFetching}
          >
            <RefreshCw
              className={`h-4 w-4 sm:h-5 sm:w-5 ${usersFetching || rolesFetching ? "animate-spin" : ""}`}
            />
          </button>
          <button
            onClick={openCreate}
            className="btn btn-primary btn-sm sm:btn-md gap-2"
          >
            <UserPlus className="w-4 h-4" />
            <span className="hidden xs:inline">
              {t("admin.add_user", "Add User")}
            </span>
          </button>
        </div>
      </header>

      <div className="flex-1 overflow-auto p-3 sm:p-6 lg:p-8 space-y-3.5 sm:space-y-6 max-w-7xl mx-auto w-full">
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-2.5 sm:gap-3 bg-base-200/40 border border-base-200 p-2 sm:p-2.5 rounded-xl sm:rounded-2xl">
          <div className="flex-1 flex flex-col sm:flex-row items-stretch sm:items-center gap-2 sm:gap-2.5">
            <div className="relative flex-1">
              <Search className="absolute left-3 sm:left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-base-content/40 pointer-events-none z-10" />
              <input
                type="text"
                value={query}
                onChange={(e) => {
                  setQuery(e.target.value);
                  resetPaging();
                }}
                placeholder={t(
                  "admin.search_users_placeholder",
                  "Search users by name or email...",
                )}
                className="input input-bordered w-full pl-9 sm:pl-10 pr-8 focus:input-primary h-9 sm:h-10 text-xs sm:text-sm rounded-xl bg-base-100"
              />
              {query && (
                <button
                  type="button"
                  onClick={() => {
                    setQuery("");
                    resetPaging();
                  }}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 z-10 p-1 text-base-content/40 hover:text-base-content rounded-full cursor-pointer"
                  title={t("common.clear", "Clear")}
                >
                  <X className="w-3.5 h-3.5" />
                </button>
              )}
            </div>

            {/* Desktop show deleted checkbox */}
            <label className="hidden sm:flex cursor-pointer items-center gap-2 px-3.5 h-10 rounded-xl border border-base-200 bg-base-100 hover:bg-base-200/50 transition-colors shrink-0">
              <input
                type="checkbox"
                checked={showDeleted}
                onChange={(e) => {
                  setShowDeleted(e.target.checked);
                  resetPaging();
                }}
                className="checkbox checkbox-primary checkbox-xs rounded"
              />
              <span className="text-xs font-medium select-none text-base-content/80">
                {t("admin.show_deleted", "Show Deleted")}
              </span>
            </label>
          </div>

          {/* Mobile sub-bar: Checkbox on left + Compact stats on right */}
          <div className="flex sm:hidden items-center justify-between gap-2 px-1 pt-0.5 text-xs">
            <label className="cursor-pointer flex items-center gap-1.5 font-medium select-none text-base-content/80">
              <input
                type="checkbox"
                checked={showDeleted}
                onChange={(e) => {
                  setShowDeleted(e.target.checked);
                  resetPaging();
                }}
                className="checkbox checkbox-primary checkbox-xs rounded"
              />
              <span className="text-xs">
                {t("admin.show_deleted", "Show Deleted")}
              </span>
            </label>
            <div className="flex items-center gap-2 text-base-content/60 font-medium">
              <span>
                {t("admin.total_loaded", "Loaded Users")}:{" "}
                <strong className="text-primary font-bold">
                  {users.length}
                </strong>
              </span>
              <span className="text-base-content/30">•</span>
              <span>
                {t("admin.active_users", "Active")}:{" "}
                <strong className="text-success font-bold">
                  {activeUsers}
                </strong>
              </span>
            </div>
          </div>

          {/* Desktop Stats Counters */}
          <div className="hidden sm:flex items-center gap-2 shrink-0 sm:border-l sm:border-base-200/80 sm:pl-3">
            <div className="flex items-center gap-2 px-3 h-10 rounded-xl bg-base-100 border border-base-200 text-xs">
              <span className="text-base-content/60 font-medium">
                {t("admin.total_loaded", "Loaded Users")}:
              </span>
              <span className="font-bold text-primary text-sm">
                {users.length}
              </span>
            </div>
            <div className="flex items-center gap-2 px-3 h-10 rounded-xl bg-base-100 border border-base-200 text-xs">
              <span className="text-base-content/60 font-medium">
                {t("admin.active_users", "Active")}:
              </span>
              <span className="font-bold text-success text-sm">
                {activeUsers}
              </span>
            </div>
          </div>
        </div>

        {/* Bulk User Actions Toolbar */}
        {selectedUserIds.length > 0 && (
          <div className="mb-3 px-3 py-2 bg-primary/10 rounded-xl border border-primary/20 flex flex-col sm:flex-row sm:items-center justify-between gap-2 text-xs shadow-sm">
            <div className="flex items-center justify-between sm:justify-start gap-2 shrink-0">
              <span className="font-semibold text-primary whitespace-nowrap">
                {t("admin.selected_users", "Selected {{count}} users", {
                  count: selectedUserIds.length,
                })}
              </span>
              <button
                type="button"
                onClick={clearSelection}
                className="btn btn-ghost btn-xs text-xs opacity-75 hover:opacity-100 h-6 min-h-0 whitespace-nowrap shrink-0 gap-1 px-1.5"
              >
                <X className="w-3 h-3" />
                <span>{t("common.deselect_all", "Clear selection")}</span>
              </button>
            </div>

            <div className="grid grid-cols-2 sm:flex sm:flex-wrap sm:items-center gap-1.5 w-full sm:w-auto pt-2 sm:pt-0 border-t border-primary/15 sm:border-t-0 shrink-0">
              {/* Email */}
              <button
                type="button"
                title={t("admin.bulk_email", "Send Email")}
                onClick={() => setBulkModal("email")}
                className="btn btn-outline btn-xs h-8 min-h-8 px-2.5 gap-1.5 rounded-lg text-xs font-medium justify-center items-center whitespace-nowrap"
              >
                <Mail className="w-3.5 h-3.5 shrink-0" />
                <span className="truncate whitespace-nowrap">{t("admin.bulk_email", "Send Email")}</span>
              </button>

              {/* Change Roles */}
              <button
                type="button"
                title={t("admin.bulk_roles", "Change Roles")}
                onClick={() => setBulkModal("roles")}
                className="btn btn-outline btn-xs h-8 min-h-8 px-2.5 gap-1.5 rounded-lg text-xs font-medium justify-center items-center whitespace-nowrap"
              >
                <Shield className="w-3.5 h-3.5 shrink-0" />
                <span className="truncate whitespace-nowrap">{t("admin.bulk_roles", "Change Roles")}</span>
              </button>

              {/* Edit Info */}
              <button
                type="button"
                title={t("admin.bulk_info", "Edit Info")}
                onClick={() => setBulkModal("info")}
                className={`btn btn-outline btn-xs h-8 min-h-8 px-2.5 gap-1.5 rounded-lg text-xs font-medium justify-center items-center whitespace-nowrap ${
                  !selectedUsers.some((u) => u.is_deleted) && !selectedUsers.some((u) => !u.is_deleted)
                    ? "col-span-2 sm:col-span-1"
                    : ""
                }`}
              >
                <UserCog className="w-3.5 h-3.5 shrink-0" />
                <span className="truncate whitespace-nowrap">{t("admin.bulk_info", "Edit Info")}</span>
              </button>

              {/* Restore (visible if any selected user is deleted) */}
              {selectedUsers.some((u) => u.is_deleted) && (
                <button
                  type="button"
                  title={t("admin.bulk_restore", "Restore")}
                  onClick={() => setBulkModal("restore")}
                  className={`btn btn-success btn-outline btn-xs h-8 min-h-8 px-2.5 gap-1.5 rounded-lg text-xs font-medium justify-center items-center whitespace-nowrap ${
                    !selectedUsers.some((u) => !u.is_deleted) ? "col-span-2 sm:col-span-1" : ""
                  }`}
                >
                  <RotateCcw className="w-3.5 h-3.5 shrink-0" />
                  <span className="truncate whitespace-nowrap">{t("admin.bulk_restore", "Restore")}</span>
                </button>
              )}

              {/* Delete (visible if any selected user is active) */}
              {selectedUsers.some((u) => !u.is_deleted) && (
                <button
                  type="button"
                  title={t("admin.bulk_delete", "Delete selected")}
                  onClick={() => setBulkModal("delete")}
                  className={`btn btn-error btn-xs h-8 min-h-8 px-2.5 gap-1.5 rounded-lg text-xs font-medium justify-center items-center whitespace-nowrap text-error-content ${
                    selectedUsers.some((u) => u.is_deleted) ? "col-span-2 sm:col-span-1" : ""
                  }`}
                >
                  <Trash2 className="w-3.5 h-3.5 shrink-0" />
                  <span className="truncate whitespace-nowrap">{t("common.delete", "Xóa")}</span>
                </button>
              )}
            </div>
          </div>
        )}

        {/* User Table */}
        <UserTable
          users={users}
          t={t}
          onEdit={openEdit}
          onPassword={openPassword}
          onRoles={openRoles}
          onEmail={openEmail}
          onRevokeSessions={handleRevokeSessions}
          onDelete={setUserToDelete}
          onRestore={handleRestore}
          currentUserId={currentUser?.id}
          isCallerOwner={Boolean(currentUser?.is_owner)}
          selectedUserIds={selectedUserIds}
          onToggleSelectAll={handleToggleSelectAll}
          onToggleSelectUser={handleToggleSelectUser}
          isAllSelected={isAllSelected}
        />

        {(cursorHistory.length > 0 || usersData?.nextCursor) && (
          <div className="flex justify-end gap-2 mt-4">
            <button
              className="btn btn-sm"
              disabled={cursorHistory.length === 0}
              onClick={() => {
                setCursorHistory((prev) => {
                  setCursor(prev[prev.length - 1] ?? "");
                  return prev.slice(0, -1);
                });
              }}
            >
              {t("common.previous")}
            </button>
            <button
              className="btn btn-sm"
              disabled={!usersData?.nextCursor}
              onClick={() => {
                if (!usersData?.nextCursor) return;
                setCursorHistory((prev) => [...prev, cursor]);
                setCursor(usersData.nextCursor);
              }}
            >
              {t("common.next")}
            </button>
          </div>
        )}
      </div>

      {/* Modals */}
      {modal === "create" && (
        <dialog className="modal modal-open">
          <div className="modal-box max-w-md max-h-[80dvh] sm:max-h-[85vh] p-0 overflow-hidden flex flex-col">
            <div className="px-4 sm:px-6 py-3.5 sm:py-4 border-b border-base-200 flex items-center justify-between gap-3 shrink-0 bg-base-100">
              <h3 className="font-bold text-lg leading-tight">
                {t("admin.create_user_title", "Create New User")}
              </h3>
              <button
                type="button"
                onClick={() => setModal(null)}
                className="btn btn-ghost btn-circle btn-sm -mr-1.5 text-base-content/70 hover:text-base-content shrink-0"
                aria-label={t("common.close", "Close")}
              >
                <X className="w-4 h-4 sm:w-5 sm:h-5 stroke-[2.5]" />
              </button>
            </div>
            <form onSubmit={handleCreate} className="p-4 sm:p-6 overflow-y-auto flex-1 min-h-0 space-y-4">
              {error && (
                <div className="alert alert-error mb-4 py-2 text-sm rounded-lg flex items-center gap-2">
                  <AlertCircle className="w-4 h-4 shrink-0" />
                  <span>{error}</span>
                </div>
              )}
              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">
                    {t("auth.email", "Email Address")}
                  </span>
                </label>
                <input
                  type="email"
                  required
                  value={form.email}
                  onChange={(e) => setForm({ ...form, email: e.target.value })}
                  placeholder="user@example.com"
                  className="input input-bordered w-full focus:input-primary"
                />
              </div>

              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">
                    {t("user.full_name", "Full Name")}
                  </span>
                </label>
                <input
                  type="text"
                  required
                  value={form.full_name}
                  onChange={(e) =>
                    setForm({ ...form, full_name: e.target.value })
                  }
                  placeholder="John Doe"
                  className="input input-bordered w-full focus:input-primary"
                />
              </div>

              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">
                    {t("auth.password", "Initial Password")}
                  </span>
                </label>
                <input
                  type="password"
                  required
                  minLength={8}
                  value={form.password}
                  onChange={(e) =>
                    setForm({ ...form, password: e.target.value })
                  }
                  placeholder={t("auth.password_min", "Minimum 8 characters")}
                  className="input input-bordered w-full focus:input-primary"
                  autoComplete="new-password"
                />
                <PasswordStrength password={form.password} />
              </div>

              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">
                    {t("settings.confirm_password", "Confirm new password")}
                  </span>
                </label>
                <input
                  type="password"
                  required
                  minLength={8}
                  value={confirmCreatePassword}
                  onChange={(e) => setConfirmCreatePassword(e.target.value)}
                  placeholder={t("auth.password_min", "Minimum 8 characters")}
                  className="input input-bordered w-full focus:input-primary"
                  autoComplete="new-password"
                />
              </div>

              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">
                    {t("admin.assign_roles", "Assign Roles")}
                  </span>
                </label>
                <div className="space-y-2 bg-base-200/50 p-3 rounded-xl border border-base-200">
                  {roles.map((role) => (
                    <label
                      key={role.id}
                      className="label cursor-pointer justify-start gap-3 py-1"
                    >
                      <input
                        type="checkbox"
                        checked={(form.role_ids || []).includes(role.id)}
                        onChange={(e) => {
                          const checked = e.target.checked;
                          const currentIds = form.role_ids || [];
                          setForm((prev) => ({
                            ...prev,
                            role_ids: checked
                              ? [...currentIds, role.id]
                              : currentIds.filter((id) => id !== role.id),
                          }));
                        }}
                        className="checkbox checkbox-primary checkbox-sm"
                      />
                      <div>
                        <span className="font-semibold text-sm">
                          {role.name}
                        </span>
                        {role.description && (
                          <p className="text-xs text-base-content/60">
                            {role.description}
                          </p>
                        )}
                      </div>
                    </label>
                  ))}
                </div>
              </div>

              <div className="modal-action border-t border-base-200 pt-4 mt-6">
                <button
                  type="button"
                  onClick={() => setModal(null)}
                  className="btn btn-ghost"
                >
                  {t("common.cancel", "Cancel")}
                </button>
                <button
                  type="submit"
                  disabled={saving}
                  className="btn btn-primary"
                >
                  {saving ? (
                    <span className="loading loading-spinner"></span>
                  ) : (
                    t("common.create", "Create User")
                  )}
                </button>
              </div>
            </form>
          </div>
          <form method="dialog" className="modal-backdrop">
            <button onClick={() => setModal(null)}>close</button>
          </form>
        </dialog>
      )}

      {modal === "edit" && selected && (
        <dialog className="modal modal-open">
          <div className="modal-box max-w-md max-h-[80dvh] sm:max-h-[85vh] p-0 overflow-hidden flex flex-col">
            <div className="px-4 sm:px-6 py-3.5 sm:py-4 border-b border-base-200 flex items-center justify-between gap-3 shrink-0 bg-base-100">
              <h3 className="font-bold text-lg leading-tight">
                {t("admin.edit_user_title", "Edit Profile")}
              </h3>
              <button
                type="button"
                onClick={closeModal}
                className="btn btn-ghost btn-circle btn-sm -mr-1.5 text-base-content/70 hover:text-base-content shrink-0"
                aria-label={t("common.close", "Close")}
              >
                <X className="w-4 h-4 sm:w-5 sm:h-5 stroke-[2.5]" />
              </button>
            </div>
            <form onSubmit={handleEdit} className="p-4 sm:p-6 overflow-y-auto flex-1 min-h-0 space-y-4">
              {error && (
                <div className="alert alert-error mb-4 py-2 text-sm rounded-lg flex items-center gap-2">
                  <AlertCircle className="w-4 h-4 shrink-0" />
                  <span>{error}</span>
                </div>
              )}
              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">
                    {t("auth.email", "Email Address")}
                  </span>
                </label>
                <input
                  type="email"
                  disabled
                  value={form.email}
                  className="input input-bordered w-full opacity-60 cursor-not-allowed"
                />
              </div>

              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">
                    {t("user.full_name", "Full Name")}
                  </span>
                </label>
                <input
                  type="text"
                  required
                  value={form.full_name}
                  onChange={(e) =>
                    setForm({ ...form, full_name: e.target.value })
                  }
                  className="input input-bordered w-full focus:input-primary"
                />
              </div>

              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">
                    {t("user.avatar", "Avatar")}
                  </span>
                </label>
                <div className="flex items-center gap-4 p-3 rounded-xl border border-base-200 bg-base-200/30">
                  <div className="avatar">
                    <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center border border-primary/20 text-primary font-bold text-xl overflow-hidden shadow-sm shrink-0">
                      {form.avatar_url ? (
                        <img
                          src={getMediaUrl(
                            form.avatar_url,
                            undefined,
                            selected.updated_at,
                          )}
                          alt={t("common.alt_avatar", "Avatar")}
                          loading="lazy"
                          className="object-cover w-full h-full animate-in fade-in duration-300"
                          onError={(e) => {
                            e.currentTarget.style.display = "none";
                            const fallback = e.currentTarget.nextElementSibling;
                            if (fallback) {
                              (fallback as HTMLElement).style.display = "flex";
                            }
                          }}
                        />
                      ) : null}
                      <span
                        className="w-full h-full flex items-center justify-center font-bold text-xl text-primary"
                        style={{ display: form.avatar_url ? "none" : "flex" }}
                      >
                        {form.full_name
                          ? form.full_name.charAt(0).toUpperCase()
                          : selected.email.charAt(0).toUpperCase()}
                      </span>
                    </div>
                  </div>

                  <div className="flex-1 min-w-0 space-y-2">
                    <div className="flex flex-wrap gap-2">
                      <label
                        className={`btn btn-xs btn-primary cursor-pointer ${
                          uploadingAvatar ? "btn-disabled" : ""
                        }`}
                      >
                        <span className="font-medium">
                          {uploadingAvatar ? (
                            <span className="loading loading-spinner loading-xs"></span>
                          ) : (
                            t("user.upload_avatar", "Upload Photo")
                          )}
                        </span>
                        <input
                          type="file"
                          accept="image/*"
                          className="hidden"
                          disabled={uploadingAvatar}
                          onChange={handleFileChange}
                        />
                      </label>
                      <button
                        type="button"
                        onClick={() => setUrlInputOpen(!urlInputOpen)}
                        className="btn btn-xs btn-outline"
                        disabled={uploadingAvatar}
                      >
                        {t("user.load_url", "From URL")}
                      </button>
                      {form.avatar_url && (
                        <button
                          type="button"
                          onClick={handleRemoveAvatar}
                          className="btn btn-xs btn-ghost text-error"
                          disabled={uploadingAvatar}
                        >
                          {t("user.remove_avatar", "Remove")}
                        </button>
                      )}
                    </div>

                    {urlInputOpen && (
                      <div className="flex gap-1.5 items-center">
                        <input
                          type="text"
                          placeholder="https://example.com/avatar.png"
                          value={tempUrl}
                          onChange={(e) => setTempUrl(e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === "Enter") {
                              e.preventDefault();
                              void handleUrlSubmit();
                            }
                          }}
                          className="input input-bordered input-xs flex-1 focus:input-primary font-mono text-xs"
                        />
                        <button
                          type="button"
                          onClick={() => void handleUrlSubmit()}
                          className="btn btn-xs btn-primary font-bold shrink-0"
                        >
                          {t("common.ok", "OK")}
                        </button>
                      </div>
                    )}
                  </div>
                </div>
              </div>

              {/* Max Allowed Age Rating */}
              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">
                    {t("admin.bulk_field_age_rating", "Max Allowed Age Rating")}
                  </span>
                </label>
                <select
                  value={form.max_allowed_age_rating || "R18+"}
                  onChange={(e) =>
                    setForm({ ...form, max_allowed_age_rating: e.target.value })
                  }
                  className="select select-bordered w-full focus:select-primary"
                >
                  {AGE_RATINGS.map((rating) => (
                    <option key={rating} value={rating}>
                      {rating}
                    </option>
                  ))}
                </select>
              </div>

              {/* Kids Mode Toggle */}
              <div className="form-control">
                <label className="flex items-center justify-between gap-3 cursor-pointer select-none py-1">
                  <div className="flex-1 min-w-0 pr-2">
                    <span className="label-text font-semibold text-sm block text-base-content">
                      {t("admin.bulk_field_kids_mode", "Kids Mode")}
                    </span>
                    <span className="text-xs text-base-content/60 leading-relaxed block">
                      {t(
                        "admin.kids_mode_desc",
                        "Enforce filtered child-friendly library interface.",
                      )}
                    </span>
                  </div>
                  <input
                    type="checkbox"
                    className="toggle toggle-primary toggle-sm shrink-0"
                    checked={Boolean(form.is_kids_mode)}
                    onChange={(e) =>
                      setForm({ ...form, is_kids_mode: e.target.checked })
                    }
                  />
                </label>
              </div>

              <div className="modal-action border-t border-base-200 pt-4 mt-6">
                <button
                  type="button"
                  onClick={closeModal}
                  className="btn btn-ghost"
                >
                  {t("common.cancel", "Cancel")}
                </button>
                <button
                  type="submit"
                  disabled={saving || uploadingAvatar}
                  className="btn btn-primary"
                >
                  {saving ? (
                    <span className="loading loading-spinner"></span>
                  ) : (
                    t("common.save", "Save Changes")
                  )}
                </button>
              </div>
            </form>
          </div>
          <form method="dialog" className="modal-backdrop">
            <button onClick={closeModal}>close</button>
          </form>
        </dialog>
      )}

      {modal === "password" && selected && (
        <dialog className="modal modal-open">
          <div className="modal-box max-w-md max-h-[80dvh] sm:max-h-[85vh] p-0 overflow-hidden flex flex-col">
            <div className="px-4 sm:px-6 py-3.5 sm:py-4 border-b border-base-200 flex items-center justify-between gap-3 shrink-0 bg-base-100">
              <div>
                <h3 className="font-bold text-lg leading-tight">
                  {t("admin.reset_password_title", "Reset Password")}
                </h3>
                <p className="text-xs text-base-content/60 mt-0.5">
                  {t("admin.reset_password_desc", "Set a new password for user:")}{" "}
                  <span className="font-bold text-base-content">
                    {selected.email}
                  </span>
                </p>
              </div>
              <button
                type="button"
                onClick={() => {
                  setModal(null);
                  setNewPassword("");
                  setConfirmPassword("");
                }}
                className="btn btn-ghost btn-circle btn-sm -mr-1.5 text-base-content/70 hover:text-base-content shrink-0"
                aria-label={t("common.close", "Close")}
              >
                <X className="w-4 h-4 sm:w-5 sm:h-5 stroke-[2.5]" />
              </button>
            </div>
            <form onSubmit={handlePassword} className="p-4 sm:p-6 overflow-y-auto flex-1 min-h-0 space-y-4">
              {error && (
                <div className="alert alert-error mb-4 py-2 text-sm rounded-lg flex items-center gap-2">
                  <AlertCircle className="w-4 h-4 shrink-0" />
                  <span>{error}</span>
                </div>
              )}
              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">
                    {t("auth.new_password", "New Password")}
                  </span>
                </label>
                <input
                  type="password"
                  required
                  minLength={8}
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  placeholder={t("auth.password_min", "Minimum 8 characters")}
                  className="input input-bordered w-full focus:input-primary"
                  autoComplete="new-password"
                />
                <PasswordStrength password={newPassword} />
              </div>

              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">
                    {t("settings.confirm_password", "Confirm new password")}
                  </span>
                </label>
                <input
                  type="password"
                  required
                  minLength={8}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  placeholder={t("auth.password_min", "Minimum 8 characters")}
                  className="input input-bordered w-full focus:input-primary"
                  autoComplete="new-password"
                />
              </div>

              <div className="modal-action border-t border-base-200 pt-4 mt-6">
                <button
                  type="button"
                  onClick={() => {
                    setModal(null);
                    setNewPassword("");
                    setConfirmPassword("");
                  }}
                  className="btn btn-ghost"
                >
                  {t("common.cancel", "Cancel")}
                </button>
                <button
                  type="submit"
                  disabled={saving || !newPassword || !confirmPassword}
                  className="btn btn-primary"
                >
                  {saving ? (
                    <span className="loading loading-spinner"></span>
                  ) : (
                    t("admin.update_password", "Update Password")
                  )}
                </button>
              </div>
            </form>
          </div>
          <form method="dialog" className="modal-backdrop">
            <button
              onClick={() => {
                setModal(null);
                setNewPassword("");
                setConfirmPassword("");
              }}
            >
              close
            </button>
          </form>
        </dialog>
      )}

      {modal === "roles" && selected && (
        <dialog className="modal modal-open">
          <div className="modal-box max-w-lg w-11/12 sm:w-full p-0 h-[72dvh] sm:h-auto sm:min-h-125 max-h-[82dvh] sm:max-h-[85vh] flex flex-col overflow-hidden">
            <div className="px-4 sm:px-6 py-3.5 sm:py-4 border-b border-base-200 flex items-start justify-between gap-3 shrink-0 bg-base-100">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2 flex-wrap">
                  <h3 className="font-bold text-lg flex items-center gap-2 leading-tight">
                    <Shield className="w-5 h-5 text-primary shrink-0" />
                    {t("admin.manage_user_roles", "Manage User Roles")}
                  </h3>
                  {selected.is_owner && (
                    <span className="badge badge-warning text-xs font-bold shrink-0">
                      {t("admin.role_owner", "Owner")}
                    </span>
                  )}
                </div>
                <p className="text-xs text-base-content/70 mt-0.5 truncate">
                  {t(
                    "admin.manage_user_roles_desc",
                    "Select active security roles for:",
                  )}{" "}
                  <span className="font-semibold text-base-content">
                    {selected.email}
                  </span>
                </p>
              </div>
              <button
                type="button"
                onClick={() => setModal(null)}
                className="btn btn-ghost btn-circle btn-sm -mr-1.5 text-base-content/70 hover:text-base-content shrink-0"
                aria-label={t("common.close", "Close")}
              >
                <X className="w-4 h-4 sm:w-5 sm:h-5 stroke-[2.5]" />
              </button>
            </div>

            <div className="p-4 sm:p-6 flex flex-col flex-1 min-h-0">
              {selected.is_owner && !currentUser?.is_owner && (
                <div className="alert alert-warning py-2 px-3 text-xs rounded-xl mb-3 flex items-center gap-2 shrink-0">
                  <AlertCircle className="w-4 h-4 shrink-0" />
                  <span>
                    {t(
                      "admin.owner_role_protected",
                      "Owner account roles cannot be modified.",
                    )}
                  </span>
                </div>
              )}

              {error && (
                <div className="alert alert-error mb-3 py-2 px-3 text-xs rounded-xl flex items-center gap-2 shrink-0">
                  <AlertCircle className="w-4 h-4 shrink-0" />
                  <span>{error}</span>
                </div>
              )}

              <form onSubmit={handleRoles} className="flex flex-col flex-1 min-h-0">
              <div className="space-y-2.5 flex-1 min-h-0 overflow-y-auto pr-1">
                {roles.map((role) => {
                  const isChecked = roleIDs.includes(role.id);
                  const isOwnerTarget = Boolean(selected.is_owner);
                  const isSelf = selected.id === currentUser?.id;
                  const isRoleAdmin = role.is_admin || role.name === "ADMIN";
                  const isDisabled =
                    (isOwnerTarget && !currentUser?.is_owner) ||
                    (isSelf && isRoleAdmin && isChecked) ||
                    (!currentUser?.is_owner && isRoleAdmin);

                  let roleBadgeClass = "badge-ghost";
                  if (role.is_admin || role.name === "ADMIN") {
                    roleBadgeClass = "badge-error text-white font-semibold";
                  } else if (role.is_banned || role.name === "BANNED") {
                    roleBadgeClass = "badge-warning text-black font-semibold";
                  } else if (role.name === "MOD") {
                    roleBadgeClass = "badge-info text-white font-semibold";
                  } else if (role.name === "USER") {
                    roleBadgeClass = "badge-ghost font-medium";
                  }

                  return (
                    <label
                      key={role.id}
                      className={`flex items-center justify-between p-3.5 rounded-xl border transition-all cursor-pointer select-none ${
                        isChecked
                          ? "border-primary/50 bg-primary/5 shadow-sm"
                          : "border-base-300 dark:border-base-700 hover:border-base-content/20 bg-base-100"
                      } ${isDisabled ? "opacity-60 cursor-not-allowed bg-base-200/40" : ""}`}
                    >
                      <div className="flex items-start gap-3 min-w-0 pr-2">
                        <input
                          type="checkbox"
                          checked={isChecked}
                          disabled={isDisabled}
                          onChange={(e) => {
                            if (isDisabled) return;
                            const checked = e.target.checked;
                            setRoleIDs((prev) =>
                              checked
                                ? [...prev, role.id]
                                : prev.filter((id) => id !== role.id),
                            );
                          }}
                          className="checkbox checkbox-primary checkbox-sm mt-0.5"
                        />
                        <div className="min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="font-semibold text-sm leading-tight text-base-content">
                              {role.name}
                            </span>
                            <span className={`badge badge-sm ${roleBadgeClass}`}>
                              {role.name}
                            </span>
                            {role.is_system && (
                              <span className="badge badge-xs badge-outline opacity-70">
                                {t("admin.system_role", "System")}
                              </span>
                            )}
                            {role.auto_assign && (
                              <span className="badge badge-xs badge-outline badge-primary opacity-80">
                                {t("admin.auto_assign", "Auto-assign")}
                              </span>
                            )}
                          </div>
                          {role.description && (
                            <p className="text-xs text-base-content/60 mt-1 line-clamp-2">
                              {role.description}
                            </p>
                          )}
                          {isDisabled && (
                            <p className="text-[11px] text-warning mt-1 flex items-center gap-1">
                              <Lock className="w-3 h-3 shrink-0" />
                              {isOwnerTarget
                                ? t("admin.role_protected_owner", "Protected: Owner account")
                                : isSelf && isRoleAdmin
                                  ? t("admin.cannot_remove_own_admin", "Cannot remove your own Admin role")
                                  : t("admin.only_owner_assign_admin", "Only the owner can assign this role")}
                            </p>
                          )}
                        </div>
                      </div>
                    </label>
                  );
                })}
              </div>

              <div className="modal-action shrink-0 pt-3 border-t border-base-200 mt-3 flex items-center justify-end gap-2">
                <button
                  type="button"
                  onClick={() => setModal(null)}
                  className="btn btn-ghost"
                >
                  {t("common.cancel", "Cancel")}
                </button>
                <button
                  type="submit"
                  disabled={saving || (selected.is_owner && !currentUser?.is_owner)}
                  className="btn btn-primary"
                >
                  {saving ? (
                    <span className="loading loading-spinner"></span>
                  ) : (
                    t("common.save", "Save Roles")
                  )}
                </button>
              </div>
            </form>
          </div>
        </div>
        <form method="dialog" className="modal-backdrop">
            <button onClick={() => setModal(null)}>close</button>
          </form>
        </dialog>
      )}

      {/* Send Email Modal */}
      <SendEmailModal
        open={modal === "email" && Boolean(selected)}
        user={selected}
        onClose={() => setModal(null)}
        onSend={(data) => {
          if (!selected) return;
          setError("");
          sendEmailMutation.mutate(
            { id: selected.id, data },
            {
              onSuccess: () => {
                toast.success(t("admin.email_sent", "Email sent"));
                setModal(null);
              },
              onError: (err) =>
                setError(err instanceof Error ? err.message : String(err)),
            },
          );
        }}
        sending={sendEmailMutation.isPending}
        error={error}
        initialSubject={emailForm.subject}
        initialBody={emailForm.body}
      />

      {/* Delete User Confirmation Modal */}
      {userToDelete && (
        <ConfirmModal
          open={Boolean(userToDelete)}
          title={t("admin.delete_user_confirm", "Delete User Account?")}
          message={
            <div className="space-y-2">
              <p>
                {t(
                  "admin.delete_user_desc",
                  "This user account will be soft-deleted. They will immediately lose access to NovelHub.",
                )}
              </p>
              <div className="p-3 rounded-xl bg-error/10 border border-error/20 text-error flex flex-col gap-0.5 min-w-0">
                <span className="font-semibold wrap-break-word">{userToDelete.full_name || userToDelete.email}</span>
                {userToDelete.full_name && userToDelete.email && (
                  <span className="text-xs opacity-70 font-normal break-all">{userToDelete.email}</span>
                )}
              </div>
            </div>
          }
          variant="danger"
          loading={saving}
          confirmText={t("common.delete", "Delete")}
          cancelText={t("common.cancel", "Cancel")}
          onConfirm={confirmDeleteUser}
          onClose={() => setUserToDelete(null)}
        />
      )}

      {/* Restore User Confirmation Modal */}
      {userToRestore && (
        <ConfirmModal
          open={Boolean(userToRestore)}
          title={t("admin.restore_user_title", "Restore User Account?")}
          message={
            <div className="space-y-2">
              <p>
                {t(
                  "admin.restore_user_confirm_msg",
                  "Are you sure you want to restore access for user:",
                )}
              </p>
              <div className="p-3 rounded-xl bg-base-200/60 text-base-content flex flex-col gap-0.5 min-w-0">
                <span className="font-semibold wrap-break-word">{userToRestore.full_name || userToRestore.email}</span>
                {userToRestore.full_name && userToRestore.email && (
                  <span className="text-xs opacity-60 font-normal break-all">{userToRestore.email}</span>
                )}
              </div>
              <p className="text-xs opacity-70">
                {t(
                  "admin.restore_user_note",
                  "This account will be reactivated and will be able to log in to NovelHub again.",
                )}
              </p>
            </div>
          }
          variant="success"
          confirmText={t("admin.restore_action", "Restore User")}
          cancelText={t("common.cancel", "Cancel")}
          onConfirm={confirmRestoreUser}
          onClose={() => setUserToRestore(null)}
        />
      )}

      {/* Revoke User Sessions Confirmation Modal */}
      {userToRevoke && (
        <ConfirmModal
          open={Boolean(userToRevoke)}
          title={t("admin.revoke_sessions_confirm_title", "Revoke User Sessions?")}
          message={
            <div className="space-y-2">
              <p>
                {t(
                  "admin.confirm_revoke_sessions",
                  "Are you sure you want to force this user to log out from all devices?",
                )}
              </p>
              <div className="p-3 rounded-xl bg-warning/10 border border-warning/20 text-base-content flex flex-col gap-0.5 min-w-0">
                <span className="font-semibold wrap-break-word">{userToRevoke.full_name || userToRevoke.email}</span>
                {userToRevoke.full_name && userToRevoke.email && (
                  <span className="text-xs opacity-70 font-normal break-all">{userToRevoke.email}</span>
                )}
              </div>
            </div>
          }
          variant="warning"
          loading={revokeUserSessionsMutation.isPending}
          confirmText={t("admin.revoke_sessions", "Revoke Sessions")}
          cancelText={t("common.cancel", "Cancel")}
          onConfirm={confirmRevokeSessions}
          onClose={() => setUserToRevoke(null)}
        />
      )}

      {selectedImage && (
        <ImageCropperModal
          imageSrc={selectedImage}
          onCrop={handleCropApply}
          onCancel={() => setSelectedImage(null)}
          cropSize={200}
        />
      )}

      {bulkModal === "delete" && (
        <BulkDeleteUsersModal
          isOpen={bulkModal === "delete"}
          selectedUsers={selectedUsers}
          currentUserId={currentUser?.id}
          isCallerOwner={Boolean(currentUser?.is_owner)}
          onClose={() => setBulkModal(null)}
          onSuccess={() => {
            clearSelection();
            void refetchUsers();
          }}
        />
      )}

      {bulkModal === "restore" && (
        <BulkRestoreUsersModal
          isOpen={bulkModal === "restore"}
          selectedUsers={selectedUsers}
          isCallerOwner={Boolean(currentUser?.is_owner)}
          onClose={() => setBulkModal(null)}
          onSuccess={() => {
            clearSelection();
            void refetchUsers();
          }}
        />
      )}

      {bulkModal === "roles" && (
        <BulkChangeRolesModal
          isOpen={bulkModal === "roles"}
          selectedUsers={selectedUsers}
          roles={roles}
          currentUserId={currentUser?.id}
          isCallerOwner={Boolean(currentUser?.is_owner)}
          onClose={() => setBulkModal(null)}
          onSuccess={() => {
            clearSelection();
            void refetchUsers();
          }}
        />
      )}

      {bulkModal === "info" && (
        <BulkEditUsersModal
          isOpen={bulkModal === "info"}
          selectedUsers={selectedUsers}
          currentUserId={currentUser?.id}
          isCallerOwner={Boolean(currentUser?.is_owner)}
          onClose={() => setBulkModal(null)}
          onSuccess={() => {
            clearSelection();
            void refetchUsers();
          }}
        />
      )}

      {bulkModal === "email" && (
        <BulkSendEmailModal
          isOpen={bulkModal === "email"}
          selectedUsers={selectedUsers}
          onClose={() => setBulkModal(null)}
          onSuccess={() => {
            clearSelection();
          }}
        />
      )}
    </div>
  );
}
