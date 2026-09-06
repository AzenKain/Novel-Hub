import { authService } from "@/services";
import type { User } from "@/types";
import { isBannedUser } from "@/utils/permission";
import { create } from "zustand";

type AuthStore = {
  user: User | null;
  booted: boolean;
  isLoginModalOpen: boolean;
  isRegisterModalOpen: boolean;
  isProfileModalOpen: boolean;

  setUser: (user: User | null) => void;
  setBooted: (booted: boolean) => void;
  setLoginModalOpen: (open: boolean) => void;
  setRegisterModalOpen: (open: boolean) => void;
  setProfileModalOpen: (open: boolean) => void;
  logout: () => Promise<void>;
  bootstrap: () => Promise<void>;
};

export const useAuthStore = create<AuthStore>((set) => ({
  user: null,
  booted: false,
  isLoginModalOpen: false,
  isRegisterModalOpen: false,
  isProfileModalOpen: false,

  setUser: (user) => set({ user }),
  setBooted: (booted) => set({ booted }),
  setLoginModalOpen: (open) => set({ isLoginModalOpen: open }),
  setRegisterModalOpen: (open) => set({ isRegisterModalOpen: open }),
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
      const user = me.data || null;
      if (user && isBannedUser(user)) {
        set({ user: null, booted: true });
        return;
      }
      set({ user, booted: true });
    } catch {
      set({ user: null, booted: true });
    }
  },
}));
