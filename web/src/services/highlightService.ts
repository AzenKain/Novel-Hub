import { api } from "@/config/api";
import type { Highlight } from "@/types";

export const highlightService = {
  async createHighlight(
    book_id: string,
    chapter_id: string,
    text_content: string,
    start_index: number,
    end_index: number,
    color: string,
    note?: string
  ): Promise<Highlight> {
    const { data } = await api.post("/highlights", {
      book_id,
      chapter_id,
      text_content,
      start_index,
      end_index,
      color,
      note,
    });
    return data.data;
  },

  async getHighlights(chapter_id: string): Promise<Highlight[]> {
    const { data } = await api.get(`/highlights?chapter_id=${chapter_id}`);
    return data.data;
  },

  async updateHighlightNote(
    id: string,
    color: string,
    note?: string
  ): Promise<Highlight> {
    const { data } = await api.put(`/highlights/${id}`, { color, note });
    return data.data;
  },

  async deleteHighlight(id: string): Promise<void> {
    await api.delete(`/highlights/${id}`);
  },
};
