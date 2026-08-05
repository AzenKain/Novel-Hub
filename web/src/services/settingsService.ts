import { api } from "@/config/api";
import type {
  CommonResponse,
  OTPPurpose,
  OTPRequestResponse,
  OTPVerifyResponse,
  PublicSettings,
  RegisterRequest,
  ResetPasswordWithOTPRequest,
} from "@/types";
import axios from "axios";

export const settingsService = {
  async getPublic(): Promise<CommonResponse<PublicSettings>> {
    try {
      const res = await api.get("/settings/public");
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<PublicSettings>;
      throw error;
    }
  },

  async getSetupStatus(): Promise<CommonResponse<{ required: boolean }>> {
    try {
      const res = await api.get("/settings/setup-status");
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<{ required: boolean }>;
      throw error;
    }
  },

  async register(data: RegisterRequest): Promise<CommonResponse<unknown>> {
    try {
      const res = await api.post("/auth/register", data);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<unknown>;
      throw error;
    }
  },

  async requestOTP(email: string, purpose: OTPPurpose): Promise<CommonResponse<OTPRequestResponse>> {
    try {
      const res = await api.post("/auth/otp/request", { email, purpose });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<OTPRequestResponse>;
      throw error;
    }
  },

  async verifyOTP(email: string, purpose: OTPPurpose, code: string): Promise<CommonResponse<OTPVerifyResponse>> {
    try {
      const res = await api.post("/auth/otp/verify", { email, purpose, code });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<OTPVerifyResponse>;
      throw error;
    }
  },

  async resetPasswordWithOTP(data: ResetPasswordWithOTPRequest): Promise<CommonResponse<unknown>> {
    try {
      const res = await api.post("/auth/password/reset", data);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<unknown>;
      throw error;
    }
  },

  async submitSetup(data: Record<string, unknown>): Promise<CommonResponse<unknown>> {
    try {
      const res = await api.post("/setup", data);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<unknown>;
      throw error;
    }
  },

  async uploadSetupLogo(data: FormData): Promise<CommonResponse<{ url: string }>> {
    try {
      const res = await api.post("/setup/logo", data, {
        headers: {
          "Content-Type": "multipart/form-data"
        }
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response)
        return error.response.data as CommonResponse<{ url: string }>;
      throw error;
    }
  },
};
