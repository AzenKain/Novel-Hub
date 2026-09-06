import { CreateUserRequest, Role, User } from "@/types";
import { create } from "zustand";

type ModalMode = "create" | "edit" | "password" | "roles" | "email" | null;

const emptyCreate: CreateUserRequest = {
  email: "",
  password: "",
  full_name: "",
  avatar_url: "",
  role_ids: [],
};

interface UserAdminState {
  users: User[];
  roles: Role[];
  selectedUser: User | null;
  selectedRole: Role | null;
  query: string;
  showDeleted: boolean;
  loading: boolean;
  saving: boolean;
  notice: string;
  error: string;
  modal: ModalMode;
  form: CreateUserRequest;
  newPassword: string;
  roleIDs: string[];
  userToDelete: User | null;
  selectedUserIds: string[];
  bulkModal: "delete" | "restore" | "roles" | "info" | "email" | null;

  setUsers: (users: User[]) => void;
  setRoles: (roles: Role[]) => void;
  setSelectedUser: (
    user: User | null | ((prev: User | null) => User | null),
  ) => void;
  setSelectedRole: (
    role: Role | null | ((prev: Role | null) => Role | null),
  ) => void;
  setQuery: (query: string) => void;
  setShowDeleted: (show: boolean) => void;
  setLoading: (loading: boolean) => void;
  setSaving: (saving: boolean) => void;
  setNotice: (notice: string) => void;
  setError: (error: string | ((prev: string) => string)) => void;
  setModal: (modal: ModalMode) => void;
  setForm: (
    form: CreateUserRequest | ((prev: CreateUserRequest) => CreateUserRequest),
  ) => void;
  setNewPassword: (password: string) => void;
  setRoleIDs: (ids: string[] | ((prev: string[]) => string[])) => void;
  setUserToDelete: (user: User | null) => void;
  setSelectedUserIds: (
    ids: string[] | ((prev: string[]) => string[]),
  ) => void;
  clearSelection: () => void;
  setBulkModal: (
    modal: "delete" | "restore" | "roles" | "info" | "email" | null,
  ) => void;
  reset: () => void;
}

const initialState = {
  users: [],
  roles: [],
  selectedUser: null,
  selectedRole: null,
  query: "",
  showDeleted: false,
  loading: true,
  saving: false,
  notice: "",
  error: "",
  modal: null as ModalMode,
  form: emptyCreate,
  newPassword: "",
  roleIDs: [],
  userToDelete: null,
  selectedUserIds: [] as string[],
  bulkModal: null as "delete" | "restore" | "roles" | "info" | "email" | null,
};

export const useUserAdminStore = create<UserAdminState>((set) => ({
  ...initialState,

  setUsers: (users) => set({ users }),
  setRoles: (roles) => set({ roles }),
  setSelectedUser: (selectedUser) =>
    set((state) => ({
      selectedUser:
        typeof selectedUser === "function"
          ? selectedUser(state.selectedUser)
          : selectedUser,
    })),
  setSelectedRole: (selectedRole) =>
    set((state) => ({
      selectedRole:
        typeof selectedRole === "function"
          ? selectedRole(state.selectedRole)
          : selectedRole,
    })),
  setQuery: (query) => set({ query }),
  setShowDeleted: (showDeleted) => set({ showDeleted }),
  setLoading: (loading) => set({ loading }),
  setSaving: (saving) => set({ saving }),
  setNotice: (notice) => set({ notice }),
  setError: (error) =>
    set((state) => ({
      error: typeof error === "function" ? error(state.error) : error,
    })),
  setModal: (modal) => set({ modal }),
  setForm: (form) =>
    set((state) => ({
      form: typeof form === "function" ? form(state.form) : form,
    })),
  setNewPassword: (newPassword) => set({ newPassword }),
  setRoleIDs: (roleIDs) =>
    set((state) => ({
      roleIDs: typeof roleIDs === "function" ? roleIDs(state.roleIDs) : roleIDs,
    })),
  setUserToDelete: (userToDelete) => set({ userToDelete }),
  setSelectedUserIds: (selectedUserIds) =>
    set((state) => ({
      selectedUserIds:
        typeof selectedUserIds === "function"
          ? selectedUserIds(state.selectedUserIds)
          : selectedUserIds,
    })),
  clearSelection: () => set({ selectedUserIds: [] }),
  setBulkModal: (bulkModal) => set({ bulkModal }),
  reset: () => set(initialState),
}));
