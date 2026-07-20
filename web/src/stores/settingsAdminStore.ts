import { adminService, libraryService } from "@/services";
import type { Library as LibraryType, PublicSettings } from "@/types";
import { create } from "zustand";

interface SettingsAdminState {
  settings: PublicSettings | null;
  libraries: LibraryType[];
  loading: boolean;
  saving: string | null;

  site: { title: string; description: string; favicon: string; logo: string; meta_description: string };
  sidebarItems: string[];
  homeSections: { random_books: boolean; top_books: boolean };
  registration: boolean;
  guestMode: string;
  guestLibraryIds: string[];
  downloadMode: string;
  downloadLibraryIds: string[];
  bookmarkMode: string;
  bookmarkLibraryIds: string[];
  collectionMode: string;
  collectionLibraryIds: string[];
  reviewMode: string;
  reviewLibraryIds: string[];

  setSettings: (settings: PublicSettings | null) => void;
  setLibraries: (libraries: LibraryType[]) => void;
  setLoading: (loading: boolean) => void;
  setSaving: (saving: string | null) => void;

  setSite: (site: { title: string; description: string; favicon: string; logo: string; meta_description: string } | ((prev: any) => any)) => void;
  setSidebarItems: (items: string[] | ((prev: string[]) => string[])) => void;
  setHomeSections: (sections: { random_books: boolean; top_books: boolean } | ((prev: any) => any)) => void;
  setRegistration: (enabled: boolean) => void;
  setGuestMode: (mode: string) => void;
  setGuestLibraryIds: (ids: string[] | ((prev: string[]) => string[])) => void;
  setDownloadMode: (mode: string) => void;
  setDownloadLibraryIds: (ids: string[] | ((prev: string[]) => string[])) => void;
  setBookmarkMode: (mode: string) => void;
  setBookmarkLibraryIds: (ids: string[] | ((prev: string[]) => string[])) => void;
  setCollectionMode: (mode: string) => void;
  setCollectionLibraryIds: (ids: string[] | ((prev: string[]) => string[])) => void;
  setReviewMode: (mode: string) => void;
  setReviewLibraryIds: (ids: string[] | ((prev: string[]) => string[])) => void;

  loadData: () => Promise<void>;
  reset: () => void;
}

const initialSite = { title: "", description: "", favicon: "", logo: "", meta_description: "" };
const initialHomeSections = { random_books: true, top_books: true };

const initialState = {
  settings: null,
  libraries: [],
  loading: true,
  saving: null,

  site: initialSite,
  sidebarItems: [],
  homeSections: initialHomeSections,
  registration: true,
  guestMode: "all",
  guestLibraryIds: [],
  downloadMode: "all",
  downloadLibraryIds: [],
  bookmarkMode: "all",
  bookmarkLibraryIds: [],
  collectionMode: "all",
  collectionLibraryIds: [],
  reviewMode: "all",
  reviewLibraryIds: [],
};

export const useSettingsAdminStore = create<SettingsAdminState>((set, get) => ({
  ...initialState,

  setSettings: (settings) => set({ settings }),
  setLibraries: (libraries) => set({ libraries }),
  setLoading: (loading) => set({ loading }),
  setSaving: (saving) => set({ saving }),

  setSite: (site) => set((state) => ({ site: typeof site === "function" ? site(state.site) : site })),
  setSidebarItems: (sidebarItems) => set((state) => ({ sidebarItems: typeof sidebarItems === "function" ? sidebarItems(state.sidebarItems) : sidebarItems })),
  setHomeSections: (homeSections) => set((state) => ({ homeSections: typeof homeSections === "function" ? homeSections(state.homeSections) : homeSections })),
  setRegistration: (registration) => set({ registration }),
  setGuestMode: (guestMode) => set({ guestMode }),
  setGuestLibraryIds: (guestLibraryIds) => set((state) => ({ guestLibraryIds: typeof guestLibraryIds === "function" ? guestLibraryIds(state.guestLibraryIds) : guestLibraryIds })),
  setDownloadMode: (downloadMode) => set({ downloadMode }),
  setDownloadLibraryIds: (downloadLibraryIds) => set((state) => ({ downloadLibraryIds: typeof downloadLibraryIds === "function" ? downloadLibraryIds(state.downloadLibraryIds) : downloadLibraryIds })),
  setBookmarkMode: (bookmarkMode) => set({ bookmarkMode }),
  setBookmarkLibraryIds: (bookmarkLibraryIds) => set((state) => ({ bookmarkLibraryIds: typeof bookmarkLibraryIds === "function" ? bookmarkLibraryIds(state.bookmarkLibraryIds) : bookmarkLibraryIds })),
  setCollectionMode: (collectionMode) => set({ collectionMode }),
  setCollectionLibraryIds: (collectionLibraryIds) => set((state) => ({ collectionLibraryIds: typeof collectionLibraryIds === "function" ? collectionLibraryIds(state.collectionLibraryIds) : collectionLibraryIds })),
  setReviewMode: (reviewMode) => set({ reviewMode }),
  setReviewLibraryIds: (reviewLibraryIds) => set((state) => ({ reviewLibraryIds: typeof reviewLibraryIds === "function" ? reviewLibraryIds(state.reviewLibraryIds) : reviewLibraryIds })),

  loadData: async () => {
    set({ loading: true });
    try {
      const [settingsRes, libRes] = await Promise.all([
        adminService.getAdminSettings(),
        libraryService.getLibraries(),
      ]);
      const s = settingsRes.data;
      set({ settings: s || null, libraries: libRes.data || [] });
      if (s) {
        set({
          site: s.site || initialSite,
          sidebarItems: s.sidebar_visible_items || [],
          homeSections: s.home_sections || initialHomeSections,
          registration: s.registration_enabled,
          guestMode: s.guest_access.mode,
          guestLibraryIds: s.guest_access.library_ids || [],
          downloadMode: s.download.mode,
          downloadLibraryIds: s.download.library_ids || [],
          bookmarkMode: s.bookmark.mode,
          bookmarkLibraryIds: s.bookmark.library_ids || [],
          collectionMode: s.collection.mode,
          collectionLibraryIds: s.collection.library_ids || [],
          reviewMode: s.review.mode,
          reviewLibraryIds: s.review.library_ids || [],
        });
      }
    } catch (err) {
      console.error(err);
    } finally {
      set({ loading: false });
    }
  },

  reset: () => set(initialState),
}));
