import { api } from "@/config/api";
import type { CommonResponse, KoboSetup } from "@/types";
import axios from "axios";

function toCommonResponse<T>(error: unknown): CommonResponse<T> {
  if (axios.isAxiosError(error) && error.response) {
    return error.response.data as CommonResponse<T>;
  }
  throw error;
}

export const koboService = {
  async getSetup(): Promise<CommonResponse<KoboSetup>> {
    try {
      const res = await api.get("/kobo/setup");
      return res.data;
    } catch (error) {
      return toCommonResponse<KoboSetup>(error);
    }
  },

  async regenerate(): Promise<CommonResponse<KoboSetup>> {
    try {
      const res = await api.post("/kobo/setup/regenerate");
      return res.data;
    } catch (error) {
      return toCommonResponse<KoboSetup>(error);
    }
  },

  async revoke(): Promise<CommonResponse<void>> {
    try {
      const res = await api.delete("/kobo/setup");
      return res.data;
    } catch (error) {
      return toCommonResponse<void>(error);
    }
  },
};
