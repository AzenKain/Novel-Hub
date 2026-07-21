import { api } from "@/config/api";

export interface Highlight {
  id: string;
  userId: number;
  bookId: string;
  chapterId: string;
  textContent: string;
  startIndex: number;
  endIndex: number;
  color: string;
  note?: string;
  createdAt: string;
  updatedAt: string;
}

export const createHighlight = async (
  bookId: string,
  chapterId: string,
  textContent: string,
  startIndex: number,
  endIndex: number,
  color: string,
  note?: string
): Promise<Highlight> => {
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
};

export const getHighlights = async (chapterId: string): Promise<Highlight[]> => {
  const { data } = await api.get(`/highlights?chapterId=${chapterId}`);
  return data.data;
};

export const updateHighlightNote = async (
  id: string,
  color: string,
  note?: string
): Promise<Highlight> => {
  const { data } = await api.put(`/highlights/${id}`, { color, note });
  return data.data;
};

export const deleteHighlight = async (id: string): Promise<void> => {
  await api.delete(`/highlights/${id}`);
};
