import { UserTable } from "@/components/admin";
import { PasswordStrength } from "@/components/common";
import {
  useChangeUserRolesMutation,
  useCreateUserMutation,
  useDeleteUserMutation,
  useResetUserPasswordMutation,
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
  RefreshCw,
  Search,
  UserPlus
} from "lucide-react";
import { SyntheticEvent, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from 'react-toastify';
import { useShallow } from "zustand/react/shallow";

const emptyCreate: CreateUserRequest = {
  email: "",
  password: "",
  full_name: "",
  avatar_url: "",
  role_ids: []
};

export function Users() {
  const { t } = useTranslation();
  const currentUser = useAuthStore((state) => state.user);

  const {
    selectedUser: selected, setSelectedUser: setSelected,
    query, setQuery,
    showDeleted, setShowDeleted,
    error, setError,
    modal, setModal,
    form, setForm,
    newPassword, setNewPassword,
    roleIDs, setRoleIDs,
    userToDelete, setUserToDelete,
  } = useUserAdminStore(useShallow((state) => ({
    selectedUser: state.selectedUser, setSelectedUser: state.setSelectedUser,
    query: state.query, setQuery: state.setQuery,
    showDeleted: state.showDeleted, setShowDeleted: state.setShowDeleted,
    error: state.error, setError: state.setError,
    modal: state.modal, setModal: state.setModal,
    form: state.form, setForm: state.setForm,
    newPassword: state.newPassword, setNewPassword: state.setNewPassword,
    roleIDs: state.roleIDs, setRoleIDs: state.setRoleIDs,
    userToDelete: state.userToDelete, setUserToDelete: state.setUserToDelete,
  })));

  const [cursor, setCursor] = useState("");
  const [cursorHistory, setCursorHistory] = useState<string[]>([]);
  const resetPaging = () => {
    setCursor("");
    setCursorHistory([]);
  };
  const { data: usersData, isLoading: usersLoading, refetch: refetchUsers } = useUsersQuery({
    cursor: cursor || undefined,
    limit: 50,
    search: query || undefined,
    is_deleted: showDeleted ? undefined : false,
    sort: "created_at",
    order: "desc"
  });

  const { data: roles = [], isLoading: rolesLoading, refetch: refetchRoles } = useRolesQuery();

  const createUserMutation = useCreateUserMutation();
  const updateUserMutation = useUpdateUserMutation();
  const resetPasswordMutation = useResetUserPasswordMutation();
  const changeRolesMutation = useChangeUserRolesMutation();
  const deleteUserMutation = useDeleteUserMutation();
  const sendEmailMutation = useSendUserEmailMutation();

  const users = usersData?.users || [];
  const loading = usersLoading || rolesLoading;
  const saving =
    createUserMutation.isPending ||
    updateUserMutation.isPending ||
    resetPasswordMutation.isPending ||
    changeRolesMutation.isPending ||
    deleteUserMutation.isPending ||
    sendEmailMutation.isPending;

  const activeUsers = useMemo(() => users.filter((item) => !item.is_deleted).length, [users]);
  const [emailForm, setEmailForm] = useState({ subject: "", body: "" });

  function openCreate() {
    setForm({ ...emptyCreate, role_ids: roles.filter((role) => role.auto_assign).map((role) => role.id) });
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
      role_ids: target.roles.map((role) => role.id)
    });
    setModal("edit");
    setError("");
  }

  function openPassword(target: User) {
    setSelected(target);
    setNewPassword("");
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
          toast.success(t('admin.email_sent', 'Email sent'));
          setModal(null);
        },
        onError: (err) => setError(err instanceof Error ? err.message : String(err)),
      }
    );
  }

  function handleCreate(event: SyntheticEvent) {
    event.preventDefault();
    setError("");
    createUserMutation.mutate(form, {
      onSuccess: () => {
        toast.success(t('common.success', 'Success'));
        setModal(null);
      },
      onError: (err) => setError(err instanceof Error ? err.message : String(err)),
    });
  }

  function handleEdit(event: SyntheticEvent) {
    event.preventDefault();
    if (!selected) return;
    setError("");
    updateUserMutation.mutate(
      {
        id: selected.id,
        data: { full_name: form.full_name, avatar_url: form.avatar_url || undefined },
      },
      {
        onSuccess: () => {
          toast.success(t('common.success', 'Success'));
          setModal(null);
        },
        onError: (err) => setError(err instanceof Error ? err.message : String(err)),
      }
    );
  }

  function handlePassword(event: SyntheticEvent) {
    event.preventDefault();
    if (!selected) return;
    setError("");
    resetPasswordMutation.mutate(
      { id: selected.id, password: newPassword },
      {
        onSuccess: () => {
          toast.success(t('common.success', 'Success'));
          setModal(null);
        },
        onError: (err) => setError(err instanceof Error ? err.message : String(err)),
      }
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
          toast.success(t('common.success', 'Success'));
          setModal(null);
        },
        onError: (err) => setError(err instanceof Error ? err.message : String(err)),
      }
    );
  }

  function confirmDeleteUser() {
    if (!userToDelete) return;
    setError("");
    deleteUserMutation.mutate(userToDelete.id, {
      onSuccess: () => {
        toast.success(t('common.success', 'Success'));
        setUserToDelete(null);
      },
      onError: (err) => setError(err instanceof Error ? err.message : String(err)),
    });
  }

  async function handleRestore(target: User) {
    setError("");
    try {
      await adminService.restoreUser(target.id);
      toast.success(t('common.success', 'Success'));
      void refetchUsers();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <div className="flex flex-col h-full bg-base-100">
      {/* Header */}
      <header className="px-4 py-5 sm:px-6 lg:px-8 lg:py-6 border-b border-base-200 flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-base-100/50 backdrop-blur-xl sticky top-0 z-10">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t('admin.user_management', 'User Management')}</h1>
          <p className="text-sm text-base-content/60 mt-1">
            {t('admin.user_subtitle', 'Manage accounts, roles, access levels, and security credentials.')}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => {
              void refetchUsers();
              void refetchRoles();
            }}
            className="btn btn-square btn-ghost btn-sm sm:btn-md"
            title="Refresh"
          >
            <RefreshCw className={`h-5 w-5 ${loading ? "animate-spin" : ""}`} />
          </button>
          <button onClick={openCreate} className="btn btn-primary btn-sm sm:btn-md gap-2">
            <UserPlus className="w-4 h-4" />
            {t('admin.add_user', 'Add User')}
          </button>
        </div>
      </header>

      <div className="flex-1 overflow-auto p-4 sm:p-6 lg:p-8 space-y-6 max-w-7xl mx-auto w-full">
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 bg-base-200/40 border border-base-200 p-2.5 rounded-2xl">
          <div className="flex-1 flex flex-col sm:flex-row items-stretch sm:items-center gap-2.5">
            <div className="relative flex-1">
              <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-base-content/40 pointer-events-none" />
              <input
                type="text"
                value={query}
                onChange={(e) => { setQuery(e.target.value); resetPaging(); }}
                placeholder={t('admin.search_users_placeholder', 'Search users by name or email...')}
                className="input input-bordered w-full pl-10 focus:input-primary h-10 text-sm rounded-xl bg-base-100"
              />
            </div>
            <label className="cursor-pointer flex items-center gap-2 px-3.5 h-10 rounded-xl border border-base-200 bg-base-100 hover:bg-base-200/50 transition-colors shrink-0">
              <input
                type="checkbox"
                checked={showDeleted}
                onChange={(e) => { setShowDeleted(e.target.checked); resetPaging(); }}
                className="checkbox checkbox-primary checkbox-xs rounded"
              />
              <span className="text-xs font-medium select-none text-base-content/80">{t('admin.show_deleted', 'Show Deleted')}</span>
            </label>
          </div>

          {/* Right: Stats Counters */}
          <div className="flex items-center gap-2 shrink-0 sm:border-l sm:border-base-200/80 sm:pl-3">
            <div className="flex items-center gap-2 px-3 h-10 rounded-xl bg-base-100 border border-base-200 text-xs">
              <span className="text-base-content/60 font-medium">{t('admin.total_loaded', 'Loaded Users')}:</span>
              <span className="font-bold text-primary text-sm">{users.length}</span>
            </div>
            <div className="flex items-center gap-2 px-3 h-10 rounded-xl bg-base-100 border border-base-200 text-xs">
              <span className="text-base-content/60 font-medium">{t('admin.active_users', 'Active')}:</span>
              <span className="font-bold text-success text-sm">{activeUsers}</span>
            </div>
          </div>
        </div>

        {/* User Table */}
        <UserTable
          users={users}
          t={t}
          onEdit={openEdit}
          onPassword={openPassword}
          onRoles={openRoles}
          onEmail={openEmail}
          onDelete={setUserToDelete}
          onRestore={handleRestore}
          currentUserId={currentUser?.id}
          isCallerOwner={Boolean(currentUser?.is_owner)}
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
          <div className="modal-box max-w-md">
            <h3 className="font-bold text-lg mb-4">{t('admin.create_user_title', 'Create New User')}</h3>
            {error && (
              <div className="alert alert-error mb-4 py-2 text-sm rounded-lg flex items-center gap-2">
                <AlertCircle className="w-4 h-4 shrink-0" />
                <span>{error}</span>
              </div>
            )}
            <form onSubmit={handleCreate} className="space-y-4">
              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">{t('auth.email', 'Email Address')}</span>
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
                  <span className="label-text font-semibold">{t('user.full_name', 'Full Name')}</span>
                </label>
                <input
                  type="text"
                  required
                  value={form.full_name}
                  onChange={(e) => setForm({ ...form, full_name: e.target.value })}
                  placeholder="John Doe"
                  className="input input-bordered w-full focus:input-primary"
                />
              </div>

              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">{t('auth.password', 'Initial Password')}</span>
                </label>
                <input
                  type="password"
                  required
                  value={form.password}
                  onChange={(e) => setForm({ ...form, password: e.target.value })}
                  placeholder="••••••••"
                  className="input input-bordered w-full focus:input-primary"
                />
                <PasswordStrength password={form.password} />
              </div>

              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">{t('admin.assign_roles', 'Assign Roles')}</span>
                </label>
                <div className="space-y-2 bg-base-200/50 p-3 rounded-xl border border-base-200">
                  {roles.map((role) => (
                    <label key={role.id} className="label cursor-pointer justify-start gap-3 py-1">
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
                              : currentIds.filter((id) => id !== role.id)
                          }));
                        }}
                        className="checkbox checkbox-primary checkbox-sm"
                      />
                      <div>
                        <span className="font-semibold text-sm">{role.name}</span>
                        {role.description && (
                          <p className="text-xs text-base-content/60">{role.description}</p>
                        )}
                      </div>
                    </label>
                  ))}
                </div>
              </div>

              <div className="modal-action">
                <button type="button" onClick={() => setModal(null)} className="btn btn-ghost">
                  {t('common.cancel', 'Cancel')}
                </button>
                <button type="submit" disabled={saving} className="btn btn-primary">
                  {saving ? <span className="loading loading-spinner"></span> : t('common.create', 'Create User')}
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
          <div className="modal-box max-w-md">
            <h3 className="font-bold text-lg mb-4">{t('admin.edit_user_title', 'Edit Profile')}</h3>
            {error && (
              <div className="alert alert-error mb-4 py-2 text-sm rounded-lg flex items-center gap-2">
                <AlertCircle className="w-4 h-4 shrink-0" />
                <span>{error}</span>
              </div>
            )}
            <form onSubmit={handleEdit} className="space-y-4">
              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">{t('auth.email', 'Email Address')}</span>
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
                  <span className="label-text font-semibold">{t('user.full_name', 'Full Name')}</span>
                </label>
                <input
                  type="text"
                  required
                  value={form.full_name}
                  onChange={(e) => setForm({ ...form, full_name: e.target.value })}
                  className="input input-bordered w-full focus:input-primary"
                />
              </div>

              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">{t('user.avatar_url', 'Avatar URL')}</span>
                </label>
                <input
                  type="text"
                  value={form.avatar_url || ""}
                  onChange={(e) => setForm({ ...form, avatar_url: e.target.value })}
                  placeholder="https://example.com/avatar.jpg"
                  className="input input-bordered w-full focus:input-primary"
                />
              </div>

              <div className="modal-action">
                <button type="button" onClick={() => setModal(null)} className="btn btn-ghost">
                  {t('common.cancel', 'Cancel')}
                </button>
                <button type="submit" disabled={saving} className="btn btn-primary">
                  {saving ? <span className="loading loading-spinner"></span> : t('common.save', 'Save Changes')}
                </button>
              </div>
            </form>
          </div>
          <form method="dialog" className="modal-backdrop">
            <button onClick={() => setModal(null)}>close</button>
          </form>
        </dialog>
      )}

      {modal === "password" && selected && (
        <dialog className="modal modal-open">
          <div className="modal-box max-w-md">
            <h3 className="font-bold text-lg mb-2">{t('admin.reset_password_title', 'Reset Password')}</h3>
            <p className="text-xs text-base-content/60 mb-4">
              {t('admin.reset_password_desc', 'Set a new password for user:')} <span className="font-bold text-base-content">{selected.email}</span>
            </p>
            {error && (
              <div className="alert alert-error mb-4 py-2 text-sm rounded-lg flex items-center gap-2">
                <AlertCircle className="w-4 h-4 shrink-0" />
                <span>{error}</span>
              </div>
            )}
            <form onSubmit={handlePassword} className="space-y-4">
              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">{t('auth.new_password', 'New Password')}</span>
                </label>
                <input
                  type="password"
                  required
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  placeholder="••••••••"
                  className="input input-bordered w-full focus:input-primary"
                />
                <PasswordStrength password={newPassword} />
              </div>

              <div className="modal-action">
                <button type="button" onClick={() => setModal(null)} className="btn btn-ghost">
                  {t('common.cancel', 'Cancel')}
                </button>
                <button type="submit" disabled={saving || !newPassword} className="btn btn-primary">
                  {saving ? <span className="loading loading-spinner"></span> : t('admin.update_password', 'Update Password')}
                </button>
              </div>
            </form>
          </div>
          <form method="dialog" className="modal-backdrop">
            <button onClick={() => setModal(null)}>close</button>
          </form>
        </dialog>
      )}

      {modal === "roles" && selected && (
        <dialog className="modal modal-open">
          <div className="modal-box max-w-md">
            <h3 className="font-bold text-lg mb-2">{t('admin.manage_user_roles', 'Manage User Roles')}</h3>
            <p className="text-xs text-base-content/60 mb-4">
              {t('admin.manage_user_roles_desc', 'Select active security roles for:')} <span className="font-bold text-base-content">{selected.email}</span>
            </p>
            {error && (
              <div className="alert alert-error mb-4 py-2 text-sm rounded-lg flex items-center gap-2">
                <AlertCircle className="w-4 h-4 shrink-0" />
                <span>{error}</span>
              </div>
            )}
            <form onSubmit={handleRoles} className="space-y-4">
              <div className="space-y-2 bg-base-200/50 p-3 rounded-xl border border-base-200">
                {roles.map((role) => (
                  <label key={role.id} className="label cursor-pointer justify-start gap-3 py-1">
                    <input
                      type="checkbox"
                      checked={roleIDs.includes(role.id)}
                      onChange={(e) => {
                        const checked = e.target.checked;
                        setRoleIDs((prev) =>
                          checked ? [...prev, role.id] : prev.filter((id) => id !== role.id)
                        );
                      }}
                      className="checkbox checkbox-primary checkbox-sm"
                    />
                    <div>
                      <span className="font-semibold text-sm">{role.name}</span>
                      {role.description && (
                        <p className="text-xs text-base-content/60">{role.description}</p>
                      )}
                    </div>
                  </label>
                ))}
              </div>

              <div className="modal-action">
                <button type="button" onClick={() => setModal(null)} className="btn btn-ghost">
                  {t('common.cancel', 'Cancel')}
                </button>
                <button type="submit" disabled={saving} className="btn btn-primary">
                  {saving ? <span className="loading loading-spinner"></span> : t('common.save', 'Save Roles')}
                </button>
              </div>
            </form>
          </div>
          <form method="dialog" className="modal-backdrop">
            <button onClick={() => setModal(null)}>close</button>
          </form>
        </dialog>
      )}

      {modal === "email" && selected && (
        <dialog className="modal modal-open">
          <div className="modal-box max-w-md">
            <h3 className="font-bold text-lg mb-2">{t('admin.send_email_title', 'Send Email')}</h3>
            <p className="text-xs text-base-content/60 mb-4">
              {t('admin.send_email_desc', 'This message goes to the address on file:')}{" "}
              <span className="font-bold text-base-content">{selected.email}</span>
            </p>
            {error && (
              <div className="alert alert-error mb-4 py-2 text-sm rounded-lg flex items-center gap-2">
                <AlertCircle className="w-4 h-4 shrink-0" />
                <span>{error}</span>
              </div>
            )}
            <form onSubmit={handleSendEmail} className="space-y-4">
              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">{t('admin.email_subject', 'Subject')}</span>
                </label>
                <input
                  type="text"
                  required
                  maxLength={200}
                  value={emailForm.subject}
                  onChange={(e) => setEmailForm({ ...emailForm, subject: e.target.value })}
                  className="input input-bordered w-full focus:input-primary"
                />
              </div>

              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">{t('admin.email_body', 'Message')}</span>
                </label>
                <textarea
                  required
                  rows={6}
                  maxLength={10000}
                  value={emailForm.body}
                  onChange={(e) => setEmailForm({ ...emailForm, body: e.target.value })}
                  className="textarea textarea-bordered w-full focus:textarea-primary"
                />
              </div>

              <div className="modal-action">
                <button type="button" onClick={() => setModal(null)} className="btn btn-ghost">
                  {t('common.cancel', 'Cancel')}
                </button>
                <button
                  type="submit"
                  disabled={saving || !emailForm.subject.trim() || !emailForm.body.trim()}
                  className="btn btn-primary"
                >
                  {saving ? <span className="loading loading-spinner"></span> : t('admin.send_email', 'Send')}
                </button>
              </div>
            </form>
          </div>
          <form method="dialog" className="modal-backdrop">
            <button onClick={() => setModal(null)}>close</button>
          </form>
        </dialog>
      )}

      {/* Delete User Modal */}
      {userToDelete && (
        <dialog className="modal modal-open">
          <div className="modal-box max-w-sm text-center">
            <div className="w-12 h-12 rounded-full bg-error/10 text-error flex items-center justify-center mx-auto mb-3">
              <AlertCircle className="w-6 h-6" />
            </div>
            <h3 className="font-bold text-lg">{t('admin.delete_user_confirm', 'Delete User Account?')}</h3>
            <p className="text-xs text-base-content/60 mt-1 mb-6">
              {t('admin.delete_user_desc', 'This user account will be soft-deleted. They will immediately lose access to NovelHub.')}
            </p>
            <div className="flex gap-2 justify-end">
              <button onClick={() => setUserToDelete(null)} className="btn btn-ghost flex-1">
                {t('common.cancel', 'Cancel')}
              </button>
              <button onClick={confirmDeleteUser} disabled={saving} className="btn btn-error text-white flex-1">
                {saving ? <span className="loading loading-spinner"></span> : t('common.delete', 'Delete')}
              </button>
            </div>
          </div>
          <form method="dialog" className="modal-backdrop">
            <button onClick={() => setUserToDelete(null)}>close</button>
          </form>
        </dialog>
      )}
    </div>
  );
}
