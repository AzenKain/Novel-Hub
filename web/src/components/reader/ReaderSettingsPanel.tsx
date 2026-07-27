import type { TFunction } from "i18next";
import {
  AlignLeft,
  Minus,
  Moon,
  Plus,
  RotateCcw,
  Sun,
} from "lucide-react";
import React from "react";

import type { PageFit, ReaderTheme, ReadingDirection, ReadingMode } from "@/stores";

type ReaderSettingsPanelProps = {
  t: TFunction;
  theme: ReaderTheme;
  fontFamily: string;
  fontSize: number;
  maxWidth: number;
  effectiveReadingMode: ReadingMode;
  canUseDoubleMode: boolean;
  isVisualContent: boolean;
  readingDirection: ReadingDirection;
  pageFit: PageFit;
  setTheme: (theme: ReaderTheme) => void;
  setFontFamily: (family: string) => void;
  setFontSize: (size: number | ((prev: number) => number)) => void;
  setMaxWidth: (width: number | ((prev: number) => number)) => void;
  setReadingMode: (mode: ReadingMode) => void;
  setReadingDirection: (direction: ReadingDirection) => void;
  setPageFit: (fit: PageFit) => void;
  resetSettings: () => void;
};

export const ReaderSettingsPanel: React.FC<ReaderSettingsPanelProps> = ({
  t,
  theme,
  fontFamily,
  fontSize,
  maxWidth,
  effectiveReadingMode,
  canUseDoubleMode,
  isVisualContent,
  readingDirection,
  pageFit,
  setTheme,
  setFontFamily,
  setFontSize,
  setMaxWidth,
  setReadingMode,
  setReadingDirection,
  setPageFit,
  resetSettings,
}) => {

  return (
  <div className="reader-settings-panel absolute right-0 top-full z-50 mt-2 max-h-[calc(100vh-5rem)] w-72 overflow-y-auto rounded-2xl border p-4 shadow-2xl transition-colors duration-300">
    <h3 className="mb-4 text-xs font-bold uppercase tracking-wider opacity-50">
      {t("reader.settings", "Reader Settings")}
    </h3>

    <div className="mb-6 grid grid-cols-3 gap-2">
      <button
        onClick={() => setTheme("light")}
        className={`reader-theme-choice flex items-center justify-center rounded-lg border py-2 transition-all ${theme === "light" ? "reader-theme-choice-active" : ""}`}
      >
        <Sun className="h-4 w-4" />
      </button>
      <button
        onClick={() => setTheme("sepia")}
        className={`reader-theme-choice flex items-center justify-center rounded-lg border py-2 transition-all ${theme === "sepia" ? "reader-theme-choice-active" : ""}`}
      >
        <AlignLeft className="h-4 w-4" />
      </button>
      <button
        onClick={() => setTheme("warm")}
        className={`reader-theme-choice flex items-center justify-center rounded-lg border py-2 transition-all ${theme === "warm" ? "reader-theme-choice-active" : ""}`}
      >
        <span className="text-xs font-medium">
          {t("reader.theme_warm", "Warm")}
        </span>
      </button>
      <button
        onClick={() => setTheme("dark")}
        className={`reader-theme-choice flex items-center justify-center rounded-lg border py-2 transition-all ${theme === "dark" ? "reader-theme-choice-active" : ""}`}
      >
        <Moon className="h-4 w-4" />
      </button>
      <button
        onClick={() => setTheme("dim")}
        className={`reader-theme-choice flex items-center justify-center rounded-lg border py-2 transition-all ${theme === "dim" ? "reader-theme-choice-active" : ""}`}
      >
        <span className="text-xs font-medium">
          {t("reader.theme_dim", "Dim")}
        </span>
      </button>
      <button
        onClick={() => setTheme("coffee")}
        className={`reader-theme-choice flex items-center justify-center rounded-lg border py-2 transition-all ${theme === "coffee" ? "reader-theme-choice-active" : ""}`}
      >
        <span className="text-xs font-medium">
          {t("common.coffee", "Coffee")}
        </span>
      </button>
    </div>

    <div className="mb-4">
      <div className="mb-2 flex items-center justify-between">
        <span className="text-sm font-medium opacity-80">
          {t("reader.font_family", "Font Family")}
        </span>
      </div>
      <select
        className="reader-select select select-bordered select-sm w-full"
        value={fontFamily}
        onChange={(event) => setFontFamily(event.target.value)}
      >
        <option value="sans-serif">
          {t("reader.system_default", "System Default")}
        </option>
        <option value="'Noto Sans', sans-serif">Noto Sans (Multi-lang)</option>
        <option value="'Inter', sans-serif">Inter</option>
        <option value="'Roboto', sans-serif">Roboto</option>
        <option value="'Open Sans', sans-serif">Open Sans</option>
        <option value="'Quicksand', sans-serif">Quicksand</option>
        <option value="serif">
          {t("reader.serif_default", "Serif (Default)")}
        </option>
        <option value="'Merriweather', serif">Merriweather</option>
        <option value="'Lora', serif">Lora</option>
      </select>
    </div>

    <div className="mb-4">
      <div className="mb-2 flex items-center justify-between">
        <span className="text-sm font-medium opacity-80">
          {t("reader.font_size", "Font Size")}
        </span>
        <span className="font-mono text-xs opacity-50">{fontSize}px</span>
      </div>
      <div className="flex gap-2">
        <button
          onClick={() => setFontSize((size) => Math.max(12, size - 1))}
          className="reader-control-btn btn btn-square btn-sm"
        >
          <Minus className="h-4 w-4" />
        </button>
        <input
          type="range"
          min="12"
          max="40"
          value={fontSize}
          onChange={(event) => setFontSize(parseInt(event.target.value))}
          className="range range-primary range-sm flex-1"
        />
        <button
          onClick={() => setFontSize((size) => Math.min(40, size + 1))}
          className="reader-control-btn btn btn-square btn-sm"
        >
          <Plus className="h-4 w-4" />
        </button>
      </div>
    </div>

    <div>
      <div className="mb-2 flex items-center justify-between">
        <span className="text-sm font-medium opacity-80">
          {t("reader.content_width", "Content Width")}
        </span>
      </div>
      <input
        type="range"
        min="400"
        max="1600"
        step="100"
        value={maxWidth}
        onChange={(event) => setMaxWidth(parseInt(event.target.value))}
        className="range range-primary range-sm w-full"
      />
    </div>

    <div className="mt-4 border-t border-current/10 pt-4">
      <div className="mb-2 flex items-center justify-between">
        <span className="text-sm font-medium opacity-80">
          {t("reader.mode", "Reading Mode")}
        </span>
      </div>
      <div className="grid grid-cols-2 gap-1">
        <button
          onClick={() => setReadingMode("scroll")}
          className={`reader-segment-btn btn h-auto min-h-[30px] flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${effectiveReadingMode === "scroll" ? "reader-segment-btn-active" : ""}`}
        >
          {t("reader.mode_scroll", "Scroll")}
        </button>
        <button
          onClick={() => setReadingMode("single")}
          className={`reader-segment-btn btn h-auto min-h-[30px] flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${effectiveReadingMode === "single" ? "reader-segment-btn-active" : ""}`}
        >
          {t("reader.mode_single", "Single Page")}
        </button>
        <button
          disabled={!canUseDoubleMode}
          onClick={() => setReadingMode("double")}
          className={`reader-segment-btn btn h-auto min-h-[30px] flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${effectiveReadingMode === "double" ? "reader-segment-btn-active" : ""} ${!canUseDoubleMode ? "opacity-40" : ""}`}
        >
          {t("reader.mode_double", "Double Page")}
        </button>
        <button
          onClick={() => setReadingMode("webtoon")}
          className={`reader-segment-btn btn h-auto min-h-[30px] flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${effectiveReadingMode === "webtoon" ? "reader-segment-btn-active" : ""}`}
        >
          {t("reader.mode_webtoon", "Webtoon")}
        </button>
      </div>
    </div>

    {isVisualContent && (
      <>
        <div className="mt-4 border-t border-current/10 pt-4">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-sm font-medium opacity-80">
              {t("reader.direction", "Reading Direction")}
            </span>
          </div>
          <div className="grid grid-cols-2 gap-1">
            <button
              onClick={() => setReadingDirection("ltr")}
              className={`reader-segment-btn btn h-auto min-h-[30px] flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${readingDirection === "ltr" ? "reader-segment-btn-active" : ""}`}
            >
              {t("reader.direction_ltr", "Left to right")}
            </button>
            <button
              onClick={() => setReadingDirection("rtl")}
              className={`reader-segment-btn btn h-auto min-h-[30px] flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${readingDirection === "rtl" ? "reader-segment-btn-active" : ""}`}
            >
              {t("reader.direction_rtl", "Right to left")}
            </button>
          </div>
        </div>

        <div className="mt-4 border-t border-current/10 pt-4">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-sm font-medium opacity-80">
              {t("reader.fit", "Page Fit")}
            </span>
          </div>
          <div className="grid grid-cols-3 gap-1">
            <button
              onClick={() => setPageFit("width")}
              className={`reader-segment-btn btn h-auto min-h-[30px] flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${pageFit === "width" ? "reader-segment-btn-active" : ""}`}
            >
              {t("reader.fit_width", "Width")}
            </button>
            <button
              onClick={() => setPageFit("height")}
              className={`reader-segment-btn btn h-auto min-h-[30px] flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${pageFit === "height" ? "reader-segment-btn-active" : ""}`}
            >
              {t("reader.fit_height", "Height")}
            </button>
            <button
              onClick={() => setPageFit("original")}
              className={`reader-segment-btn btn h-auto min-h-[30px] flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${pageFit === "original" ? "reader-segment-btn-active" : ""}`}
            >
              {t("reader.fit_original", "Original")}
            </button>
          </div>
        </div>
      </>
    )}

    <div className="mt-4 border-t border-current/10 pt-4">
      <button
        type="button"
        onClick={resetSettings}
        className="reader-outline-btn btn btn-sm w-full animate-none gap-2 rounded-lg"
      >
        <RotateCcw className="h-4 w-4" />
        {t("reader.reset_settings", "Reset settings")}
      </button>
    </div>
  </div>
  );
};
