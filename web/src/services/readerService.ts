import { api } from "@/config/api";
import type { BootstrapResponse, CommonResponse, ReadingGoal, SearchSnippet } from "@/types";
import axios from "axios";

export const readerService = {
  async getBootstrap(book_id: string, file_id?: string): Promise<CommonResponse<BootstrapResponse>> {
    const query = file_id ? `?file_id=${encodeURIComponent(file_id)}` : "";
    try {
      const res = await api.get(`/reader/${book_id}/bootstrap${query}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<BootstrapResponse>;
      throw error;
    }
  },

  async getChapterHtml(book_id: string, chapter_id: string, file_id?: string): Promise<string> {
    const query = file_id ? `?file_id=${encodeURIComponent(file_id)}` : "";
    try {
      const res = await api.get(`/reader/${book_id}/chapter/${encodeURIComponent(chapter_id)}${query}`, {
        responseType: 'text'
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as string;
      throw error;
    }
  },

  async searchInBook(book_id: string, query: string): Promise<CommonResponse<SearchSnippet[]>> {
    try {
      const res = await api.get(`/books/${book_id}/search`, {
        params: { q: query },
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<SearchSnippet[]>;
      throw error;
    }
  },

  async syncReadingSession(book_id: string, duration: number, words: number): Promise<CommonResponse<void>> {
    try {
      const res = await api.post('/reader/stats/session', {
        book_id,
        duration,
        words,
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<void>;
      throw error;
    }
  },

  async getReadingHeatmap(): Promise<CommonResponse<any>> {
    try {
      const res = await api.get('/reader/stats/heatmap');
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<any>;
      throw error;
    }
  },

  async getReadingGoal(): Promise<CommonResponse<ReadingGoal>> {
    try {
      const res = await api.get('/reader/goals/');
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<ReadingGoal>;
      throw error;
    }
  },

  async upsertReadingGoal(target_words_per_day: number, target_books_per_year: number): Promise<CommonResponse<ReadingGoal>> {
    try {
      const res = await api.put('/reader/goals/', { target_words_per_day, target_books_per_year });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<ReadingGoal>;
      throw error;
    }
  }
};
