import { offlineStore } from "@/lib/offlineStore";
import { authService, settingsService } from "@/services";
import { useAuthStore } from "@/stores";
import type {
  ChangePasswordRequest,
  OTPPurpose,
  RegisterRequest,
  ResetPasswordWithOTPRequest,
  UpdateProfileRequest,
  User,
} from "@/types";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

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
    mutationFn: async ({ email, password, totpCode }: { email: string; password: string; totpCode?: string }) => {
      const res = await authService.signin(email, password, totpCode);
      if (!res.status) throw new Error(res.message || "Invalid credentials");
      if (res.data?.totp_required) return null;
      const me = await authService.me();
      if (!me.status) throw new Error(me.message || "Failed to load user profile");
      return me.data || null;
    },
    onSuccess: (user) => {
      if (!user) return;
      setUser(user);
      setLoginModalOpen(false);
      void queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
    },
  });
}

export function useLoginFlow() {
  const loginMutation = useLoginMutation();
  const [needsCode, setNeedsCode] = useState(false);

  const submit = (email: string, password: string, totpCode?: string) =>
    loginMutation.mutate(
      { email, password, totpCode },
      { onSuccess: (user) => setNeedsCode(!user) },
    );

  return {
    mutation: loginMutation,
    needsCode,
    resetCode: () => setNeedsCode(false),
    submit,
  };
}

export function useLogoutMutation() {
  const setUser = useAuthStore((state) => state.setUser);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      await authService.logout();
      await offlineStore.clearAll().catch(() => undefined);
    },
    onSuccess: () => {
      setUser(null);
      queryClient.clear();
    },
  });
}

// Đổi mật khẩu bump token_version ở BE nên mọi session bị thu hồi -> phải đăng nhập lại.
export function useChangePasswordMutation() {
  const setUser = useAuthStore((state) => state.setUser);
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: ChangePasswordRequest) => {
      const res = await authService.changePassword(data);
      if (!res.status) throw new Error(res.message || "Failed to change password");
      return res;
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

export function useRequestOTPMutation() {
  return useMutation({
    mutationFn: async ({ email, purpose }: { email: string; purpose: OTPPurpose }) => {
      const res = await settingsService.requestOTP(email, purpose);
      if (!res.status || !res.data) throw new Error(res.message || "Failed to send the code");
      return res.data;
    },
  });
}

export function useVerifyOTPMutation() {
  return useMutation({
    mutationFn: async ({ email, purpose, code }: { email: string; purpose: OTPPurpose; code: string }) => {
      const res = await settingsService.verifyOTP(email, purpose, code);
      if (!res.status || !res.data) throw new Error(res.message || "The code is invalid or has expired");
      return res.data;
    },
  });
}

export function useRegisterMutation() {
  return useMutation({
    mutationFn: async (data: RegisterRequest) => {
      const res = await settingsService.register(data);
      if (!res.status) throw new Error(res.message || "Registration failed");
      return res;
    },
  });
}

export function useResetPasswordWithOTPMutation() {
  return useMutation({
    mutationFn: async (data: ResetPasswordWithOTPRequest) => {
      const res = await settingsService.resetPasswordWithOTP(data);
      if (!res.status) throw new Error(res.message || "Failed to reset the password");
      return res;
    },
  });
}

export function useTOTPStatusQuery(enabled = true) {
  return useQuery({
    queryKey: ["auth", "totp"],
    queryFn: async () => {
      const res = await authService.totpStatus();
      if (!res.status || !res.data) throw new Error(res.message || "Failed to load two-factor status");
      return res.data;
    },
    enabled,
    retry: false,
  });
}

export function useTOTPEnrollMutation() {
  return useMutation({
    mutationFn: async () => {
      const res = await authService.totpEnroll();
      if (!res.status || !res.data) throw new Error(res.message || "Failed to start the setup");
      return res.data;
    },
  });
}

export function useTOTPConfirmMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (code: string) => {
      const res = await authService.totpConfirm(code);
      if (!res.status || !res.data) throw new Error(res.message || "The code is invalid or has expired");
      return res.data;
    },
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["auth", "totp"] }),
  });
}

export function useTOTPDisableMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (code: string) => {
      const res = await authService.totpDisable(code);
      if (!res.status) throw new Error(res.message || "The code is invalid or has expired");
      return res;
    },
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["auth", "totp"] }),
  });
}

export function useTOTPRecoveryCodesMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (code: string) => {
      const res = await authService.totpRecoveryCodes(code);
      if (!res.status || !res.data) throw new Error(res.message || "The code is invalid or has expired");
      return res.data;
    },
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["auth", "totp"] }),
  });
}
