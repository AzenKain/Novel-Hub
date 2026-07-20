import { usePermissionsQuery, useRolesQuery } from "@/hooks";
import { adminService } from "@/services";
import type { CreateRoleRequest } from "@/types";
import { useQueryClient } from "@tanstack/react-query";
import {
  AlertCircle,
  Loader2,
  Pencil,
  Plus,
  RefreshCw,
  Save,
  Shield,
  Trash2,
  X,
} from "lucide-react";
import { FormEvent, useEffect } from "react";
import { toast } from "react-toastify";
import { useShallow } from "zustand/react/shallow";

import { useRoleAdminStore } from "@/stores";

interface PermissionAssignment {
  permission_key: string;
  effect: "allow" | "deny";
  conditions: Record<string, unknown>;
}

export function Roles() {
  const queryClient = useQueryClient();
  const { data: rolesData, isLoading: rolesLoading, refetch: refetchRoles } = useRolesQuery();
  const { data: permissionsData, isLoading: permissionsLoading, refetch: refetchPermissions } = usePermissionsQuery();

  const {
    roles, setRoles,
    permissions, setPermissions,
    selectedRole, setSelectedRole,
    loading, setLoading,
    saving, setSaving,
    error, setError,
    assignments, setAssignments,
    showModal, setShowModal,
    modalMode, setModalMode,
    form, setForm,
    roleToDelete, setRoleToDelete,
    libraryIdsInput, setLibraryIdsInput,
    reset
  } = useRoleAdminStore(useShallow((state) => ({
    roles: state.roles, setRoles: state.setRoles,
    permissions: state.permissions, setPermissions: state.setPermissions,
    selectedRole: state.selectedRole, setSelectedRole: state.setSelectedRole,
    loading: state.loading, setLoading: state.setLoading,
    saving: state.saving, setSaving: state.setSaving,
    error: state.error, setError: state.setError,
    assignments: state.assignments, setAssignments: state.setAssignments,
    showModal: state.showModal, setShowModal: state.setShowModal,
    modalMode: state.modalMode, setModalMode: state.setModalMode,
    form: state.form, setForm: state.setForm,
    roleToDelete: state.roleToDelete, setRoleToDelete: state.setRoleToDelete,
    libraryIdsInput: state.libraryIdsInput, setLibraryIdsInput: state.setLibraryIdsInput,
    reset: state.reset
  })));

  useEffect(() => {
    if (rolesData) {
      setRoles(rolesData);
      setSelectedRole((current) => {
        if (current && rolesData.some((r) => r.id === current.id)) {
          return rolesData.find((r) => r.id === current.id) || rolesData[0] || null;
        }
        return rolesData[0] || null;
      });
    }
  }, [rolesData, setRoles, setSelectedRole]);

  useEffect(() => {
    if (permissionsData) {
      setPermissions(permissionsData);
    }
  }, [permissionsData, setPermissions]);

  useEffect(() => {
    setLoading(rolesLoading || permissionsLoading);
  }, [rolesLoading, permissionsLoading, setLoading]);

  useEffect(() => {
    return () => {
      reset();
    };
  }, [reset]);



  // Sync assignments when selected role changes
  useEffect(() => {
    if (selectedRole?.permissions) {
      setAssignments(
        selectedRole.permissions.map((rp) => ({
          permission_key: rp.permission_key,
          effect: rp.effect || "allow",
          conditions: rp.conditions || {},
        }))
      );
    } else {
      setAssignments([]);
    }
    setLibraryIdsInput({});
  }, [selectedRole?.id]);

  function isAssigned(key: string): boolean {
    return assignments.some((a) => a.permission_key === key);
  }

  function getAssignment(key: string): PermissionAssignment | undefined {
    return assignments.find((a) => a.permission_key === key);
  }

  function togglePermission(key: string) {
    setAssignments((prev) => {
      if (prev.some((a) => a.permission_key === key)) {
        return prev.filter((a) => a.permission_key !== key);
      }
      return [...prev, { permission_key: key, effect: "allow", conditions: {} }];
    });
  }

  function setEffect(key: string, effect: "allow" | "deny") {
    setAssignments((prev) =>
      prev.map((a) => (a.permission_key === key ? { ...a, effect } : a))
    );
  }

  function setLibraryIds(key: string, input: string) {
    setLibraryIdsInput((prev) => ({ ...prev, [key]: input }));
  }

  function applyLibraryIds(key: string) {
    const input = libraryIdsInput[key] || "";
    const ids = input
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean);
    setAssignments((prev) =>
      prev.map((a) =>
        a.permission_key === key
          ? { ...a, conditions: ids.length > 0 ? { library_ids: ids } : {} }
          : a
      )
    );
  }

  async function savePermissions() {
    if (!selectedRole) return;
    if (selectedRole.is_admin) {
      toast.warning("Cannot modify admin role permissions.");
      return;
    }
    setSaving(true);
    try {
      await adminService.updateRolePermissions(selectedRole.id, {
        permissions: assignments.map((a) => ({
          permission_key: a.permission_key,
          effect: a.effect,
          conditions: a.conditions,
        })),
      });
      toast.success("Permissions updated");
      await queryClient.invalidateQueries({ queryKey: ["admin", "roles"] });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  function openCreate() {
    setForm({ name: "", description: "", auto_assign: false });
    setModalMode("create");
    setShowModal(true);
  }

  function openEdit() {
    if (!selectedRole) return;
    if (selectedRole.is_admin) {
      toast.warning("Cannot edit admin role.");
      return;
    }
    setForm({
      name: selectedRole.name,
      description: selectedRole.description || "",
      auto_assign: selectedRole.auto_assign || false,
    });
    setModalMode("edit");
    setShowModal(true);
  }

  async function handleSave(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      if (modalMode === "create") {
        const data: CreateRoleRequest = {
          name: form.name,
          description: form.description,
          auto_assign: form.auto_assign,
        };
        await adminService.createRole(data);
        toast.success("Role created");
      } else if (selectedRole) {
        await adminService.updateRole(selectedRole.id, {
          name: form.name,
          description: form.description,
          auto_assign: form.auto_assign,
        });
        toast.success("Role updated");
      }
      setShowModal(false);
      await queryClient.invalidateQueries({ queryKey: ["admin", "roles"] });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  async function confirmDelete() {
    if (!roleToDelete) return;
    setSaving(true);
    try {
      await adminService.deleteRole(roleToDelete.id);
      toast.success("Role deleted");
      setRoleToDelete(null);
      await queryClient.invalidateQueries({ queryKey: ["admin", "roles"] });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }

  const canModify = selectedRole && !selectedRole.is_system && !selectedRole.is_admin;
  const isAdminRole = selectedRole?.is_admin;

  return (
    <div className="flex flex-col h-full bg-base-100">
      {/* Header */}
      <header className="px-4 py-5 sm:px-6 lg:px-8 lg:py-6 border-b border-base-200 flex items-center justify-between bg-base-100/50 backdrop-blur-xl sticky top-0 z-10">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Role Management</h1>
          <p className="text-sm text-base-content/60 mt-1">Create, edit roles and manage permissions</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={() => {
              void refetchRoles();
              void refetchPermissions();
            }}
            className="btn btn-square btn-ghost btn-sm sm:btn-md"
            title="Refresh"
          >
            <RefreshCw className={`h-5 w-5 ${loading ? "animate-spin" : ""}`} />
          </button>
          <button onClick={openCreate} className="btn btn-primary btn-sm sm:btn-md gap-2">
            <Plus className="h-4 w-4" />
            Create Role
          </button>
        </div>
      </header>

      {/* Main Content */}
      <div className="flex-1 overflow-auto p-4 sm:p-6 lg:p-8 flex flex-col xl:flex-row gap-6 items-start">
        {/* Left: Role List */}
        <div className="w-full xl:flex-1 card bg-base-100 border border-base-200 shadow-sm overflow-hidden">
          <div className="overflow-x-auto">
            <table className="table w-full">
              <thead>
                <tr className="bg-base-200/50">
                  <th className="font-semibold uppercase text-xs tracking-wider opacity-70">Role</th>
                  <th className="font-semibold uppercase text-xs tracking-wider opacity-70">Type</th>
                  <th className="font-semibold uppercase text-xs tracking-wider opacity-70">Auto Assign</th>
                  <th className="font-semibold uppercase text-xs tracking-wider opacity-70">Permissions</th>
                  <th className="font-semibold uppercase text-xs tracking-wider opacity-70">Status</th>
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  <tr>
                    <td colSpan={5} className="py-12 text-center opacity-50">
                      <Loader2 className="animate-spin h-6 w-6 mx-auto mb-2 text-primary" />
                      Loading roles...
                    </td>
                  </tr>
                ) : roles.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="py-12 text-center opacity-50">No roles found</td>
                  </tr>
                ) : (
                  roles.map((role) => (
                    <tr
                      key={role.id}
                      className={`cursor-pointer hover:bg-base-200/50 transition-colors ${
                        selectedRole?.id === role.id ? "bg-primary/10 border-l-4 border-primary" : ""
                      }`}
                      onClick={() => setSelectedRole(role)}
                    >
                      <td>
                        <div className="flex items-center gap-2">
                          {role.is_admin && <Shield className="h-4 w-4 text-warning" />}
                          <span className={`font-semibold ${role.is_admin ? "text-warning" : "text-primary"}`}>
                            {role.name}
                          </span>
                          {role.is_system && (
                            <span className="badge badge-xs badge-outline">system</span>
                          )}
                        </div>
                        <p className="text-xs text-base-content/50 mt-0.5 truncate max-w-[200px]">
                          {role.description || "—"}
                        </p>
                      </td>
                      <td>
                        {role.is_admin ? (
                          <span className="badge badge-warning badge-sm">Admin</span>
                        ) : role.is_system ? (
                          <span className="badge badge-info badge-sm">System</span>
                        ) : (
                          <span className="badge badge-ghost badge-sm">Custom</span>
                        )}
                      </td>
                      <td>
                        {role.auto_assign ? (
                          <span className="badge badge-success badge-sm">Yes</span>
                        ) : (
                          <span className="text-xs opacity-40">No</span>
                        )}
                      </td>
                      <td>
                        <span className="text-sm font-medium">
                          {role.permissions?.length || 0} assigned
                        </span>
                      </td>
                      <td>
                        <span
                          className={`badge badge-sm font-medium ${
                            role.is_deleted ? "badge-error badge-outline" : "badge-success badge-outline"
                          }`}
                        >
                          {role.is_deleted ? "Deleted" : "Active"}
                        </span>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* Right: Role Details & Permissions */}
        <aside className="w-full xl:w-[500px] shrink-0 flex flex-col gap-4">
          {selectedRole ? (
            <>
              {/* Role Info Card */}
              <div className="card bg-base-100 border border-base-200 shadow-sm p-5">
                <div className="flex items-center justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <div
                      className={`h-10 w-10 rounded-full flex items-center justify-center border ${
                        isAdminRole
                          ? "bg-warning/10 text-warning border-warning/20"
                          : "bg-primary/10 text-primary border-primary/20"
                      }`}
                    >
                      <Shield size={20} />
                    </div>
                    <div>
                      <h2 className="text-lg font-bold flex items-center gap-2">
                        {selectedRole.name}
                        {selectedRole.is_admin && (
                          <span className="badge badge-warning badge-xs">admin</span>
                        )}
                        {selectedRole.is_system && (
                          <span className="badge badge-info badge-xs">system</span>
                        )}
                      </h2>
                      <p className="text-xs opacity-50">ID: {selectedRole.id}</p>
                    </div>
                  </div>
                  <div className="flex gap-1">
                    {canModify && (
                      <button
                        onClick={openEdit}
                        className="btn btn-ghost btn-sm btn-square"
                        title="Edit role"
                      >
                        <Pencil className="h-4 w-4" />
                      </button>
                    )}
                    {canModify && (
                      <button
                        onClick={() => setRoleToDelete(selectedRole)}
                        className="btn btn-ghost btn-sm btn-square text-error"
                        title="Delete role"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    )}
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <span className="block text-xs font-medium uppercase tracking-wider opacity-50">Description</span>
                    <span className="font-medium">{selectedRole.description || "—"}</span>
                  </div>
                  <div>
                    <span className="block text-xs font-medium uppercase tracking-wider opacity-50">Auto Assign</span>
                    <span className={`badge ${selectedRole.auto_assign ? "badge-success" : "badge-ghost"} badge-sm`}>
                      {selectedRole.auto_assign ? "Yes" : "No"}
                    </span>
                  </div>
                  <div>
                    <span className="block text-xs font-medium uppercase tracking-wider opacity-50">Created</span>
                    <span className="text-xs opacity-70">
                      {selectedRole.created_at ? new Date(selectedRole.created_at).toLocaleString() : "—"}
                    </span>
                  </div>
                </div>
              </div>

              {/* Permissions Card */}
              <div className="card bg-base-100 border border-base-200 shadow-sm p-5">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="font-bold text-sm uppercase tracking-wider opacity-70">
                    Permissions ({assignments.length}/{permissions.length})
                  </h3>
                  {!isAdminRole && (
                    <button
                      onClick={() => void savePermissions()}
                      disabled={saving}
                      className="btn btn-primary btn-sm gap-1"
                    >
                      {saving ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <Save className="h-4 w-4" />
                      )}
                      Save Permissions
                    </button>
                  )}
                  {isAdminRole && (
                    <span className="text-xs text-warning font-medium">Full access — managed by system</span>
                  )}
                </div>

                {isAdminRole ? (
                  <div className="text-sm opacity-60 italic text-center py-4 border border-dashed border-base-200 rounded-lg bg-base-200/30">
                    Admin roles automatically have full access to all resources. Permissions cannot be modified.
                  </div>
                ) : (
                  <div className="space-y-1 max-h-[500px] overflow-y-auto pr-1">
                    {permissions.map((perm) => {
                      const assigned = isAssigned(perm.key);
                      const assignment = getAssignment(perm.key);
                      const libInput = libraryIdsInput[perm.key] ?? (
                        assignment?.conditions && "library_ids" in assignment.conditions
                          ? (assignment.conditions.library_ids as string[]).join(", ")
                          : ""
                      );

                      return (
                        <div
                          key={perm.key}
                          className={`border rounded-lg p-3 transition-colors ${
                            assigned ? "border-primary/30 bg-primary/5" : "border-base-200 bg-base-100"
                          }`}
                        >
                          <div className="flex items-start gap-3">
                            <input
                              type="checkbox"
                              className="checkbox checkbox-sm checkbox-primary mt-0.5"
                              checked={assigned}
                              onChange={() => togglePermission(perm.key)}
                            />
                            <div className="flex-1 min-w-0">
                              <div className="flex items-center gap-2 flex-wrap">
                                <label
                                  className={`text-sm font-semibold cursor-pointer ${
                                    assigned ? "text-primary" : "text-base-content/70"
                                  }`}
                                  onClick={() => togglePermission(perm.key)}
                                >
                                  {perm.key}
                                </label>
                                {assigned && (
                                  <select
                                    className="select select-xs select-bordered"
                                    value={assignment?.effect || "allow"}
                                    onChange={(e) =>
                                      setEffect(perm.key, e.target.value as "allow" | "deny")
                                    }
                                  >
                                    <option value="allow">Allow</option>
                                    <option value="deny">Deny</option>
                                  </select>
                                )}
                              </div>
                              <p className="text-[11px] text-base-content/40 mt-0.5">{perm.description}</p>

                              {assigned && (
                                <div className="mt-2 flex items-center gap-2">
                                  <input
                                    type="text"
                                    className="input input-xs input-bordered flex-1 text-xs"
                                    placeholder="Library IDs (comma separated, empty = all)"
                                    value={libInput}
                                    onChange={(e) => setLibraryIds(perm.key, e.target.value)}
                                    onBlur={() => applyLibraryIds(perm.key)}
                                    onKeyDown={(e) => {
                                      if (e.key === "Enter") {
                                        e.preventDefault();
                                        applyLibraryIds(perm.key);
                                      }
                                    }}
                                  />
                                </div>
                              )}
                            </div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            </>
          ) : (
            <div className="card bg-base-100 border border-base-200 shadow-sm p-6 text-center opacity-50 italic">
              Select a role to view and manage permissions.
            </div>
          )}
        </aside>
      </div>

      {/* Create/Edit Modal */}
      <dialog className={`modal ${showModal ? "modal-open" : ""}`}>
        <div className="modal-box">
          <button
            onClick={() => setShowModal(false)}
            className="btn btn-sm btn-circle btn-ghost absolute right-2 top-2"
          >
            <X className="h-4 w-4" />
          </button>
          <h3 className="font-bold text-lg border-b border-base-200 pb-4 mb-4">
            {modalMode === "create" ? "Create New Role" : "Edit Role"}
          </h3>
          <form onSubmit={handleSave}>
            <div className="flex flex-col gap-4">
              <div className="flex flex-col gap-1.5">
                <label className="text-sm font-medium pl-1">Role Name</label>
                <input
                  type="text"
                  required
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  className="input input-bordered focus:input-primary"
                  placeholder="e.g. EDITOR"
                  disabled={modalMode === "edit" && selectedRole?.is_system}
                />
              </div>
              <div className="flex flex-col gap-1.5">
                <label className="text-sm font-medium pl-1">Description</label>
                <textarea
                  className="textarea textarea-bordered focus:textarea-primary"
                  value={form.description}
                  onChange={(e) => setForm({ ...form, description: e.target.value })}
                  placeholder="Role description..."
                  rows={2}
                />
              </div>
              <label className="flex items-center gap-3 cursor-pointer p-3 bg-base-200/50 rounded-lg">
                <input
                  type="checkbox"
                  className="toggle toggle-primary"
                  checked={form.auto_assign}
                  onChange={(e) => setForm({ ...form, auto_assign: e.target.checked })}
                  disabled={modalMode === "edit" && selectedRole?.is_system}
                />
                <div>
                  <span className="text-sm font-medium">Auto-assign to new users</span>
                  <p className="text-xs text-base-content/50">
                    Newly registered users will automatically receive this role.
                  </p>
                </div>
              </label>
            </div>
            <div className="modal-action mt-6">
              <button
                type="button"
                onClick={() => setShowModal(false)}
                disabled={saving}
                className="btn btn-ghost"
              >
                Cancel
              </button>
              <button type="submit" disabled={saving} className="btn btn-primary">
                {saving && <span className="loading loading-spinner"></span>}
                {modalMode === "create" ? "Create Role" : "Save Changes"}
              </button>
            </div>
          </form>
        </div>
        <form method="dialog" className="modal-backdrop">
          <button onClick={() => setShowModal(false)}>close</button>
        </form>
      </dialog>

      {/* Delete Confirmation Modal */}
      <dialog className={`modal ${roleToDelete ? "modal-open" : ""}`}>
        <div className="modal-box">
          <h3 className="font-bold text-lg text-error flex items-center gap-2">
            <AlertCircle className="w-6 h-6" />
            Delete Role
          </h3>
          <p className="py-4 text-sm opacity-80">
            Are you sure you want to delete the role <strong>{roleToDelete?.name}</strong>? Users
            assigned to this role will lose its permissions. This action cannot be undone.
          </p>
          <div className="modal-action">
            <button onClick={() => setRoleToDelete(null)} className="btn btn-ghost">
              Cancel
            </button>
            <button
              onClick={() => void confirmDelete()}
              className="btn btn-error"
              disabled={saving}
            >
              {saving ? <span className="loading loading-spinner loading-xs"></span> : "Delete"}
            </button>
          </div>
        </div>
        <form method="dialog" className="modal-backdrop">
          <button onClick={() => setRoleToDelete(null)}>close</button>
        </form>
      </dialog>
    </div>
  );
}
