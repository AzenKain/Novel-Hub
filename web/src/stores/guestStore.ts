import { create } from "zustand";
import { persist } from "zustand/middleware";

export interface GuestBookmark {
  book_id: string;
  created_at: string;
}

export interface GuestProgress {
  book_id: string;
  chapter_id: string;
  progress_percent: number;
  updated_at: string;
}

interface GuestState {
  bookmarks: GuestBookmark[];
  progressMap: Record<string, GuestProgress>;

  addBookmark: (book_id: string) => void;
  removeBookmark: (book_id: string) => void;
  isBookmarked: (book_id: string) => boolean;
  saveProgress: (book_id: string, chapter_id: string, progress_percent: number) => void;
  getProgress: (book_id: string) => GuestProgress | null;
  clearGuestData: () => void;
}

export const useGuestStore = create<GuestState>()(
  persist(
    (set, get) => ({
      bookmarks: [],
      progressMap: {},

      addBookmark: (book_id) => {
        const { bookmarks } = get();
        if (!bookmarks.some((b) => b.book_id === book_id)) {
          set({ bookmarks: [...bookmarks, { book_id, created_at: new Date().toISOString() }] });
        }
      },

      removeBookmark: (book_id) => {
        set({ bookmarks: get().bookmarks.filter((b) => b.book_id !== book_id) });
      },

      isBookmarked: (book_id) => {
        return get().bookmarks.some((b) => b.book_id === book_id);
      },

      saveProgress: (book_id, chapter_id, progress_percent) => {
        set({
          progressMap: {
            ...get().progressMap,
            [book_id]: { book_id, chapter_id, progress_percent, updated_at: new Date().toISOString() },
          },
        });
      },

      getProgress: (book_id) => {
        return get().progressMap[book_id] || null;
      },

      clearGuestData: () => {
        set({ bookmarks: [], progressMap: {} });
      },
    }),
    {
      name: "novelhub_guest_storage",
    }
  )
);
