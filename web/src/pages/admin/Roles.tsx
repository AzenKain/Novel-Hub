import {
  useAssignRolePermissionsMutation,
  useCreateRoleMutation,
  useDeleteRoleMutation,
  usePermissionsQuery,
  useRolesQuery,
  useUpdateRoleMutation,
} from "@/hooks";
import type { CreateRoleRequest } from "@/types";
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
import { SyntheticEvent, useEffect } from "react";
import { toast } from "react-toastify";

interface PermissionAssignment {
  permission_key: string;
  effect: "allow" | "deny";
  conditions: Record<string, unknown>;
}

import { useRoleAdminStore } from "@/stores";
import { useShallow } from "zustand/react/shallow";

export function Roles() {
  const { data: roles = [], isLoading: rolesLoading, refetch: refetchRoles } = useRolesQuery();
  const { data: permissions = [], isLoading: permissionsLoading, refetch: refetchPermissions } = usePermissionsQuery();

  const createRoleMutation = useCreateRoleMutation();
  const updateRoleMutation = useUpdateRoleMutation();
  const deleteRoleMutation = useDeleteRoleMutation();
  const assignPermissionsMutation = useAssignRolePermissionsMutation();

  const {
    selectedRole, setSelectedRole,
    assignments, setAssignments,
    showModal, setShowModal,
    modalMode, setModalMode,
    form, setForm,
    roleToDelete, setRoleToDelete,
    libraryIdsInput, setLibraryIdsInput,
  } = useRoleAdminStore(useShallow((state) => ({
    selectedRole: state.selectedRole, setSelectedRole: state.setSelectedRole,
    assignments: state.assignments, setAssignments: state.setAssignments,
    showModal: state.showModal, setShowModal: state.setShowModal,
    modalMode: state.modalMode, setModalMode: state.setModalMode,
    form: state.form, setForm: state.setForm,
    roleToDelete: state.roleToDelete, setRoleToDelete: state.setRoleToDelete,
    libraryIdsInput: state.libraryIdsInput, setLibraryIdsInput: state.setLibraryIdsInput,
  })));

  const loading = rolesLoading || permissionsLoading;
  const saving =
    createRoleMutation.isPending ||
    updateRoleMutation.isPending ||
    deleteRoleMutation.isPending ||
    assignPermissionsMutation.isPending;

  // Auto select initial role when roles load
  useEffect(() => {
    if (roles.length > 0) {
      if (!selectedRole || !roles.some((r) => r.id === selectedRole.id)) {
        setSelectedRole(roles[0]);
      } else {
        const updated = roles.find((r) => r.id === selectedRole.id);
        if (updated) setSelectedRole(updated);
      }
    }
  }, [roles]);

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

  function savePermissions() {
    if (!selectedRole) return;
    if (selectedRole.is_admin) {
      toast.warning("Cannot modify admin role permissions.");
      return;
    }
    assignPermissionsMutation.mutate(
      {
        roleID: selectedRole.id,
        assignments: assignments.map((a) => ({
          permission_key: a.permission_key,
          effect: a.effect,
          conditions: a.conditions,
        })),
      },
      {
        onSuccess: () => toast.success("Permissions updated"),
        onError: (err) => toast.error(err instanceof Error ? err.message : String(err)),
      }
    );
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

  function handleSave(e: SyntheticEvent) {
    e.preventDefault();
    if (modalMode === "create") {
      const data: CreateRoleRequest = {
        name: form.name,
        description: form.description,
        auto_assign: form.auto_assign,
      };
      createRoleMutation.mutate(data, {
        onSuccess: () => {
          toast.success("Role created");
          setShowModal(false);
        },
        onError: (err) => toast.error(err instanceof Error ? err.message : String(err)),
      });
    } else if (selectedRole) {
      updateRoleMutation.mutate(
        {
          id: selectedRole.id,
          data: {
            name: form.name,
            description: form.description,
            auto_assign: form.auto_assign,
          },
        },
        {
          onSuccess: () => {
            toast.success("Role updated");
            setShowModal(false);
          },
          onError: (err) => toast.error(err instanceof Error ? err.message : String(err)),
        }
      );
    }
  }

  function confirmDelete() {
    if (!roleToDelete) return;
    deleteRoleMutation.mutate(roleToDelete.id, {
      onSuccess: () => {
        toast.success("Role deleted");
        setRoleToDelete(null);
      },
      onError: (err) => toast.error(err instanceof Error ? err.message : String(err)),
    });
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
      </header>

      {/* Main Content */}
      <div className="flex-1 overflow-auto p-4 sm:p-6 lg:p-8">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 max-w-7xl mx-auto h-full">
          {/* Roles Sidebar */}
          <div className="lg:col-span-4 flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-bold">Roles ({roles.length})</h2>
              <button onClick={openCreate} className="btn btn-primary btn-sm gap-1.5">
                <Plus className="h-4 w-4" /> Add Role
              </button>
            </div>

            <div className="space-y-2">
              {roles.map((r) => {
                const isSelected = selectedRole?.id === r.id;
                return (
                  <div
                    key={r.id}
                    onClick={() => setSelectedRole(r)}
                    className={`p-4 rounded-xl border transition-all cursor-pointer flex items-center justify-between ${isSelected
                        ? "bg-primary/10 border-primary shadow-sm"
                        : "bg-base-100 border-base-200 hover:border-base-300"
                      }`}
                  >
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <Shield className={`h-4 w-4 shrink-0 ${isSelected ? "text-primary" : "text-base-content/40"}`} />
                        <span className="font-bold text-sm truncate">{r.name}</span>
                        {r.is_admin && <span className="badge badge-error badge-xs">Admin</span>}
                        {r.is_system && !r.is_admin && <span className="badge badge-neutral badge-xs">System</span>}
                        {r.auto_assign && <span className="badge badge-info badge-xs">Auto</span>}
                      </div>
                      {r.description && (
                        <p className="text-xs text-base-content/60 truncate mt-1 pl-6">{r.description}</p>
                      )}
                    </div>

                    <div className="flex items-center gap-1 shrink-0 ml-2">
                      {!r.is_admin && !r.is_system && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            setRoleToDelete(r);
                          }}
                          className="btn btn-ghost btn-xs text-error btn-square"
                          title="Delete role"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Permissions Matrix */}
          <div className="lg:col-span-8 flex flex-col gap-4">
            {selectedRole ? (
              <>
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-base-200/50 p-4 rounded-xl border border-base-200">
                  <div>
                    <div className="flex items-center gap-2">
                      <h2 className="text-xl font-bold">{selectedRole.name}</h2>
                      {selectedRole.is_admin && <span className="badge badge-error">Full Admin Access</span>}
                    </div>
                    <p className="text-xs text-base-content/60 mt-1">
                      {selectedRole.description || "No description provided."}
                    </p>
                  </div>

                  <div className="flex items-center gap-2 self-start sm:self-center">
                    {canModify && (
                      <button onClick={openEdit} className="btn btn-outline btn-sm gap-1.5">
                        <Pencil className="h-3.5 w-3.5" /> Edit Role
                      </button>
                    )}
                    {!isAdminRole && (
                      <button
                        onClick={savePermissions}
                        disabled={saving}
                        className="btn btn-primary btn-sm gap-1.5"
                      >
                        {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                        Save Permissions
                      </button>
                    )}
                  </div>
                </div>

                {isAdminRole ? (
                  <div className="bg-base-200/30 border border-base-200 rounded-xl p-8 text-center flex flex-col items-center gap-2">
                    <Shield className="h-12 w-12 text-error opacity-80" />
                    <h3 className="font-bold text-lg">Administrator Role</h3>
                    <p className="text-xs text-base-content/60 max-w-md">
                      Users with the ADMIN role automatically bypass all permission checks and have full unrestricted system access.
                    </p>
                  </div>
                ) : (
                  <div className="bg-base-100 border border-base-200 rounded-xl overflow-hidden shadow-sm">
                    <div className="p-4 border-b border-base-200 bg-base-200/30 font-bold text-sm flex items-center justify-between">
                      <span>Permissions ({permissions.length})</span>
                      <span className="text-xs font-normal text-base-content/60">
                        {assignments.length} assigned
                      </span>
                    </div>

                    <div className="divide-y divide-base-200">
                      {permissions.map((perm) => {
                        const assigned = isAssigned(perm.key);
                        const assignment = getAssignment(perm.key);
                        const libraryIdsVal = libraryIdsInput[perm.key] ?? (assignment?.conditions?.library_ids as string[])?.join(", ") ?? "";

                        return (
                          <div key={perm.key} className="p-4 flex flex-col gap-3 hover:bg-base-200/20 transition-colors">
                            <div className="flex items-start justify-between gap-4">
                              <div className="flex items-start gap-3">
                                <input
                                  type="checkbox"
                                  checked={assigned}
                                  onChange={() => togglePermission(perm.key)}
                                  className="checkbox checkbox-primary checkbox-sm mt-0.5"
                                />
                                <div>
                                  <div className="font-bold text-sm flex items-center gap-2">
                                    <span>{perm.description || perm.key}</span>
                                    <span className="font-mono text-[10px] bg-base-200 text-base-content/70 px-1.5 py-0.5 rounded">
                                      {perm.key}
                                    </span>
                                  </div>
                                  <p className="text-xs text-base-content/60 mt-0.5">{perm.key}</p>
                                </div>
                              </div>

                              {assigned && (
                                <div className="flex items-center gap-1 bg-base-200 p-1 rounded-lg shrink-0">
                                  <button
                                    type="button"
                                    onClick={() => setEffect(perm.key, "allow")}
                                    className={`btn btn-xs ${assignment?.effect === "allow" ? "btn-success text-white font-bold" : "btn-ghost text-base-content/60"}`}
                                  >
                                    Allow
                                  </button>
                                  <button
                                    type="button"
                                    onClick={() => setEffect(perm.key, "deny")}
                                    className={`btn btn-xs ${assignment?.effect === "deny" ? "btn-error text-white font-bold" : "btn-ghost text-base-content/60"}`}
                                  >
                                    Deny
                                  </button>
                                </div>
                              )}
                            </div>

                            {/* Conditional Scope */}
                            {assigned && (
                              <div className="pl-7 pt-1 flex items-center gap-2">
                                <label className="text-xs font-semibold text-base-content/70 shrink-0">Scope Library IDs:</label>
                                <input
                                  type="text"
                                  value={libraryIdsVal}
                                  onChange={(e) => setLibraryIds(perm.key, e.target.value)}
                                  onBlur={() => applyLibraryIds(perm.key)}
                                  placeholder="All libraries (or comma separated IDs e.g. lib_1, lib_2)"
                                  className="input input-bordered input-xs flex-1 font-mono text-xs"
                                />
                              </div>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  </div>
                )}
              </>
            ) : (
              <div className="bg-base-200/30 border border-base-200 rounded-xl p-12 text-center flex flex-col items-center gap-2">
                <Shield className="h-10 w-10 text-base-content/30" />
                <p className="text-sm font-medium text-base-content/60">Select a role from the sidebar to edit permissions.</p>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Role Create/Edit Modal */}
      {showModal && (
        <dialog className="modal modal-open">
          <div className="modal-box max-w-md">
            <h3 className="font-bold text-lg mb-4">
              {modalMode === "create" ? "Create New Role" : "Edit Role"}
            </h3>
            <form onSubmit={handleSave} className="space-y-4">
              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">Role Name</span>
                </label>
                <input
                  type="text"
                  required
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  placeholder="e.g. LIBRARIAN"
                  className="input input-bordered w-full focus:input-primary font-mono"
                />
              </div>

              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">Description</span>
                </label>
                <textarea
                  rows={2}
                  value={form.description}
                  onChange={(e) => setForm({ ...form, description: e.target.value })}
                  placeholder="Role description and scope"
                  className="textarea textarea-bordered w-full focus:textarea-primary text-sm"
                />
              </div>

              <div className="form-control">
                <label className="label cursor-pointer justify-start gap-3">
                  <input
                    type="checkbox"
                    checked={form.auto_assign}
                    onChange={(e) => setForm({ ...form, auto_assign: e.target.checked })}
                    className="checkbox checkbox-primary checkbox-sm"
                  />
                  <div>
                    <span className="label-text font-semibold">Auto-assign on Register</span>
                    <p className="text-xs text-base-content/60">Newly registered users automatically get this role.</p>
                  </div>
                </label>
              </div>

              <div className="modal-action">
                <button type="button" onClick={() => setShowModal(false)} className="btn btn-ghost">
                  Cancel
                </button>
                <button type="submit" disabled={saving} className="btn btn-primary">
                  {saving ? <span className="loading loading-spinner"></span> : modalMode === "create" ? "Create Role" : "Save Changes"}
                </button>
              </div>
            </form>
          </div>
          <form method="dialog" className="modal-backdrop">
            <button onClick={() => setShowModal(false)}>close</button>
          </form>
        </dialog>
      )}

      {/* Delete Confirmation Modal */}
      {roleToDelete && (
        <dialog className="modal modal-open">
          <div className="modal-box max-w-sm text-center">
            <div className="w-12 h-12 rounded-full bg-error/10 text-error flex items-center justify-center mx-auto mb-3">
              <AlertCircle className="w-6 h-6" />
            </div>
            <h3 className="font-bold text-lg">Delete Role?</h3>
            <p className="text-xs text-base-content/60 mt-1 mb-6">
              Are you sure you want to delete role <span className="font-bold text-base-content">{roleToDelete.name}</span>? Users assigned this role will lose its permissions.
            </p>
            <div className="flex gap-2 justify-end">
              <button onClick={() => setRoleToDelete(null)} className="btn btn-ghost flex-1">
                Cancel
              </button>
              <button onClick={confirmDelete} disabled={saving} className="btn btn-error text-white flex-1">
                {saving ? <span className="loading loading-spinner"></span> : "Delete"}
              </button>
            </div>
          </div>
          <form method="dialog" className="modal-backdrop">
            <button onClick={() => setRoleToDelete(null)}>close</button>
          </form>
        </dialog>
      )}
    </div>
  );
}
