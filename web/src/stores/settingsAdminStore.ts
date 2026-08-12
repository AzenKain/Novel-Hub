import type { AdminSettings, RuntimeLimitBounds, RuntimeLimits } from "@/types";
import { create } from "zustand";

interface SettingsAdminState {
  site: { title: string; description: string; favicon: string; logo: string; meta_description: string };
  serverUrl: string;
  sidebarItems: string[];
  homeSections: { random_books: boolean; top_books: boolean };
  registration: boolean;
  loginRequired: boolean;
  requireEmailVerify: boolean;
  passwordResetEnabled: boolean;
  guestMode: string;
  guestLibraryIds: string[];
  inBookSearch: boolean;
  customFontUpload: boolean;
  anilistTracking: boolean;
  hardcoverEnabled: boolean;
  hardcoverClientId: string;
  hardcoverClientSecret: string;
  autoEnrich: boolean;
  webpCover: boolean;
  limits: RuntimeLimits | null;
  limitBounds: RuntimeLimitBounds | null;
  proxyAuth: { enabled: boolean; header_names: string[]; trusted_proxies: string[]; auto_create: boolean };

  savingSection: string | null;
  uploadingLogo: boolean;
  uploadingFavicon: boolean;
  selectedCropImage: string | null;
  cropTarget: "logo" | "favicon" | null;

  setSite: (site: { title: string; description: string; favicon: string; logo: string; meta_description: string } | ((prev: any) => any)) => void;
  setServerUrl: (url: string) => void;
  setSidebarItems: (items: string[] | ((prev: string[]) => string[])) => void;
  setHomeSections: (sections: { random_books: boolean; top_books: boolean } | ((prev: any) => any)) => void;
  setRegistration: (enabled: boolean) => void;
  setLoginRequired: (enabled: boolean) => void;
  setRequireEmailVerify: (enabled: boolean) => void;
  setPasswordResetEnabled: (enabled: boolean) => void;
  setInBookSearch: (enabled: boolean) => void;
  setCustomFontUpload: (enabled: boolean) => void;
  setAnilistTracking: (enabled: boolean) => void;
  setHardcoverEnabled: (enabled: boolean) => void;
  setHardcoverClientId: (id: string) => void;
  setHardcoverClientSecret: (secret: string) => void;
  setAutoEnrich: (enabled: boolean) => void;
  setWebpCover: (enabled: boolean) => void;
  setLimits: (limits: RuntimeLimits) => void;
  setGuestMode: (mode: string) => void;
  setGuestLibraryIds: (ids: string[] | ((prev: string[]) => string[])) => void;
  setProxyAuth: (proxyAuth: { enabled: boolean; header_names: string[]; trusted_proxies: string[]; auto_create: boolean } | ((prev: any) => any)) => void;

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
const initialProxyAuth = { enabled: false, header_names: ["X-Forwarded-User", "Remote-User", "X-Forwarded-Email"], trusted_proxies: ["127.0.0.1", "::1"], auto_create: false };

const initialState = {
  site: initialSite,
  serverUrl: "",
  sidebarItems: [],
  homeSections: initialHomeSections,
  registration: true,
  loginRequired: false,
  requireEmailVerify: false,
  passwordResetEnabled: false,
  guestMode: "all",
  guestLibraryIds: [],
  inBookSearch: false,
  customFontUpload: false,
  anilistTracking: true,
  hardcoverEnabled: false,
  hardcoverClientId: "",
  hardcoverClientSecret: "",
  autoEnrich: false,
  webpCover: false,
  limits: null,
  limitBounds: null,
  proxyAuth: initialProxyAuth,
  savingSection: null,
  uploadingLogo: false,
  uploadingFavicon: false,
  selectedCropImage: null,
  cropTarget: null,
};

export const useSettingsAdminStore = create<SettingsAdminState>((set) => ({
  ...initialState,

  setSite: (site) => set((state) => ({ site: typeof site === "function" ? site(state.site) : site })),
  setServerUrl: (serverUrl) => set({ serverUrl }),
  setSidebarItems: (sidebarItems) => set((state) => ({ sidebarItems: typeof sidebarItems === "function" ? sidebarItems(state.sidebarItems) : sidebarItems })),
  setHomeSections: (homeSections) => set((state) => ({ homeSections: typeof homeSections === "function" ? homeSections(state.homeSections) : homeSections })),
  setRegistration: (registration) => set({ registration }),
  setLoginRequired: (loginRequired) => set({ loginRequired }),
  setRequireEmailVerify: (requireEmailVerify) => set({ requireEmailVerify }),
  setPasswordResetEnabled: (passwordResetEnabled) => set({ passwordResetEnabled }),
  setInBookSearch: (inBookSearch) => set({ inBookSearch }),
  setCustomFontUpload: (customFontUpload) => set({ customFontUpload }),
  setAnilistTracking: (anilistTracking) => set({ anilistTracking }),
  setHardcoverEnabled: (hardcoverEnabled) => set({ hardcoverEnabled }),
  setHardcoverClientId: (hardcoverClientId) => set({ hardcoverClientId }),
  setHardcoverClientSecret: (hardcoverClientSecret) => set({ hardcoverClientSecret }),
  setAutoEnrich: (autoEnrich) => set({ autoEnrich }),
  setWebpCover: (webpCover) => set({ webpCover }),
  setLimits: (limits) => set({ limits }),
  setGuestMode: (guestMode) => set({ guestMode }),
  setGuestLibraryIds: (guestLibraryIds) => set((state) => ({ guestLibraryIds: typeof guestLibraryIds === "function" ? guestLibraryIds(state.guestLibraryIds) : guestLibraryIds })),
  setProxyAuth: (proxyAuth) => set((state) => ({ proxyAuth: typeof proxyAuth === "function" ? proxyAuth(state.proxyAuth) : proxyAuth })),

  setSavingSection: (savingSection) => set({ savingSection }),
  setUploadingLogo: (uploadingLogo) => set({ uploadingLogo }),
  setUploadingFavicon: (uploadingFavicon) => set({ uploadingFavicon }),
  setSelectedCropImage: (selectedCropImage) => set({ selectedCropImage }),
  setCropTarget: (cropTarget) => set({ cropTarget }),

  initFromSettings: (s) =>
    set({
      site: s.site || initialSite,
      serverUrl: s.server_url ?? "",
      sidebarItems: s.sidebar_visible_items || [],
      homeSections: s.home_sections || initialHomeSections,
      registration: s.registration_enabled,
      loginRequired: s.guest_login_required || false,
      requireEmailVerify: s.require_email_verify ?? false,
      passwordResetEnabled: s.password_reset_enabled ?? false,
      inBookSearch: s.enable_in_book_search || false,
      customFontUpload: s.enable_custom_font_upload || false,
      anilistTracking: s.enable_anilist_tracking ?? true,
      hardcoverEnabled: s.enable_hardcover_scrobbling ?? false,
      hardcoverClientId: s.hardcover?.client_id ?? "",
      hardcoverClientSecret: "",
      autoEnrich: s.enable_auto_enrich ?? false,
      webpCover: s.enable_webp_cover ?? false,
      limits: s.limits,
      limitBounds: s.bounds,
      guestMode: s.guest_access?.mode || "all",
      guestLibraryIds: s.guest_access?.library_ids || [],
      proxyAuth: s.proxy_auth || initialProxyAuth,
    }),

  reset: () => set(initialState),
}));
