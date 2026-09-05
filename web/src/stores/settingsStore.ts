import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { PublicSettings } from "@/types";

export type Theme =
  | "system"
  | "winter"
  | "night"
  | "cupcake"
  | "coffee"
  | "nord"
  | "dracula"
  | "sunset"
  | "cyberpunk"
  | "catppuccin"
  | "emerald"
  | "retro"
  | "synthwave"
  | "dim"
  | "silk";

export type Language =
  | "en"
  | "vi"
  | "ja"
  | "ko"
  | "zh"
  | "zh-CN"
  | "zh-TW"
  | "es"
  | "fr"
  | "de"
  | "pt"
  | "ru"
  | "ar"
  | "hi"
  | "id"
  | "th"
  | "it";

interface SettingsState {
  theme: Theme;
  language: Language;
  customCss: string;
  publicSettings: PublicSettings | null;
  setTheme: (theme: Theme) => void;
  setLanguage: (lang: Language) => void;
  setCustomCss: (css: string) => void;
  setPublicSettings: (settings: PublicSettings | null) => void;
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      theme: "system",
      language: "en",
      customCss: "",
      publicSettings: null,
      setTheme: (theme) => set({ theme }),
      setLanguage: (language) => set({ language }),
      setCustomCss: (customCss) => set({ customCss }),
      setPublicSettings: (publicSettings) => set({ publicSettings }),
    }),
    {
      name: "novelhub-settings",
      partialize: (state) => ({
        theme: state.theme,
        language: state.language,
        customCss: state.customCss,
      }),
    },
  ),
);
