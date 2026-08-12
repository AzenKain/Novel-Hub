import { api } from "@/config/api";
import { offlineStore } from "@/lib/offlineStore";
import type { BootstrapResponse, CommonResponse, LibraryBreakdown, ReadingETA, ReadingGoal, ReadingStatsSummary, SearchSnippet } from "@/types";
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
    const payload = { book_id, duration, words };
    if (typeof navigator !== "undefined" && !navigator.onLine) {
      void offlineStore.enqueueSyncItem({ type: "session", payload });
      return { status: true, message: "Session queued offline" };
    }
    try {
      const res = await api.post('/reader/stats/session', {
        book_id,
        duration,
        words,
        session_date: new Date().toLocaleDateString('sv-SE'),
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error)) {
        if (!error.response) {
          void offlineStore.enqueueSyncItem({ type: "session", payload });
          return { status: true, message: "Session queued offline" };
        }
        return error.response.data as CommonResponse<void>;
      }
      throw error;
    }
  },

  async getReadingHeatmap(): Promise<CommonResponse<Record<string, { duration: number; words: number }>>> {
    try {
      const res = await api.get('/reader/stats/heatmap');
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<any>;
      throw error;
    }
  },

  async getReadingStatsSummary(): Promise<CommonResponse<ReadingStatsSummary>> {
    try {
      const res = await api.get('/reader/stats/summary');
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<any>;
      throw error;
    }
  },

  async getReaderETA(book_id: string): Promise<CommonResponse<ReadingETA>> {
    try {
      const res = await api.get(`/reader/stats/eta/${encodeURIComponent(book_id)}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<any>;
      throw error;
    }
  },

  async getLibraryBreakdown(): Promise<CommonResponse<LibraryBreakdown>> {
    try {
      const res = await api.get('/library/stats/breakdown');
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
