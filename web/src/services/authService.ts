import { api } from "@/config/api";
import type {
  AuthResponse,
  ChangePasswordRequest,
  CommonResponse,
  TOTPEnrollment,
  TOTPRecoveryCodes,
  TOTPStatus,
  UpdateProfileRequest,
  User,
} from "@/types";
import axios from "axios";

export const authService = {
  async signin(
    email: string,
    password: string,
    totpCode?: string,
  ): Promise<CommonResponse<AuthResponse>> {
    try {
      const response = await api.post("/auth/signin", {
        email,
        password,
        ...(totpCode ? { totp_code: totpCode } : {}),
      });
      return response.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<AuthResponse>;
      }
      throw error;
    }
  },

  async me(): Promise<CommonResponse<User>> {
    try {
      const response = await api.get("/users/current");
      return response.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<User>;
      }
      throw error;
    }
  },

  async logout(): Promise<CommonResponse<unknown>> {
    try {
      const response = await api.post("/auth/logout");
      return response.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<unknown>;
      }
      throw error;
    }
  },

  async updateProfile(
    data: UpdateProfileRequest,
  ): Promise<CommonResponse<User>> {
    try {
      const response = await api.put("/users/current", data);
      return response.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<User>;
      }
      throw error;
    }
  },

  async uploadAvatar(
    file: File | Blob,
  ): Promise<CommonResponse<{ url: string }>> {
    try {
      const formData = new FormData();
      formData.append("file", file);
      const response = await api.post("/users/current/avatar", formData);
      return response.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<{ url: string }>;
      }
      throw error;
    }
  },

  async changePassword(
    data: ChangePasswordRequest,
  ): Promise<CommonResponse<void>> {
    try {
      const response = await api.patch("/users/current/password", data);
      return response.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<void>;
      }
      throw error;
    }
  },

  async totpStatus(): Promise<CommonResponse<TOTPStatus>> {
    const response = await api.get("/auth/totp");
    return response.data;
  },

  async totpEnroll(): Promise<CommonResponse<TOTPEnrollment>> {
    const response = await api.post("/auth/totp/enroll", {});
    return response.data;
  },

  async totpConfirm(code: string): Promise<CommonResponse<TOTPRecoveryCodes>> {
    const response = await api.post("/auth/totp/confirm", { code });
    return response.data;
  },

  async totpDisable(code: string): Promise<CommonResponse<void>> {
    const response = await api.post("/auth/totp/disable", { code });
    return response.data;
  },

  async totpRecoveryCodes(
    code: string,
  ): Promise<CommonResponse<TOTPRecoveryCodes>> {
    const response = await api.post("/auth/totp/recovery-codes", { code });
    return response.data;
  },
};
