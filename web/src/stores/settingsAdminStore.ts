import type { AdminSettings, RuntimeLimitBounds, RuntimeLimits } from "@/types";
import { create } from "zustand";

interface SettingsAdminState {
  site: { title: string; description: string; favicon: string; logo: string; meta_description: string };
  sidebarItems: string[];
  homeSections: { random_books: boolean; top_books: boolean };
  registration: boolean;
  guestMode: string;
  guestLibraryIds: string[];
  inBookSearch: boolean;
  customFontUpload: boolean;
  limits: RuntimeLimits | null;
  limitBounds: RuntimeLimitBounds | null;

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
  setLimits: (limits: RuntimeLimits) => void;
  setGuestMode: (mode: string) => void;
  setGuestLibraryIds: (ids: string[] | ((prev: string[]) => string[])) => void;

  setSavingSection: (section: string | null) => void;
  setUploadingLogo: (uploading: boolean) => void;
  setUploadingFavicon: (uploading: boolean) => void;
  setSelectedCropImage: (img: string | null) => void;
  setCropTarget: (target: "logo" | "favicon" | null) => void;

  initFromSettings: (s: AdminSettings) => void;
  reset: () => void;
}

const initialSite = { title: "", description: "", favicon: "", logo: "", meta_description: "" };
const initialHomeSections = { random_books: true, top_books: true };

const initialState = {
  site: initialSite,
  sidebarItems: [],
  homeSections: initialHomeSections,
  registration: true,
  guestMode: "all",
  guestLibraryIds: [],
  inBookSearch: false,
  customFontUpload: false,
  limits: null,
  limitBounds: null,
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
  setLimits: (limits) => set({ limits }),
  setGuestMode: (guestMode) => set({ guestMode }),
  setGuestLibraryIds: (guestLibraryIds) => set((state) => ({ guestLibraryIds: typeof guestLibraryIds === "function" ? guestLibraryIds(state.guestLibraryIds) : guestLibraryIds })),

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
      limits: s.limits,
      limitBounds: s.bounds,
      guestMode: s.guest_access?.mode || "all",
      guestLibraryIds: s.guest_access?.library_ids || [],
    }),

  reset: () => set(initialState),
}));
