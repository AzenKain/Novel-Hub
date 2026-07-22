import { api } from "@/config/api";
import type { BootstrapResponse, CommonResponse, SearchSnippet } from "@/types";
import axios from "axios";

export const readerService = {
  async getBootstrap(bookId: string, fileId?: string): Promise<CommonResponse<BootstrapResponse>> {
    const query = fileId ? `?file_id=${encodeURIComponent(fileId)}` : "";
    try {
      const res = await api.get(`/reader/${bookId}/bootstrap${query}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<BootstrapResponse>;
      throw error;
    }
  },

  async getChapterHtml(bookId: string, chapterId: string, fileId?: string): Promise<string> {
    const query = fileId ? `?file_id=${encodeURIComponent(fileId)}` : "";
    try {
      const res = await api.get(`/reader/${bookId}/chapter/${encodeURIComponent(chapterId)}${query}`, {
        responseType: 'text'
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as string;
      throw error;
    }
  },

  async searchInBook(bookId: string, query: string): Promise<CommonResponse<SearchSnippet[]>> {
    try {
      const res = await api.get(`/books/${bookId}/search`, {
        params: { q: query },
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) return error.response.data as CommonResponse<SearchSnippet[]>;
      throw error;
    }
  },

  async syncReadingSession(bookId: string, duration: number, words: number): Promise<void> {
    await api.post('/reader/stats/session', {
      bookId,
      duration,
      words,
    });
  },

  async getReadingHeatmap(): Promise<any> {
    const { data } = await api.get('/reader/stats/heatmap');
    return data.data;
  }
};
