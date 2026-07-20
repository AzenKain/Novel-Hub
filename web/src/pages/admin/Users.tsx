import { UserTable } from "@/components/admin";
import { adminService } from "@/services";
import { useUserAdminStore } from "@/stores";
import type { CreateUserRequest, User } from "@/types";
import {
  AlertCircle,
  RefreshCw,
  Search,
  UserPlus
} from "lucide-react";
import { FormEvent, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { toast } from 'react-toastify';
import { useShallow } from "zustand/react/shallow";

type ModalMode = "create" | "edit" | "password" | "roles" | null;

const emptyCreate: CreateUserRequest = {
  email: "",
  password: "",
  full_name: "",
  avatar_url: "",
  role_ids: []
};

export function Users() {
  const { t } = useTranslation();
  const {
    users, setUsers,
    roles, setRoles,
    selectedUser: selected, setSelectedUser: setSelected,
    query, setQuery,
    showDeleted, setShowDeleted,
    loading, setLoading,
    saving, setSaving,
    error, setError,
    modal, setModal,
    form, setForm,
    newPassword, setNewPassword,
    roleIDs, setRoleIDs,
    userToDelete, setUserToDelete,
    reset
  } = useUserAdminStore(useShallow((state) => ({
    users: state.users, setUsers: state.setUsers,
    roles: state.roles, setRoles: state.setRoles,
    selectedUser: state.selectedUser, setSelectedUser: state.setSelectedUser,
    query: state.query, setQuery: state.setQuery,
    showDeleted: state.showDeleted, setShowDeleted: state.setShowDeleted,
    loading: state.loading, setLoading: state.setLoading,
    saving: state.saving, setSaving: state.setSaving,
    error: state.error, setError: state.setError,
    modal: state.modal, setModal: state.setModal,
    form: state.form, setForm: state.setForm,
    newPassword: state.newPassword, setNewPassword: state.setNewPassword,
    roleIDs: state.roleIDs, setRoleIDs: state.setRoleIDs,
    userToDelete: state.userToDelete, setUserToDelete: state.setUserToDelete,
    reset: state.reset
  })));

  useEffect(() => {
    return () => {
      reset();
    };
  }, [reset]);

  const activeUsers = useMemo(() => users.filter((item) => !item.is_deleted).length, [users]);

  async function loadData() {
    setLoading(true);
    setError("");
    try {
      const [userRes, roleRes] = await Promise.all([
        adminService.searchUsers({
          page: 1,
          limit: 50,
          search: query || undefined,
          is_deleted: showDeleted ? undefined : false,
          sort: "created_at",
          order: "desc"
        }),
        adminService.getRoles()
      ]);
      const nextUsers = userRes.data || [];
      const nextRoles = roleRes.data || [];
      setUsers(nextUsers);
      setRoles(nextRoles);
      
      setSelected((current) => {
        if (current && nextUsers.some((item) => item.id === current.id)) {
          return nextUsers.find((item) => item.id === current.id) || nextUsers[0] || null;
        }
        return nextUsers[0] || null;
      });

    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadData();
  }, [showDeleted]);

  function openCreate() {
    setForm({ ...emptyCreate, role_ids: roles.filter((role) => role.name === "USER").map((role) => role.id) });
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

  async function handleCreate(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setError("");
    try {
      await adminService.createUser(form);
      toast.success(t('common.success', 'Success'));
      setModal(null);
      await loadData();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  async function handleEdit(event: FormEvent) {
    event.preventDefault();
    if (!selected) return;
    setSaving(true);
    setError("");
    try {
      await adminService.updateUser(selected.id, {
        full_name: form.full_name,
        avatar_url: form.avatar_url || undefined
      });
      toast.success(t('common.success', 'Success'));
      setModal(null);
      await loadData();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  async function handlePassword(event: FormEvent) {
    event.preventDefault();
    if (!selected) return;
    setSaving(true);
    setError("");
    try {
      await adminService.resetPassword(selected.id, newPassword);
      toast.success(t('common.success', 'Success'));
      setModal(null);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  async function handleRoles(event: FormEvent) {
    event.preventDefault();
    if (!selected) return;
    setSaving(true);
    setError("");
    try {
      await adminService.changeRoles(selected.id, roleIDs);
      toast.success(t('common.success', 'Success'));
      setModal(null);
      await loadData();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  async function confirmDeleteUser() {
    if (!userToDelete) return;
    setSaving(true);
    setError("");
    try {
      await adminService.deleteUser(userToDelete.id);
      toast.success(t('common.success', 'Success'));
      setUserToDelete(null);
      await loadData();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  async function handleRestore(target: User) {
    setSaving(true);
    setError("");
    try {
      await adminService.restoreUser(target.id);
      toast.success(t('common.success', 'Success'));
      await loadData();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  function toggleRole(id: number) {
    setRoleIDs((current) => (current.includes(id) ? current.filter((item) => item !== id) : [...current, id]));
    setForm((current) => ({
      ...current,
      role_ids: current.role_ids?.includes(id)
        ? current.role_ids.filter((item) => item !== id)
        : [...(current.role_ids || []), id]
    }));
  }

  return (
    <div className="flex flex-col h-full bg-base-100">
      <header className="px-4 py-5 sm:px-6 lg:px-8 lg:py-6 border-b border-base-200 flex flex-col gap-4 bg-base-100/50 backdrop-blur-xl sticky top-0 z-10 lg:flex-row lg:items-center lg:justify-between">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold tracking-tight">{t('admin.users', 'User Management')}</h1>
          <p className="text-sm text-base-content/60 mt-1">{t('admin.user_desc', 'Manage users, roles, and access controls')}</p>
        </div>
        <div className="flex w-full min-w-0 flex-wrap items-center gap-2 lg:w-auto lg:justify-end">
          <form
            onSubmit={(e) => {
              e.preventDefault();
              void loadData();
            }}
            className="relative min-w-0 flex-1 basis-full sm:basis-64 lg:flex-none"
          >
            <input
              type="text"
              placeholder={t('admin.search', 'Search users...')}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className="input input-bordered input-sm sm:input-md w-full lg:w-64 pl-10"
            />
            <Search className="absolute left-3 top-2.5 h-4 w-4 opacity-50" />
            <button type="submit" className="hidden" />
          </form>
          <button
            onClick={() => void loadData()}
            className="btn btn-square btn-ghost btn-sm sm:btn-md"
            title="Refresh list"
          >
            <RefreshCw className={`h-5 w-5 ${loading ? "animate-spin" : ""}`} />
          </button>
          <button
            onClick={openCreate}
            className="btn btn-primary btn-sm sm:btn-md shrink-0"
          >
            <UserPlus className="h-4 w-4 mr-1 sm:mr-2" />
            <span className="hidden sm:inline">{t('admin.add_user', 'Add User')}</span>
          </button>
        </div>
      </header>

      <div className="flex-1 overflow-auto p-8">
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-6">
            <h2 className="text-lg font-semibold">{t('admin.all_users', 'All Users')}</h2>
            <div className="flex items-center text-sm">
              <span className="badge badge-success badge-xs mr-2"></span>
              <span className="font-medium opacity-70">{activeUsers} {t('admin.active', 'Active')}</span>
            </div>
          </div>
          <label className="label cursor-pointer justify-end gap-2 p-0 hover:opacity-80">
            <span className="label-text">{t('admin.show_deleted', 'Show Deleted')}</span>
            <input
              type="checkbox"
              checked={showDeleted}
              onChange={(e) => setShowDeleted(e.target.checked)}
              className="toggle toggle-sm toggle-primary"
            />
          </label>
        </div>

        <UserTable
          users={users}
          t={t}
          onEdit={openEdit}
          onPassword={openPassword}
          onRoles={openRoles}
          onDelete={setUserToDelete}
          onRestore={(item) => void handleRestore(item)}
        />
      </div>

      {/* Modals using DaisyUI */}
      <dialog className={`modal ${modal !== null ? "modal-open" : ""}`}>
        <div className="modal-box">
          <button 
            onClick={() => setModal(null)} 
            className="btn btn-sm btn-circle btn-ghost absolute right-2 top-2"
          >
            ✕
          </button>
          
          <h3 className="font-bold text-lg border-b border-base-200 pb-4 mb-4">
            {modal === "create" && t('admin.create_user', 'Create New User')}
            {modal === "edit" && t('admin.edit_user', 'Edit User Profile')}
            {modal === "password" && t('admin.reset_password', 'Reset Password')}
            {modal === "roles" && t('admin.manage_roles', 'Manage Roles')}
          </h3>
          
          <form onSubmit={
            modal === "create" ? handleCreate :
            modal === "edit" ? handleEdit :
            modal === "password" ? handlePassword :
            handleRoles
          }>
            
            {(modal === "create" || modal === "edit") && (
              <div className="flex flex-col gap-4">
                {modal === "create" && (
                  <div className="flex flex-col gap-1.5">
                    <label className="text-sm font-medium pl-1">
                      {t('admin.email', 'Email Address')}
                    </label>
                    <input
                      type="email"
                      required
                      value={form.email}
                      onChange={(e) => setForm({ ...form, email: e.target.value })}
                      className="input input-bordered focus:input-primary"
                      placeholder="user@example.com"
                    />
                  </div>
                )}
                {modal === "create" && (
                  <div className="flex flex-col gap-1.5">
                  <label className="text-sm font-medium pl-1">
                    {t('admin.password', 'Password')}
                  </label>
                    <input
                      type="password"
                      required
                      value={form.password}
                      onChange={(e) => setForm({ ...form, password: e.target.value })}
                      className="input input-bordered focus:input-primary"
                    />
                  </div>
                )}
                  <div className="flex flex-col gap-1.5">
                    <label className="text-sm font-medium pl-1">
                      {t('admin.full_name', 'Full Name')}
                    </label>
                  <input
                    type="text"
                    required
                    value={form.full_name}
                    onChange={(e) => setForm({ ...form, full_name: e.target.value })}
                    className="input input-bordered focus:input-primary"
                    placeholder="John Doe"
                  />
                </div>
                {(modal === "create") && (
                  <div className="flex flex-col gap-1.5">
                    <label className="text-sm font-medium pl-1">
                      {t('admin.initial_roles', 'Initial Roles')}
                    </label>
                    <div className="p-3 bg-base-200/50 rounded-xl border border-base-200 max-h-40 overflow-y-auto flex flex-col gap-2">
                      {roles.map((role) => (
                        <label key={role.id} className="cursor-pointer label p-1 hover:bg-base-100 rounded-lg justify-start gap-3">
                          <input
                            type="checkbox"
                            checked={form.role_ids?.includes(role.id) || false}
                            onChange={() => toggleRole(role.id)}
                            className="checkbox checkbox-sm checkbox-primary"
                          />
                          <span className="label-text font-medium">{role.name}</span>
                        </label>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}

            {modal === "password" && (
              <div className="flex flex-col gap-1.5">
                <label className="text-sm font-medium pl-1">
                  {t('admin.new_password', 'New Password')}
                </label>
                <input
                  type="password"
                  required
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  className="input input-bordered focus:input-primary"
                />
              </div>
            )}

            {modal === "roles" && (
              <div className="flex flex-col gap-1.5">
                <label className="text-sm font-medium pl-1">
                  {t('admin.account_status', 'Account Status')}
                </label>
                <div className="p-3 bg-base-200/50 rounded-xl border border-base-200 flex flex-col gap-2">
                  {roles.map((role) => (
                    <label key={role.id} className="cursor-pointer label p-1 hover:bg-base-100 rounded-lg justify-start gap-3">
                      <input
                        type="checkbox"
                        checked={roleIDs.includes(role.id)}
                        onChange={() => toggleRole(role.id)}
                        className="checkbox checkbox-sm checkbox-primary"
                      />
                      <span className="label-text font-medium">{role.name}</span>
                    </label>
                  ))}
                </div>
              </div>
            )}

            <div className="modal-action mt-6">
              <button
                type="button"
                onClick={() => setModal(null)}
                disabled={saving}
                className="btn btn-ghost"
              >
                {t('admin.cancel', 'Cancel')}
              </button>
              <button
                type="submit"
                disabled={saving}
                className="btn btn-primary"
              >
                {saving && <span className="loading loading-spinner"></span>}
                {t('admin.save_changes', 'Save Changes')}
              </button>
            </div>
          </form>
        </div>
        <form method="dialog" className="modal-backdrop">
          <button onClick={() => setModal(null)}>close</button>
        </form>
      </dialog>

      {/* Delete User Confirmation Modal */}
      <dialog className={`modal ${userToDelete ? "modal-open" : ""}`}>
        <div className="modal-box">
          <h3 className="font-bold text-lg text-error flex items-center gap-2">
            <AlertCircle className="w-6 h-6" />
            Delete User
          </h3>
          <p className="py-4 text-sm opacity-80">
            Are you sure you want to delete user <strong>{userToDelete?.email}</strong>? This action is permanent and cannot be undone.
          </p>
          <div className="modal-action">
            <button onClick={() => setUserToDelete(null)} className="btn btn-ghost">Cancel</button>
            <button 
              onClick={() => void confirmDeleteUser()} 
              className="btn btn-error"
              disabled={saving}
            >
              {saving ? <span className="loading loading-spinner loading-xs"></span> : "Delete"}
            </button>
          </div>
        </div>
        <form method="dialog" className="modal-backdrop">
          <button onClick={() => setUserToDelete(null)}>close</button>
        </form>
      </dialog>
    </div>
  );
}
