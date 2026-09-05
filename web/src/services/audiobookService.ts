import { api } from "@/config/api";
import type {
  AudiobookChapter,
  CommonResponse,
  LookupAudiobookChaptersInput,
  MergeAudioInput,
  UpsertAudiobookChapterInput,
} from "@/types";
import axios from "axios";

export const audiobookService = {
  async listChapters(
    book_id: string,
  ): Promise<CommonResponse<AudiobookChapter[]>> {
    try {
      const res = await api.get(`/books/${book_id}/audiobook/chapters`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<AudiobookChapter[]>;
      }
      throw error;
    }
  },

  async upsertChapter(
    book_id: string,
    id: string | undefined,
    input: UpsertAudiobookChapterInput,
  ): Promise<CommonResponse<AudiobookChapter>> {
    try {
      const res = id
        ? await api.put(`/books/${book_id}/audiobook/chapters/${id}`, input)
        : await api.post(`/books/${book_id}/audiobook/chapters`, input);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<AudiobookChapter>;
      }
      throw error;
    }
  },

  async deleteChapter(
    book_id: string,
    id: string,
  ): Promise<CommonResponse<void>> {
    try {
      const res = await api.delete(
        `/books/${book_id}/audiobook/chapters/${id}`,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<void>;
      }
      throw error;
    }
  },

  async lookupChapters(
    book_id: string,
    input: LookupAudiobookChaptersInput,
  ): Promise<CommonResponse<AudiobookChapter[]>> {
    try {
      const res = await api.post(
        `/books/${book_id}/audiobook/chapters/lookup`,
        input,
      );
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<AudiobookChapter[]>;
      }
      throw error;
    }
  },

  async mergeAudio(
    book_id: string,
    input: MergeAudioInput,
  ): Promise<CommonResponse<{ job_id: string }>> {
    try {
      const res = await api.post(`/books/${book_id}/merge-audio`, input);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<{ job_id: string }>;
      }
      throw error;
    }
  },
};
