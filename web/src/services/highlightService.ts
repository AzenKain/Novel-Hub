import { api } from "@/config/api";
import type { Highlight } from "@/types";

export const highlightService = {
  async createHighlight(
    bookId: string,
    chapterId: string,
    textContent: string,
    startIndex: number,
    endIndex: number,
    color: string,
    note?: string
  ): Promise<Highlight> {
    const { data } = await api.post("/highlights", {
      bookId,
      chapterId,
      textContent,
      startIndex,
      endIndex,
      color,
      note,
    });
    return data.data;
  },

  async getHighlights(chapterId: string): Promise<Highlight[]> {
    const { data } = await api.get(`/highlights?chapterId=${chapterId}`);
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
