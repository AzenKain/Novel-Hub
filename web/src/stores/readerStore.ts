import type { Book, Chapter } from "@/types";
import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';

export type ReaderTheme = "light" | "dark" | "sepia" | "warm" | "coffee" | "dim" | "eink" | "custom";
export type ReadingMode = "scroll" | "single" | "double" | "webtoon";
export type ReadingDirection = "ltr" | "rtl";
export type PageFit = "width" | "height" | "original";
export type PageAnimation = "eink" | "none" | "fade" | "slide";
export type TextAlignment = "original" | "left" | "justify" | "center" | "right";

interface ReaderState {
  book: Book | null;
  chapters: Chapter[];
  currentChapter: Chapter | null;
  htmlContent: string;
  loading: boolean;
  sidebarOpen: boolean;
  settingsOpen: boolean;
  fontSize: number;
  fontFamily: string;
  theme: ReaderTheme;
  customBg: string;
  customText: string;
  customAccent: string;
  customCss: string;
  lineHeight: number;
  maxWidth: number;
  textAlign: TextAlignment;
  readingMode: ReadingMode;
  readingDirection: ReadingDirection;
  pageFit: PageFit;
  pageAnimation: PageAnimation;
  pageIndex: number;
  pageFrameWidth: number;
  ttsVoiceName: string | null;
  ttsRate: number;
  comicInvertColors: boolean;

  setBook: (book: Book | null) => void;
  setChapters: (chapters: Chapter[]) => void;
  setCurrentChapter: (chapter: Chapter | null) => void;
  setHtmlContent: (htmlContent: string) => void;
  setLoading: (loading: boolean) => void;
  setSidebarOpen: (open: boolean) => void;
  setSettingsOpen: (open: boolean) => void;
  setFontSize: (size: number | ((prev: number) => number)) => void;
  setFontFamily: (family: string) => void;
  setTheme: (theme: ReaderTheme) => void;
  setCustomThemeColors: (bg: string, text: string, accent: string, customCss?: string) => void;
  setLineHeight: (height: number | ((prev: number) => number)) => void;
  setMaxWidth: (width: number | ((prev: number) => number)) => void;
  setTextAlign: (textAlign: TextAlignment) => void;
  setReadingMode: (mode: ReadingMode) => void;
  setReadingDirection: (direction: ReadingDirection) => void;
  setPageFit: (fit: PageFit) => void;
  setPageAnimation: (anim: PageAnimation) => void;
  setPageIndex: (index: number | ((prev: number) => number)) => void;
  setPageFrameWidth: (width: number) => void;
  setTtsVoiceName: (voiceName: string | null) => void;
  setTtsRate: (rate: number) => void;
  setComicInvertColors: (invert: boolean) => void;
  resetSettings: () => void;
  reset: () => void;
}

const sessionInitialState = {
  book: null,
  chapters: [],
  currentChapter: null,
  htmlContent: "",
  loading: true,
  sidebarOpen: true,
  settingsOpen: false,
  pageIndex: 0,
  pageFrameWidth: 0,
};

const readerSettingDefaults = {
  fontSize: 18,
  fontFamily: "'Lora', serif",
  theme: "dark" as ReaderTheme,
  customBg: "#1e1e2e",
  customText: "#cdd6f4",
  customAccent: "#89b4fa",
  customCss: "",
  lineHeight: 1.8,
  maxWidth: 920,
  textAlign: "original" as TextAlignment,
  readingMode: "scroll" as const,
  readingDirection: "ltr" as const,
  pageFit: "height" as const,
  pageAnimation: "eink" as const,
  ttsVoiceName: null as string | null,
  ttsRate: 1.0,
  comicInvertColors: false,
};

const initialState = {
  ...sessionInitialState,
  ...readerSettingDefaults,
};

export const useReaderStore = create<ReaderState>()(
  persist(
    (set) => ({
      ...initialState,

      setBook: (book) => set({ book }),
      setChapters: (chapters) => set({ chapters }),
      setCurrentChapter: (currentChapter) => set({ currentChapter }),
      setHtmlContent: (htmlContent) => set({ htmlContent }),
      setLoading: (loading) => set({ loading }),
      setSidebarOpen: (sidebarOpen) => set({ sidebarOpen }),
      setSettingsOpen: (settingsOpen) => set({ settingsOpen }),
      setFontSize: (size) => set((state) => ({ fontSize: typeof size === 'function' ? size(state.fontSize) : size })),
      setFontFamily: (fontFamily) => set({ fontFamily }),
      setTheme: (theme) => set({ theme }),
      setCustomThemeColors: (customBg, customText, customAccent, customCss = "") =>
        set({ customBg, customText, customAccent, customCss, theme: "custom" }),
      setLineHeight: (height) => set((state) => ({ lineHeight: typeof height === 'function' ? height(state.lineHeight) : height })),
      setMaxWidth: (width) => set((state) => ({ maxWidth: typeof width === 'function' ? width(state.maxWidth) : width })),
      setTextAlign: (textAlign) => set({ textAlign }),
      setReadingMode: (readingMode) => set({ readingMode }),
      setReadingDirection: (readingDirection) => set({ readingDirection }),
      setPageFit: (pageFit) => set({ pageFit }),
      setPageAnimation: (pageAnimation) => set({ pageAnimation }),
      setPageIndex: (index) => set((state) => ({ pageIndex: typeof index === 'function' ? index(state.pageIndex) : index })),
      setPageFrameWidth: (pageFrameWidth) => set({ pageFrameWidth }),
      setTtsVoiceName: (ttsVoiceName) => set({ ttsVoiceName }),
      setTtsRate: (ttsRate) => set({ ttsRate }),
      setComicInvertColors: (comicInvertColors) => set({ comicInvertColors }),
      resetSettings: () => set(readerSettingDefaults),
      reset: () => set(sessionInitialState),
    }),
    {
      name: 'novelhub-reader-settings',
      storage: createJSONStorage(() => localStorage),
      version: 5,
      partialize: (state) => ({
        fontSize: state.fontSize,
        fontFamily: state.fontFamily,
        theme: state.theme,
        customBg: state.customBg,
        customText: state.customText,
        customAccent: state.customAccent,
        customCss: state.customCss,
        lineHeight: state.lineHeight,
        maxWidth: state.maxWidth,
        textAlign: state.textAlign,
        readingMode: state.readingMode,
        readingDirection: state.readingDirection,
        pageFit: state.pageFit,
        pageAnimation: state.pageAnimation,
        ttsVoiceName: state.ttsVoiceName,
        ttsRate: state.ttsRate,
        comicInvertColors: state.comicInvertColors,
      }),
    }
  )
);
