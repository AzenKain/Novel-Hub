import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type Theme = 'winter' | 'night' | 'cupcake' | 'coffee' | 'system';
export type Language = 'en' | 'vi' | 'ja' | 'zh' | 'ko';

interface SettingsState {
  theme: Theme;
  language: Language;
  setTheme: (theme: Theme) => void;
  setLanguage: (lang: Language) => void;
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      theme: 'system',
      language: 'en',
      setTheme: (theme) => set({ theme }),
      setLanguage: (language) => set({ language }),
    }),
    {
      name: 'novelhub-settings',
    }
  )
);
