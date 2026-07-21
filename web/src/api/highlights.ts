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
  const { data } = await api.post("/features/highlights", {
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
  const { data } = await api.get(`/features/highlights?chapterId=${chapterId}`);
  return data.data;
};

export const updateHighlightNote = async (
  id: string,
  color: string,
  note?: string
): Promise<Highlight> => {
  const { data } = await api.put(`/features/highlights/${id}`, { color, note });
  return data.data;
};

export const deleteHighlight = async (id: string): Promise<void> => {
  await api.delete(`/features/highlights/${id}`);
};
