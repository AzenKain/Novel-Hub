import type { Book, Chapter } from "@/types";
import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';

export type ReaderTheme = "light" | "dark" | "sepia" | "warm" | "coffee" | "dim";
export type ReadingMode = "scroll" | "single" | "double" | "webtoon";
export type ReadingDirection = "ltr" | "rtl";
export type PageFit = "width" | "height" | "original";
export type PageAnimation = "eink" | "none" | "fade" | "slide";

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
  lineHeight: number;
  maxWidth: number;
  readingMode: ReadingMode;
  readingDirection: ReadingDirection;
  pageFit: PageFit;
  pageAnimation: PageAnimation;
  pageIndex: number;
  pageFrameWidth: number;
  ttsVoiceName: string | null;
  ttsRate: number;

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
  setLineHeight: (height: number | ((prev: number) => number)) => void;
  setMaxWidth: (width: number | ((prev: number) => number)) => void;
  setReadingMode: (mode: ReadingMode) => void;
  setReadingDirection: (direction: ReadingDirection) => void;
  setPageFit: (fit: PageFit) => void;
  setPageAnimation: (anim: PageAnimation) => void;
  setPageIndex: (index: number | ((prev: number) => number)) => void;
  setPageFrameWidth: (width: number) => void;
  setTtsVoiceName: (voiceName: string | null) => void;
  setTtsRate: (rate: number) => void;
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
  theme: "dark" as const,
  lineHeight: 1.8,
  maxWidth: 920,
  readingMode: "scroll" as const,
  readingDirection: "ltr" as const,
  pageFit: "height" as const,
  pageAnimation: "eink" as const,
  ttsVoiceName: null as string | null,
  ttsRate: 1.0,
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
      setLineHeight: (height) => set((state) => ({ lineHeight: typeof height === 'function' ? height(state.lineHeight) : height })),
      setMaxWidth: (width) => set((state) => ({ maxWidth: typeof width === 'function' ? width(state.maxWidth) : width })),
      setReadingMode: (readingMode) => set({ readingMode }),
      setReadingDirection: (readingDirection) => set({ readingDirection }),
      setPageFit: (pageFit) => set({ pageFit }),
      setPageAnimation: (pageAnimation) => set({ pageAnimation }),
      setPageIndex: (index) => set((state) => ({ pageIndex: typeof index === 'function' ? index(state.pageIndex) : index })),
      setPageFrameWidth: (pageFrameWidth) => set({ pageFrameWidth }),
      setTtsVoiceName: (ttsVoiceName) => set({ ttsVoiceName }),
      setTtsRate: (ttsRate) => set({ ttsRate }),
      resetSettings: () => set(readerSettingDefaults),
      reset: () => set(sessionInitialState),
    }),
    {
      name: 'novelhub-reader-settings',
      storage: createJSONStorage(() => localStorage),
      version: 4,
      partialize: (state) => ({
        fontSize: state.fontSize,
        fontFamily: state.fontFamily,
        theme: state.theme,
        lineHeight: state.lineHeight,
        maxWidth: state.maxWidth,
        readingMode: state.readingMode,
        readingDirection: state.readingDirection,
        pageFit: state.pageFit,
        pageAnimation: state.pageAnimation,
        ttsVoiceName: state.ttsVoiceName,
        ttsRate: state.ttsRate,
      }),
    }
  )
);
