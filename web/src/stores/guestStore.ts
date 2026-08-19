import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { featureService as FeatureServiceType } from "@/services";

export interface GuestBookmark {
  book_id: string;
  created_at: string;
}

export interface GuestProgress {
  book_id: string;
  file_id?: string;
  chapter_id: string;
  chapter_title?: string;
  chapter_index?: number;
  progress_percent: number;
  book_title?: string;
  author_name?: string;
  cover_url?: string;
  updated_at: string;
}

export type GuestProgressInput = Omit<GuestProgress, "updated_at"> & {
  updated_at?: string;
};

interface GuestState {
  bookmarks: GuestBookmark[];
  progressMap: Record<string, GuestProgress>;

  addBookmark: (book_id: string) => void;
  removeBookmark: (book_id: string) => void;
  toggleBookmark: (book_id: string) => boolean;
  isBookmarked: (book_id: string) => boolean;
  getBookmarks: () => string[];

  saveProgress: (entry: GuestProgressInput) => void;
  recordReading: (entry: GuestProgressInput) => void;
  getProgress: (book_id: string) => GuestProgress | null;
  getReadingHistory: () => GuestProgress[];

  syncToServer: (featureService: typeof FeatureServiceType) => Promise<void>;
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
          set({
            bookmarks: [...bookmarks, { book_id, created_at: new Date().toISOString() }],
          });
        }
      },

      removeBookmark: (book_id) => {
        set({
          bookmarks: get().bookmarks.filter((b) => b.book_id !== book_id),
        });
      },

      toggleBookmark: (book_id) => {
        const { bookmarks } = get();
        const exists = bookmarks.some((b) => b.book_id === book_id);
        if (exists) {
          set({ bookmarks: bookmarks.filter((b) => b.book_id !== book_id) });
          return false;
        } else {
          set({
            bookmarks: [...bookmarks, { book_id, created_at: new Date().toISOString() }],
          });
          return true;
        }
      },

      isBookmarked: (book_id) => {
        return get().bookmarks.some((b) => b.book_id === book_id);
      },

      getBookmarks: () => {
        return get().bookmarks.map((b) => b.book_id);
      },

      saveProgress: (entry) => {
        const now = entry.updated_at || new Date().toISOString();
        const record: GuestProgress = {
          ...entry,
          updated_at: now,
        };
        set({
          progressMap: {
            ...get().progressMap,
            [entry.book_id]: record,
          },
        });
      },

      recordReading: (entry) => {
        get().saveProgress(entry);
      },

      getProgress: (book_id) => {
        return get().progressMap[book_id] || null;
      },

      getReadingHistory: () => {
        return Object.values(get().progressMap).sort(
          (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
        );
      },

      syncToServer: async (featureService) => {
        const { bookmarks, progressMap, clearGuestData } = get();

        // 1. Sync bookmarks
        const bookmarkPromises = bookmarks.map((b) =>
          featureService.setBookmark(b.book_id, true).catch(() => undefined)
        );

        // 2. Sync reading progress/activity
        const historyList = Object.values(progressMap);
        const historyPromises = historyList.map((h) =>
          featureService
            .recordReadingActivity({
              book_id: h.book_id,
              file_id: h.file_id,
              chapter_id: h.chapter_id,
              chapter_title: h.chapter_title,
              chapter_index: h.chapter_index,
              progress_percent: h.progress_percent,
              event_type: "progress_update",
            })
            .catch(() => undefined)
        );

        await Promise.allSettled([...bookmarkPromises, ...historyPromises]);
        clearGuestData();
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
