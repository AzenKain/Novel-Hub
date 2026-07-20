import { authService } from "@/services";
import type { UpdateProfileRequest, User } from "@/types";
import { create } from "zustand";

type AuthStore = {
  user: User | null;
  booted: boolean;
  loading: boolean;
  error: string;
  isLoginModalOpen: boolean;
  isProfileModalOpen: boolean;
  setLoginModalOpen: (open: boolean) => void;
  setProfileModalOpen: (open: boolean) => void;
  login: (email: string, password: string) => Promise<void>;
  bootstrap: () => Promise<void>;
  logout: () => Promise<void>;
  updateProfile: (data: UpdateProfileRequest) => Promise<void>;
  clearError: () => void;
};

export const useAuthStore = create<AuthStore>((set) => ({
  user: null,
  booted: false,
  loading: false,
  error: "",
  isLoginModalOpen: false,
  isProfileModalOpen: false,

  setLoginModalOpen: (open) => set({ isLoginModalOpen: open }),
  setProfileModalOpen: (open) => set({ isProfileModalOpen: open }),

  clearError: () => set({ error: "" }),

  login: async (email, password) => {
    set({ loading: true, error: "" });
    try {
      const res = await authService.signin(email, password);
      if (!res.status) throw new Error(res.message || "Invalid credentials");
      const me = await authService.me();
      if (!me.status) throw new Error(me.message || "Failed to load user profile");
      set({ user: me.data || null, loading: false, booted: true });
    } catch (error) {
      set({ error: error instanceof Error ? error.message : String(error), loading: false });
      throw error;
    }
  },

  bootstrap: async () => {
    set({ loading: true });
    try {
      const me = await authService.me();
      set({ user: me.data || null, booted: true, loading: false });
    } catch {
      set({ user: null, booted: true, loading: false });
    }
  },

  logout: async () => {
    set({ loading: true });
    try {
      await authService.logout();
    } catch (error) {
      console.warn("Logout error", error);
    }
    set({ user: null, loading: false });
  },

  updateProfile: async (data) => {
    set({ loading: true, error: "" });
    try {
      const res = await authService.updateProfile(data);
      if (res.status && res.data) {
        set({ user: res.data, loading: false });
      } else {
        throw new Error(res.message || "Failed to update profile");
      }
    } catch (error) {
      set({ error: error instanceof Error ? error.message : String(error), loading: false });
      throw error;
    }
  }
}));

