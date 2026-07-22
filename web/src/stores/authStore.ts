import { authService } from "@/services";
import type { User } from "@/types";
import { create } from "zustand";

type AuthStore = {
  user: User | null;
  booted: boolean;
  isLoginModalOpen: boolean;
  isProfileModalOpen: boolean;

  setUser: (user: User | null) => void;
  setBooted: (booted: boolean) => void;
  setLoginModalOpen: (open: boolean) => void;
  setProfileModalOpen: (open: boolean) => void;
  logout: () => Promise<void>;
  bootstrap: () => Promise<void>;
};

export const useAuthStore = create<AuthStore>((set) => ({
  user: null,
  booted: false,
  isLoginModalOpen: false,
  isProfileModalOpen: false,

  setUser: (user) => set({ user }),
  setBooted: (booted) => set({ booted: true }),
  setLoginModalOpen: (open) => set({ isLoginModalOpen: open }),
  setProfileModalOpen: (open) => set({ isProfileModalOpen: open }),

  logout: async () => {
    try {
      await authService.logout();
    } catch {}
    set({ user: null });
  },

  bootstrap: async () => {
    try {
      const me = await authService.me();
      set({ user: me.data || null, booted: true });
    } catch {
      set({ user: null, booted: true });
    }
  },
}));
