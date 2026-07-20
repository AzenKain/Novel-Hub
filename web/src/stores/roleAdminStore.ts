import { adminService } from "@/services";
import type { Permission, Role } from "@/types";
import { create } from "zustand";

export interface PermissionAssignment {
  permission_key: string;
  effect: "allow" | "deny";
  conditions: Record<string, unknown>;
}

interface RoleAdminState {
  roles: Role[];
  permissions: Permission[];
  selectedRole: Role | null;
  loading: boolean;
  saving: boolean;
  error: string;
  assignments: PermissionAssignment[];
  showModal: boolean;
  modalMode: "create" | "edit";
  form: { name: string; description: string; auto_assign: boolean };
  roleToDelete: Role | null;
  libraryIdsInput: Record<string, string>;

  setRoles: (roles: Role[]) => void;
  setPermissions: (permissions: Permission[]) => void;
  setSelectedRole: (role: Role | null | ((prev: Role | null) => Role | null)) => void;
  setLoading: (loading: boolean) => void;
  setSaving: (saving: boolean) => void;
  setError: (error: string) => void;
  setAssignments: (assignments: PermissionAssignment[] | ((prev: PermissionAssignment[]) => PermissionAssignment[])) => void;
  setShowModal: (show: boolean) => void;
  setModalMode: (mode: "create" | "edit") => void;
  setForm: (form: { name: string; description: string; auto_assign: boolean } | ((prev: any) => any)) => void;
  setRoleToDelete: (role: Role | null) => void;
  setLibraryIdsInput: (input: Record<string, string> | ((prev: Record<string, string>) => Record<string, string>)) => void;

  loadData: () => Promise<void>;
  reset: () => void;
}

const initialForm = { name: "", description: "", auto_assign: false };

const initialState = {
  roles: [],
  permissions: [],
  selectedRole: null,
  loading: true,
  saving: false,
  error: "",
  assignments: [],
  showModal: false,
  modalMode: "create" as "create" | "edit",
  form: initialForm,
  roleToDelete: null,
  libraryIdsInput: {},
};

export const useRoleAdminStore = create<RoleAdminState>((set, get) => ({
  ...initialState,

  setRoles: (roles) => set({ roles }),
  setPermissions: (permissions) => set({ permissions }),
  setSelectedRole: (selectedRole) => set((state) => ({ selectedRole: typeof selectedRole === "function" ? selectedRole(state.selectedRole) : selectedRole })),
  setLoading: (loading) => set({ loading }),
  setSaving: (saving) => set({ saving }),
  setError: (error) => set({ error }),
  setAssignments: (assignments) => set((state) => ({ assignments: typeof assignments === "function" ? assignments(state.assignments) : assignments })),
  setShowModal: (showModal) => set({ showModal }),
  setModalMode: (modalMode) => set({ modalMode }),
  setForm: (form) => set((state) => ({ form: typeof form === "function" ? form(state.form) : form })),
  setRoleToDelete: (roleToDelete) => set({ roleToDelete }),
  setLibraryIdsInput: (libraryIdsInput) => set((state) => ({ libraryIdsInput: typeof libraryIdsInput === "function" ? libraryIdsInput(state.libraryIdsInput) : libraryIdsInput })),

  loadData: async () => {
    set({ loading: true, error: "" });
    try {
      const [roleRes, permRes] = await Promise.all([
        adminService.getRoles(),
        adminService.getPermissions(),
      ]);
      const nextRoles = roleRes.data || [];
      const nextPerms = permRes.data || [];
      set({ roles: nextRoles, permissions: nextPerms });

      const current = get().selectedRole;
      if (current && nextRoles.some((r) => r.id === current.id)) {
        set({ selectedRole: nextRoles.find((r) => r.id === current.id) || nextRoles[0] || null });
      } else {
        set({ selectedRole: nextRoles[0] || null });
      }
    } catch (err) {
      set({ error: err instanceof Error ? err.message : String(err) });
    } finally {
      set({ loading: false });
    }
  },

  reset: () => set(initialState),
}));
