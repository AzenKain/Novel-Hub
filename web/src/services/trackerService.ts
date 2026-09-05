import { api } from "@/config/api";
import type {
  CommonResponse,
  TrackerConnection,
  TrackerSearchResult,
} from "@/types";
import axios from "axios";

export const trackerService = {
  async getConnections(): Promise<CommonResponse<TrackerConnection[]>> {
    try {
      const res = await api.get("/trackers/connections");
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<TrackerConnection[]>;
      }
      throw error;
    }
  },

  async connectTracker(
    provider: string,
    accessToken: string,
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
    externalSeriesId: string,
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

  async searchAniList(
    title: string,
  ): Promise<CommonResponse<TrackerSearchResult>> {
    try {
      const res = await api.get(
        `/trackers/search?title=${encodeURIComponent(title)}`,
      );
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
    progress?: number,
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

  async exportHighlightsToReadwise(
    book_id: string,
  ): Promise<CommonResponse<{ exported: number }>> {
    try {
      const res = await api.post("/trackers/readwise/export", { book_id });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<{ exported: number }>;
      }
      throw error;
    }
  },

  async connectHardcover(): Promise<CommonResponse<{ authorize_url: string }>> {
    try {
      const res = await api.post("/scrobble/hardcover/connect");
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<{ authorize_url: string }>;
      }
      throw error;
    }
  },

  async syncHardcoverProgress(
    book_id: string,
    progress: number,
  ): Promise<CommonResponse<void>> {
    try {
      const res = await api.post("/scrobble/hardcover/sync", {
        book_id,
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
