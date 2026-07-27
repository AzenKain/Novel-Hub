import { api } from "@/config/api";
import type { AuthResponse, ChangePasswordRequest, CommonResponse, UpdateProfileRequest, User } from "@/types";
import axios from "axios";

export const authService = {
  async signin(email: string, password: string): Promise<CommonResponse<AuthResponse>> {
    try {
      const response = await api.post("/auth/signin", { email, password });
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

  async updateProfile(data: UpdateProfileRequest): Promise<CommonResponse<User>> {
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

  async changePassword(data: ChangePasswordRequest): Promise<CommonResponse<void>> {
    try {
      const response = await api.patch("/users/current/password", data);
      return response.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<void>;
      }
      throw error;
    }
  }
};
