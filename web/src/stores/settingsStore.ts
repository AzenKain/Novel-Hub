import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { PublicSettings } from '@/types';

export type Theme = 'winter' | 'night' | 'cupcake' | 'coffee' | 'system';
export type Language = 'en' | 'vi' | 'ja' | 'ko' | 'zh' | 'zh-CN' | 'zh-TW' | 'es' | 'fr' | 'de' | 'pt' | 'ru' | 'ar' | 'hi' | 'id' | 'th' | 'it';

interface SettingsState {
  theme: Theme;
  language: Language;
  publicSettings: PublicSettings | null;
  setTheme: (theme: Theme) => void;
  setLanguage: (lang: Language) => void;
  setPublicSettings: (settings: PublicSettings | null) => void;
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      theme: 'system',
      language: 'en',
      publicSettings: null,
      setTheme: (theme) => set({ theme }),
      setLanguage: (language) => set({ language }),
      setPublicSettings: (publicSettings) => set({ publicSettings }),
    }),
    {
      name: 'novelhub-settings',
    }
  )
);
