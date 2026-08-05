import { api, toQuery } from "@/config/api";
import type { CommonResponse, CreateUserDeviceInput, UserDevice } from "@/types";
import axios from "axios";

export const deviceService = {
  async listDevices(params?: { cursor?: string; limit?: number }): Promise<CommonResponse<UserDevice[]>> {
    try {
      const res = await api.get(`/user/devices${toQuery(params || {})}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<UserDevice[]>;
      }
      throw error;
    }
  },
  async createDevice(input: CreateUserDeviceInput): Promise<CommonResponse<UserDevice>> {
    try {
      const res = await api.post("/user/devices", input);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<UserDevice>;
      }
      throw error;
    }
  },
  async deleteDevice(id: string): Promise<CommonResponse<unknown>> {
    try {
      const res = await api.delete(`/user/devices/${encodeURIComponent(id)}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<unknown>;
      }
      throw error;
    }
  },
  async pushBook(bookId: string, deviceId: string): Promise<CommonResponse<unknown>> {
    try {
      const res = await api.post(`/books/${encodeURIComponent(bookId)}/push`, { device_id: deviceId });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<unknown>;
      }
      throw error;
    }
  },
};
