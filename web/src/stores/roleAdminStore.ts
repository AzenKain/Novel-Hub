import type { Role } from "@/types";
import { create } from "zustand";

export interface PermissionAssignment {
  permission_key: string;
  effect: "allow" | "deny";
  conditions: Record<string, unknown>;
}

interface RoleAdminState {
  selectedRole: Role | null;
  error: string;
  assignments: PermissionAssignment[];
  showModal: boolean;
  modalMode: "create" | "edit";
  form: { name: string; description: string; auto_assign: boolean };
  roleToDelete: Role | null;

  setSelectedRole: (
    role: Role | null | ((prev: Role | null) => Role | null),
  ) => void;
  setError: (error: string) => void;
  setAssignments: (
    assignments:
      | PermissionAssignment[]
      | ((prev: PermissionAssignment[]) => PermissionAssignment[]),
  ) => void;
  setShowModal: (show: boolean) => void;
  setModalMode: (mode: "create" | "edit") => void;
  setForm: (
    form:
      | { name: string; description: string; auto_assign: boolean }
      | ((prev: any) => any),
  ) => void;
  setRoleToDelete: (role: Role | null) => void;

  reset: () => void;
}

const initialForm = { name: "", description: "", auto_assign: false };

const initialState = {
  selectedRole: null,
  error: "",
  assignments: [],
  showModal: false,
  modalMode: "create" as "create" | "edit",
  form: initialForm,
  roleToDelete: null,
};

export const useRoleAdminStore = create<RoleAdminState>((set) => ({
  ...initialState,

  setSelectedRole: (selectedRole) =>
    set((state) => ({
      selectedRole:
        typeof selectedRole === "function"
          ? selectedRole(state.selectedRole)
          : selectedRole,
    })),
  setError: (error) => set({ error }),
  setAssignments: (assignments) =>
    set((state) => ({
      assignments:
        typeof assignments === "function"
          ? assignments(state.assignments)
          : assignments,
    })),
  setShowModal: (showModal) => set({ showModal }),
  setModalMode: (modalMode) => set({ modalMode }),
  setForm: (form) =>
    set((state) => ({
      form: typeof form === "function" ? form(state.form) : form,
    })),
  setRoleToDelete: (roleToDelete) => set({ roleToDelete }),

  reset: () => set(initialState),
}));
