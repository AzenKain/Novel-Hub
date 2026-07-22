import { api } from "@/config/api";
import type { CommonResponse } from "@/types";
import axios from "axios";

export const trackerService = {
  async connectTracker(
    provider: string,
    accessToken: string
  ): Promise<CommonResponse<void>> {
    try {
      const res = await api.post("/trackers/connect", {
        provider,
        access_token: accessToken,
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<void>;
      }
      throw error;
    }
  },

  async mapBookTracker(
    bookId: number,
    provider: string,
    externalSeriesId: string
  ): Promise<CommonResponse<void>> {
    try {
      const res = await api.post("/trackers/map", {
        book_id: bookId,
        provider,
        external_series_id: externalSeriesId,
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<void>;
      }
      throw error;
    }
  },

  async searchAniList(
    title: string
  ): Promise<CommonResponse<{ provider: string; external_series_id: string }>> {
    try {
      const res = await api.get(`/trackers/search?title=${encodeURIComponent(title)}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<{ provider: string; external_series_id: string }>;
      }
      throw error;
    }
  },

  async syncProgress(
    bookId: number,
    title: string,
    progress: number
  ): Promise<CommonResponse<void>> {
    try {
      const res = await api.post("/trackers/sync", {
        book_id: bookId,
        title,
        progress,
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<void>;
      }
      throw error;
    }
  },
};
