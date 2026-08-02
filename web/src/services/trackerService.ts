import { api } from "@/config/api";
import type { CommonResponse, TrackerSearchResult } from "@/types";
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
    book_id: string,
    provider: string,
    externalSeriesId: string
  ): Promise<CommonResponse<void>> {
    try {
      const res = await api.post("/trackers/map", {
        book_id: book_id,
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

  async searchAniList(title: string): Promise<CommonResponse<TrackerSearchResult>> {
    try {
      const res = await api.get(`/trackers/search?title=${encodeURIComponent(title)}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<TrackerSearchResult>;
      }
      throw error;
    }
  },

  async syncProgress(
    book_id: string,
    title: string,
    progress?: number
  ): Promise<CommonResponse<void>> {
    try {
      const body: { book_id: string; title: string; progress?: number } = {
        book_id: book_id,
        title,
      };
      if (progress !== undefined) {
        body.progress = progress;
      }
      const res = await api.post("/trackers/sync", body);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<void>;
      }
      throw error;
    }
  },
};
