import { create } from "zustand";
import { persist } from "zustand/middleware";

export interface GuestBookmark {
  bookId: string;
  createdAt: string;
}

export interface GuestProgress {
  bookId: string;
  chapterId: string;
  progressPercent: number;
  updatedAt: string;
}

interface GuestState {
  bookmarks: GuestBookmark[];
  progressMap: Record<string, GuestProgress>;

  addBookmark: (bookId: string) => void;
  removeBookmark: (bookId: string) => void;
  isBookmarked: (bookId: string) => boolean;
  saveProgress: (bookId: string, chapterId: string, progressPercent: number) => void;
  getProgress: (bookId: string) => GuestProgress | null;
  clearGuestData: () => void;
}

export const useGuestStore = create<GuestState>()(
  persist(
    (set, get) => ({
      bookmarks: [],
      progressMap: {},

      addBookmark: (bookId) => {
        const { bookmarks } = get();
        if (!bookmarks.some((b) => b.bookId === bookId)) {
          set({ bookmarks: [...bookmarks, { bookId, createdAt: new Date().toISOString() }] });
        }
      },

      removeBookmark: (bookId) => {
        set({ bookmarks: get().bookmarks.filter((b) => b.bookId !== bookId) });
      },

      isBookmarked: (bookId) => {
        return get().bookmarks.some((b) => b.bookId === bookId);
      },

      saveProgress: (bookId, chapterId, progressPercent) => {
        set({
          progressMap: {
            ...get().progressMap,
            [bookId]: { bookId, chapterId, progressPercent, updatedAt: new Date().toISOString() },
          },
        });
      },

      getProgress: (bookId) => {
        return get().progressMap[bookId] || null;
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
