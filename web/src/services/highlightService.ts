import { api } from "@/config/api";
import type { CommonResponse, Highlight } from "@/types";
import axios from "axios";

export const highlightService = {
  async createHighlight(
    book_id: string,
    chapter_id: string,
    text_content: string,
    start_index: number,
    end_index: number,
    color: string,
    note?: string,
    cfi_range?: string
  ): Promise<CommonResponse<Highlight>> {
    try {
      const res = await api.post("/highlights", {
        book_id,
        chapter_id,
        text_content,
        start_index,
        end_index,
        color,
        note,
        cfi_range,
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<Highlight>;
      }
      throw error;
    }
  },

  async getHighlights(chapter_id?: string, book_id?: string): Promise<CommonResponse<Highlight[]>> {
    try {
      const params = new URLSearchParams();
      if (chapter_id) params.append("chapter_id", chapter_id);
      if (book_id) params.append("book_id", book_id);
      const res = await api.get(`/highlights?${params.toString()}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<Highlight[]>;
      }
      throw error;
    }
  },

  async updateHighlightNote(
    id: string,
    color: string,
    note?: string
  ): Promise<CommonResponse<Highlight>> {
    try {
      const res = await api.put(`/highlights/${id}`, { color, note });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<Highlight>;
      }
      throw error;
    }
  },

  async deleteHighlight(id: string): Promise<CommonResponse<void>> {
    try {
      const res = await api.delete(`/highlights/${id}`);
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<void>;
      }
      throw error;
    }
  },

  // Downloads the Markdown export as a blob so a backend error (e.g. no
  // highlights) surfaces as a toast instead of navigating to a JSON page.
  async exportMarkdown(book_id: string): Promise<Blob> {
    const res = await api.get(`/highlights/${encodeURIComponent(book_id)}/export.md`, {
      responseType: "blob",
    });
    return res.data as Blob;
  },

  async extractErrorMessage(error: unknown, fallback: string): Promise<string> {
    if (axios.isAxiosError(error) && error.response?.data instanceof Blob) {
      try {
        const parsed = JSON.parse(await error.response.data.text()) as CommonResponse<unknown>;
        if (parsed?.message) return parsed.message;
      } catch {
        // not a JSON error envelope, fall through to fallback
      }
    }
    if (error instanceof Error && error.message) return error.message;
    return fallback;
  },
};
