import type { PublicSettings } from "@/types";
import { create } from "zustand";

interface SettingsAdminState {
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
  shareMode: string;
  shareLibraryIds: string[];
  readMode: string;
  readLibraryIds: string[];
  statsMode: string;
  statsLibraryIds: string[];
  statsVisibleStats: string[];
  inBookSearch: boolean;
  customFontUpload: boolean;

  savingSection: string | null;
  uploadingLogo: boolean;
  uploadingFavicon: boolean;
  selectedCropImage: string | null;
  cropTarget: "logo" | "favicon" | null;

  setSite: (site: { title: string; description: string; favicon: string; logo: string; meta_description: string } | ((prev: any) => any)) => void;
  setSidebarItems: (items: string[] | ((prev: string[]) => string[])) => void;
  setHomeSections: (sections: { random_books: boolean; top_books: boolean } | ((prev: any) => any)) => void;
  setRegistration: (enabled: boolean) => void;
  setInBookSearch: (enabled: boolean) => void;
  setCustomFontUpload: (enabled: boolean) => void;
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
  setShareMode: (mode: string) => void;
  setShareLibraryIds: (ids: string[] | ((prev: string[]) => string[])) => void;
  setReadMode: (mode: string) => void;
  setReadLibraryIds: (ids: string[] | ((prev: string[]) => string[])) => void;
  setStatsMode: (mode: string) => void;
  setStatsLibraryIds: (ids: string[] | ((prev: string[]) => string[])) => void;
  setStatsVisibleStats: (stats: string[] | ((prev: string[]) => string[])) => void;

  setSavingSection: (section: string | null) => void;
  setUploadingLogo: (uploading: boolean) => void;
  setUploadingFavicon: (uploading: boolean) => void;
  setSelectedCropImage: (img: string | null) => void;
  setCropTarget: (target: "logo" | "favicon" | null) => void;

  initFromSettings: (s: PublicSettings) => void;
  reset: () => void;
}

const initialSite = { title: "", description: "", favicon: "", logo: "", meta_description: "" };
const initialHomeSections = { random_books: true, top_books: true };
const ALL_STATS_KEYS = ["reads", "downloads", "bookmarks", "collections", "rating", "shares"];

const initialState = {
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
  shareMode: "all",
  shareLibraryIds: [],
  readMode: "all",
  readLibraryIds: [],
  statsMode: "all",
  statsLibraryIds: [],
  statsVisibleStats: ALL_STATS_KEYS,
  inBookSearch: false,
  customFontUpload: false,
  savingSection: null,
  uploadingLogo: false,
  uploadingFavicon: false,
  selectedCropImage: null,
  cropTarget: null,
};

export const useSettingsAdminStore = create<SettingsAdminState>((set) => ({
  ...initialState,

  setSite: (site) => set((state) => ({ site: typeof site === "function" ? site(state.site) : site })),
  setSidebarItems: (sidebarItems) => set((state) => ({ sidebarItems: typeof sidebarItems === "function" ? sidebarItems(state.sidebarItems) : sidebarItems })),
  setHomeSections: (homeSections) => set((state) => ({ homeSections: typeof homeSections === "function" ? homeSections(state.homeSections) : homeSections })),
  setRegistration: (registration) => set({ registration }),
  setInBookSearch: (inBookSearch) => set({ inBookSearch }),
  setCustomFontUpload: (customFontUpload) => set({ customFontUpload }),
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
  setShareMode: (shareMode) => set({ shareMode }),
  setShareLibraryIds: (shareLibraryIds) => set((state) => ({ shareLibraryIds: typeof shareLibraryIds === "function" ? shareLibraryIds(state.shareLibraryIds) : shareLibraryIds })),
  setReadMode: (readMode) => set({ readMode }),
  setReadLibraryIds: (readLibraryIds) => set((state) => ({ readLibraryIds: typeof readLibraryIds === "function" ? readLibraryIds(state.readLibraryIds) : readLibraryIds })),
  setStatsMode: (statsMode) => set({ statsMode }),
  setStatsLibraryIds: (statsLibraryIds) => set((state) => ({ statsLibraryIds: typeof statsLibraryIds === "function" ? statsLibraryIds(state.statsLibraryIds) : statsLibraryIds })),
  setStatsVisibleStats: (statsVisibleStats) => set((state) => ({ statsVisibleStats: typeof statsVisibleStats === "function" ? statsVisibleStats(state.statsVisibleStats) : statsVisibleStats })),

  setSavingSection: (savingSection) => set({ savingSection }),
  setUploadingLogo: (uploadingLogo) => set({ uploadingLogo }),
  setUploadingFavicon: (uploadingFavicon) => set({ uploadingFavicon }),
  setSelectedCropImage: (selectedCropImage) => set({ selectedCropImage }),
  setCropTarget: (cropTarget) => set({ cropTarget }),

  initFromSettings: (s) =>
    set({
      site: s.site || initialSite,
      sidebarItems: s.sidebar_visible_items || [],
      homeSections: s.home_sections || initialHomeSections,
      registration: s.registration_enabled,
      inBookSearch: s.enable_in_book_search || false,
      customFontUpload: s.enable_custom_font_upload || false,
      guestMode: s.guest_access?.mode || "all",
      guestLibraryIds: s.guest_access?.library_ids || [],
      downloadMode: s.download?.mode || "all",
      downloadLibraryIds: s.download?.library_ids || [],
      bookmarkMode: s.bookmark?.mode || "all",
      bookmarkLibraryIds: s.bookmark?.library_ids || [],
      collectionMode: s.collection?.mode || "all",
      collectionLibraryIds: s.collection?.library_ids || [],
      reviewMode: s.review?.mode || "all",
      reviewLibraryIds: s.review?.library_ids || [],
      shareMode: s.share?.mode || "all",
      shareLibraryIds: s.share?.library_ids || [],
      readMode: s.read?.mode || "all",
      readLibraryIds: s.read?.library_ids || [],
      statsMode: s.stats?.mode || "all",
      statsLibraryIds: s.stats?.library_ids || [],
      statsVisibleStats: s.stats?.visible_stats || ALL_STATS_KEYS,
    }),

  reset: () => set(initialState),
}));
