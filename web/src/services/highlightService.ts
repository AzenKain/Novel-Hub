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
    note?: string
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
      });
      return res.data;
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        return error.response.data as CommonResponse<Highlight>;
      }
      throw error;
    }
  },

  async getHighlights(chapter_id: string): Promise<CommonResponse<Highlight[]>> {
    try {
      const res = await api.get(`/highlights?chapter_id=${chapter_id}`);
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
};
