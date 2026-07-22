import { authService } from "@/services";
import { useAuthStore } from "@/stores";
import type { UpdateProfileRequest, User } from "@/types";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export function useCurrentUserQuery() {
  const setUser = useAuthStore((state) => state.setUser);
  const setBooted = useAuthStore((state) => state.setBooted);

  return useQuery<User | null>({
    queryKey: ["auth", "me"],
    queryFn: async () => {
      try {
        const me = await authService.me();
        const user = me.data || null;
        setUser(user);
        setBooted(true);
        return user;
      } catch {
        setUser(null);
        setBooted(true);
        return null;
      }
    },
    staleTime: 1000 * 60 * 5,
    retry: false,
  });
}

export function useLoginMutation() {
  const setUser = useAuthStore((state) => state.setUser);
  const setLoginModalOpen = useAuthStore((state) => state.setLoginModalOpen);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ email, password }: { email: string; password: string }) => {
      const res = await authService.signin(email, password);
      if (!res.status) throw new Error(res.message || "Invalid credentials");
      const me = await authService.me();
      if (!me.status) throw new Error(me.message || "Failed to load user profile");
      return me.data || null;
    },
    onSuccess: (user) => {
      setUser(user);
      setLoginModalOpen(false);
      void queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
    },
  });
}

export function useLogoutMutation() {
  const setUser = useAuthStore((state) => state.setUser);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      await authService.logout();
    },
    onSuccess: () => {
      setUser(null);
      queryClient.clear();
    },
  });
}

export function useUpdateProfileMutation() {
  const setUser = useAuthStore((state) => state.setUser);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: UpdateProfileRequest) => {
      const res = await authService.updateProfile(data);
      if (!res.status || !res.data) throw new Error(res.message || "Failed to update profile");
      return res.data;
    },
    onSuccess: (updatedUser) => {
      setUser(updatedUser);
      void queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
    },
  });
}
