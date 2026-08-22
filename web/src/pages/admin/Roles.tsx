import {
  useAssignRolePermissionsMutation,
  useCreateRoleMutation,
  useDeleteRoleMutation,
  useLibrariesQuery,
  usePermissionsQuery,
  useReorderRolesMutation,
  useRolesQuery,
  useUpdateRoleMutation,
} from "@/hooks";
import { LibraryScopeSelector } from "@/components/admin";
import type { CreateRoleRequest } from "@/types";
import {
  AlertCircle,
  ChevronDown,
  ChevronUp,
  GripVertical,
  Loader2,
  Pencil,
  Plus,
  RefreshCw,
  Save,
  Shield,
  Trash2,
} from "lucide-react";
import { SyntheticEvent, useEffect, useState } from "react";
import { toast } from "react-toastify";

interface PermissionAssignment {
  permission_key: string;
  effect: "allow" | "deny";
  conditions: Record<string, unknown>;
}

import { useRoleAdminStore } from "@/stores";
import { Trans, useTranslation } from "react-i18next";
import { useQueryClient } from "@tanstack/react-query";
import { useShallow } from "zustand/react/shallow";

export function Roles() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { data: roles = [], isLoading: rolesLoading, isFetching: rolesFetching, refetch: refetchRoles } = useRolesQuery();
  const { data: permissions = [], isLoading: permissionsLoading, isFetching: permissionsFetching, refetch: refetchPermissions } = usePermissionsQuery();
  const { data: libraries = [] } = useLibrariesQuery();

  const createRoleMutation = useCreateRoleMutation();
  const updateRoleMutation = useUpdateRoleMutation();
  const deleteRoleMutation = useDeleteRoleMutation();
  const assignPermissionsMutation = useAssignRolePermissionsMutation();
  const reorderRolesMutation = useReorderRolesMutation();

  const reorderRoles = (ids: string[]) =>
    reorderRolesMutation.mutate(ids, {
      onError: (err) => toast.error(err instanceof Error ? err.message : String(err)),
    });

  const handleMoveUp = (index: number, e: SyntheticEvent) => {
    e.stopPropagation();
    if (index <= 0) return;
    const newRoles = [...roles];
    const temp = newRoles[index - 1];
    newRoles[index - 1] = newRoles[index];
    newRoles[index] = temp;
    reorderRoles(newRoles.map((r) => r.id));
  };

  const handleMoveDown = (index: number, e: SyntheticEvent) => {
    e.stopPropagation();
    if (index >= roles.length - 1) return;
    const newRoles = [...roles];
    const temp = newRoles[index + 1];
    newRoles[index + 1] = newRoles[index];
    newRoles[index] = temp;
    reorderRoles(newRoles.map((r) => r.id));
  };

  const [draggedIndex, setDraggedIndex] = useState<number | null>(null);

  const handleDragStart = (index: number) => {
    setDraggedIndex(index);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
  };

  const handleDrop = (dropIndex: number) => {
    if (draggedIndex === null || draggedIndex === dropIndex) return;
    const newRoles = [...roles];
    const [moved] = newRoles.splice(draggedIndex, 1);
    newRoles.splice(dropIndex, 0, moved);
    setDraggedIndex(null);
    reorderRoles(newRoles.map((r) => r.id));
  };

  const {
    selectedRole, setSelectedRole,
    assignments, setAssignments,
    showModal, setShowModal,
    modalMode, setModalMode,
    form, setForm,
    roleToDelete, setRoleToDelete,
  } = useRoleAdminStore(useShallow((state) => ({
    selectedRole: state.selectedRole, setSelectedRole: state.setSelectedRole,
    assignments: state.assignments, setAssignments: state.setAssignments,
    showModal: state.showModal, setShowModal: state.setShowModal,
    modalMode: state.modalMode, setModalMode: state.setModalMode,
    form: state.form, setForm: state.setForm,
    roleToDelete: state.roleToDelete, setRoleToDelete: state.setRoleToDelete,
  })));

  const loading = rolesLoading || permissionsLoading;
  const saving =
    createRoleMutation.isPending ||
    updateRoleMutation.isPending ||
    deleteRoleMutation.isPending ||
    assignPermissionsMutation.isPending;

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
          toast.success(t("admin.role_created", "Role created"));
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
            toast.success(t("admin.role_updated", "Role updated"));
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
        toast.success(t("admin.role_deleted", "Role deleted"));
        setRoleToDelete(null);
      },
      onError: (err) => toast.error(err instanceof Error ? err.message : String(err)),
    });
  }

  const canModify = selectedRole && !selectedRole.is_admin && !selectedRole.is_banned;

  return (
    <div className="flex flex-col h-full bg-base-100">
      {/* Header */}
      <header className="px-4 py-5 sm:px-6 lg:px-8 lg:py-6 border-b border-base-200 flex items-center justify-between bg-base-100/50 backdrop-blur-xl sticky top-0 z-10">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("admin.role_management", "Role Management")}</h1>
          <p className="text-sm text-base-content/60 mt-1">{t("admin.role_management_desc", "Create, edit roles and manage permissions")}</p>
        </div>
        <button
          onClick={async () => {
            await queryClient.invalidateQueries({ queryKey: ["admin", "roles"] });
            await queryClient.invalidateQueries({ queryKey: ["admin", "permissions"] });
            await Promise.all([refetchRoles(), refetchPermissions()]);
            toast.info(t("common.refreshed", "Đã làm mới dữ liệu"));
          }}
          className="btn btn-square btn-ghost btn-sm sm:btn-md"
          title={t("admin.operations.refresh", "Refresh")}
          disabled={rolesFetching || permissionsFetching}
        >
          <RefreshCw className={`h-5 w-5 ${(rolesFetching || permissionsFetching) ? "animate-spin" : ""}`} />
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
                <Plus className="h-4 w-4" /> {t("admin.add_role", "Add Role")}
              </button>
            </div>

            <div className="space-y-2">
              {roles.map((r, index) => {
                const isSelected = selectedRole?.id === r.id;
                return (
                  <div
                    key={r.id}
                    draggable
                    onDragStart={() => handleDragStart(index)}
                    onDragOver={handleDragOver}
                    onDrop={() => handleDrop(index)}
                    onClick={() => setSelectedRole(r)}
                    className={`p-4 rounded-xl border transition-all cursor-pointer flex items-center justify-between ${isSelected
                      ? "bg-primary/10 border-primary shadow-sm"
                      : "bg-base-100 border-base-200 hover:border-base-300"
                      }`}
                  >
                    <div className="min-w-0 flex-1 flex items-center gap-2">
                      <GripVertical className="h-4 w-4 shrink-0 text-base-content/30 cursor-grab active:cursor-grabbing" />
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <Shield className={`h-4 w-4 shrink-0 ${isSelected ? "text-primary" : "text-base-content/40"}`} />
                          <span className="font-bold text-sm truncate">{r.name}</span>
                          <span className="badge badge-ghost badge-xs font-mono">{t("admin.role_rank", "Rank #{{n}}", { n: index + 1 })}</span>
                          {r.is_admin && <span className="badge badge-error badge-xs">{t("admin.role_badge_admin", "Admin")}</span>}
                          {r.is_system && !r.is_admin && <span className="badge badge-neutral badge-xs">{t("admin.role_badge_system", "System")}</span>}
                          {r.auto_assign && <span className="badge badge-info badge-xs">{t("admin.role_badge_auto", "Auto")}</span>}
                        </div>
                        {r.description && (
                          <p className="text-xs text-base-content/60 truncate mt-1 pl-6">{r.description}</p>
                        )}
                      </div>
                    </div>

                    <div className="flex items-center gap-1 shrink-0 ml-2">
                      <button
                        disabled={index === 0 || reorderRolesMutation.isPending}
                        onClick={(e) => handleMoveUp(index, e)}
                        className="btn btn-ghost btn-xs btn-square"
                        title={t("admin.role_move_up", "Move Up Priority")}
                      >
                        <ChevronUp className="h-3.5 w-3.5" />
                      </button>
                      <button
                        disabled={index === roles.length - 1 || reorderRolesMutation.isPending}
                        onClick={(e) => handleMoveDown(index, e)}
                        className="btn btn-ghost btn-xs btn-square"
                        title={t("admin.role_move_down", "Move Down Priority")}
                      >
                        <ChevronDown className="h-3.5 w-3.5" />
                      </button>
                      {!r.is_admin && !r.is_system && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            setRoleToDelete(r);
                          }}
                          className="btn btn-ghost btn-xs text-error btn-square"
                          title={t("admin.role_delete", "Delete role")}
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
                      {selectedRole.is_admin && <span className="badge badge-error">{t("admin.role_full_access", "Full Admin Access")}</span>}
                      {selectedRole.is_banned && <span className="badge badge-warning text-warning-content font-bold">{t("admin.role_blocked", "Blocked Account")}</span>}
                    </div>
                    <p className="text-xs text-base-content/60 mt-1">
                      {selectedRole.description || t("admin.role_no_description", "No description provided.")}
                    </p>
                  </div>

                  <div className="flex items-center gap-2 self-start sm:self-center">
                    {canModify && (
                      <button onClick={openEdit} className="btn btn-outline btn-sm gap-1.5">
                        <Pencil className="h-3.5 w-3.5" /> {t("admin.role_edit", "Edit Role")}
                      </button>
                    )}
                    {!selectedRole.is_admin && !selectedRole.is_banned && (
                      <button
                        onClick={savePermissions}
                        disabled={saving}
                        className="btn btn-primary btn-sm gap-1.5"
                      >
                        {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                        {t("admin.role_save_permissions", "Save Permissions")}
                      </button>
                    )}
                  </div>
                </div>

                {selectedRole.is_admin ? (
                  <div className="bg-base-200/30 border border-base-200 rounded-xl p-8 text-center flex flex-col items-center gap-2">
                    <Shield className="h-12 w-12 text-error opacity-80" />
                    <h3 className="font-bold text-lg">{t("admin.role_admin_title", "Administrator Role")}</h3>
                    <p className="text-xs text-base-content/60 max-w-md">
                      {t("admin.role_admin_desc", "Users with the ADMIN role automatically bypass all permission checks and have full unrestricted system access.")}
                    </p>
                  </div>
                ) : selectedRole.is_banned ? (
                  <div className="bg-base-200/30 border border-base-200 rounded-xl p-8 text-center flex flex-col items-center gap-2">
                    <Shield className="h-12 w-12 text-warning opacity-80" />
                    <h3 className="font-bold text-lg">{t("admin.role_banned_title", "Banned / Blocked Role")}</h3>
                    <p className="text-xs text-base-content/60 max-w-md">
                      {t("admin.role_banned_desc", "Users with the BANNED role are blocked immediately. They are denied all actions and system permissions.")}
                    </p>
                  </div>
                ) : (
                  <div className="bg-base-100 border border-base-200 rounded-xl overflow-hidden shadow-sm">
                    <div className="p-4 border-b border-base-200 bg-base-200/30 font-bold text-sm flex items-center justify-between">
                      <span>{t("admin.role_permissions_count", "Permissions ({{n}})", { n: permissions.length })}</span>
                      <span className="text-xs font-normal text-base-content/60">
                        {t("admin.role_assigned_count", "{{n}} assigned", { n: assignments.length })}
                      </span>
                    </div>

                    <div className="divide-y divide-base-200">
                      {(() => {
                        const categories = [
                          {
                            id: "reading",
                            name: "📖 Book Reading & Discovery",
                            keys: ["book.read", "book.tts", "book.search.deep", "book.download", "book.offline", "book.send_email", "book.share"]
                          },
                          {
                            id: "interactions",
                            name: "💬 Interactions & Personal Features",
                            keys: ["book.bookmark", "book.collection", "book.highlight", "book.review.create", "book.review.delete", "user.stats.read", "tracker.sync"]
                          },
                          {
                            id: "content",
                            name: "📦 Book Content Management",
                            keys: ["book.upload", "book.edit", "book.metadata.fetch", "book.delete", "book.duplicate.manage", "book.archive", "book.bulk.manage"]
                          },
                          {
                            id: "library",
                            name: "📚 Library Management",
                            keys: ["library.read", "library.manage"]
                          },
                          {
                            id: "integration",
                            name: "🔄 External Sync & Integration",
                            keys: ["opds.read", "opds.download", "kobo.sync", "komga.sync", "calibre.sync"]
                          },
                          {
                            id: "admin",
                            name: "⚙️ System Administration",
                            keys: ["admin.access", "user.manage", "role.manage", "setting.manage", "job.read", "job.manage", "system.log.read", "system.backup", "webhook.manage"]
                          }
                        ];
                        const allCategoryKeys = categories.flatMap((cat) => cat.keys);

                        return categories.map((category) => {
                          const categoryPerms = permissions.filter((p) => category.keys.includes(p.key));
                          const otherPerms = permissions.filter((p) => !allCategoryKeys.includes(p.key));
                          if (category.id === "admin" && otherPerms.length > 0) {
                            categoryPerms.push(...otherPerms);
                          }
                        if (categoryPerms.length === 0) return null;

                        const assignedCount = categoryPerms.filter((p) => isAssigned(p.key)).length;

                        return (
                          <div key={category.id} className="border-b border-base-200 last:border-b-0">
                            <div className="bg-base-200/40 px-4 py-2.5 flex items-center justify-between font-semibold text-xs tracking-wide uppercase text-base-content/70">
                              <span>{category.name}</span>
                              <span className="badge badge-sm badge-ghost">{assignedCount}/{categoryPerms.length} assigned</span>
                            </div>
                            <div className="divide-y divide-base-200/60">
                              {categoryPerms.map((perm) => {
                                const assigned = isAssigned(perm.key);
                                const assignment = getAssignment(perm.key);

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
                                        </div>
                                      </div>

                                      {assigned && (
                                        <div className="flex items-center gap-1 bg-base-200 p-1 rounded-lg shrink-0">
                                          <button
                                            type="button"
                                            onClick={() => setEffect(perm.key, "allow")}
                                            className={`btn btn-xs ${assignment?.effect === "allow" ? "btn-success font-bold" : "btn-ghost text-base-content/60"}`}
                                          >
                                            {t("admin.role_effect_allow", "Allow")}
                                          </button>
                                          <button
                                            type="button"
                                            onClick={() => setEffect(perm.key, "deny")}
                                            className={`btn btn-xs ${assignment?.effect === "deny" ? "btn-error font-bold" : "btn-ghost text-base-content/60"}`}
                                          >
                                            {t("admin.role_effect_deny", "Deny")}
                                          </button>
                                        </div>
                                      )}
                                    </div>

                                    {/* Conditional Scope */}
                                    {assigned && (
                                      <div className="pl-7">
                                        <LibraryScopeSelector
                                          selectedLibraryIds={(assignment?.conditions?.library_ids as string[]) || []}
                                          onChange={(ids) => {
                                            setAssignments((prev) =>
                                              prev.map((a) =>
                                                a.permission_key === perm.key
                                                  ? { ...a, conditions: ids.length > 0 ? { library_ids: ids } : {} }
                                                  : a
                                              )
                                            );
                                          }}
                                          libraries={libraries}
                                        />
                                      </div>
                                    )}
                                  </div>
                                );
                              })}
                            </div>
                          </div>
                        );
                      });
                    })()}
                    </div>
                  </div>
                )}
              </>
            ) : (
              <div className="bg-base-200/30 border border-base-200 rounded-xl p-12 text-center flex flex-col items-center gap-2">
                <Shield className="h-10 w-10 text-base-content/30" />
                <p className="text-sm font-medium text-base-content/60">{t("admin.role_select_prompt", "Select a role from the sidebar to edit permissions.")}</p>
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
              {modalMode === "create" ? t("admin.role_create_title", "Create New Role") : t("admin.role_edit", "Edit Role")}
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
                  placeholder={t("admin.role_name_placeholder", "e.g. LIBRARIAN")}
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
                  placeholder={t("admin.role_desc_placeholder", "Role description and scope")}
                  className="textarea textarea-bordered w-full focus:textarea-primary text-sm"
                />
              </div>

              {!(modalMode === "edit" && (selectedRole?.is_admin || selectedRole?.is_banned || selectedRole?.name?.toUpperCase() === "GUEST")) && (
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
              )}

              <div className="modal-action">
                <button type="button" onClick={() => setShowModal(false)} className="btn btn-ghost">
                  {t("common.cancel", "Cancel")}
                </button>
                <button type="submit" disabled={saving} className="btn btn-primary">
                  {saving ? <span className="loading loading-spinner"></span> : modalMode === "create" ? t("admin.role_create_action", "Create Role") : t("admin.save_changes", "Save Changes")}
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
            <h3 className="font-bold text-lg">{t("admin.role_delete_title", "Delete Role?")}</h3>
            <p className="text-xs text-base-content/60 mt-1 mb-6">
              <Trans
                i18nKey="admin.role_delete_confirm"
                values={{ name: roleToDelete.name }}
                defaults="Are you sure you want to delete role <bold>{{name}}</bold>? Users assigned this role will lose its permissions."
                components={{ bold: <span className="font-bold text-base-content" /> }}
              />
            </p>
            <div className="flex gap-2 justify-end">
              <button onClick={() => setRoleToDelete(null)} className="btn btn-ghost flex-1">
                {t("common.cancel", "Cancel")}
              </button>
              <button onClick={confirmDelete} disabled={saving} className="btn btn-error text-white flex-1">
                {saving ? <span className="loading loading-spinner"></span> : t("common.delete", "Delete")}
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
