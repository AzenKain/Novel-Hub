import { api } from "@/config/api";
import type { CommonResponse, Podcast, PodcastEpisode, SubscribePodcastInput, UpdatePodcastInput } from "@/types";
import axios from "axios";

export const podcastService = {
  async listPodcasts(): Promise<CommonResponse<Podcast[]>> {
    try {
      const res = await api.get("/podcasts");
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<Podcast[]>;
      }
      throw error;
    }
  },

  async subscribe(input: SubscribePodcastInput): Promise<CommonResponse<Podcast>> {
    try {
      const res = await api.post("/podcasts", input);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<Podcast>;
      }
      throw error;
    }
  },

  async updatePodcast(id: string, input: UpdatePodcastInput): Promise<CommonResponse<Podcast>> {
    try {
      const res = await api.put(`/podcasts/${id}`, input);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<Podcast>;
      }
      throw error;
    }
  },

  async deletePodcast(id: string): Promise<CommonResponse<void>> {
    try {
      const res = await api.delete(`/podcasts/${id}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<void>;
      }
      throw error;
    }
  },

  async listEpisodes(podcastId: string): Promise<CommonResponse<PodcastEpisode[]>> {
    try {
      const res = await api.get(`/podcasts/${podcastId}/episodes`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<PodcastEpisode[]>;
      }
      throw error;
    }
  },

  async refreshPodcast(id: string): Promise<CommonResponse<{ job_id: string }>> {
    try {
      const res = await api.post(`/podcasts/${id}/refresh`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<{ job_id: string }>;
      }
      throw error;
    }
  },

  async downloadEpisode(podcastId: string, episodeId: string): Promise<CommonResponse<{ job_id: string }>> {
    try {
      const res = await api.post(`/podcasts/${podcastId}/episodes/${episodeId}/download`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<{ job_id: string }>;
      }
      throw error;
    }
  },
};