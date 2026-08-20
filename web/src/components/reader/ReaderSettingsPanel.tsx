import type { TFunction } from "i18next";
import {
  AlignLeft,
  Minus,
  Moon,
  Plus,
  RotateCcw,
  Sun,
  Palette,
  Eye,
} from "lucide-react";
import React, { useEffect } from "react";

import type { PageAnimation, PageFit, ReaderTheme, ReadingDirection, ReadingMode } from "@/stores";
import { useReaderStore } from "@/stores/readerStore";
import { useCustomFontsQuery, useCustomThemesQuery } from "@/hooks/useCustomization";

type ReaderSettingsPanelProps = {
  t: TFunction;
  theme: ReaderTheme;
  fontFamily: string;
  fontSize: number;
  lineHeight: number;
  maxWidth: number;
  effectiveReadingMode: ReadingMode;
  canUseDoubleMode: boolean;
  isVisualContent: boolean;
  readingDirection: ReadingDirection;
  pageFit: PageFit;
  pageAnimation: PageAnimation;
  setTheme: (theme: ReaderTheme) => void;
  setFontFamily: (family: string) => void;
  setFontSize: (size: number | ((prev: number) => number)) => void;
  setLineHeight: (height: number | ((prev: number) => number)) => void;
  setMaxWidth: (width: number | ((prev: number) => number)) => void;
  setReadingMode: (mode: ReadingMode) => void;
  setReadingDirection: (direction: ReadingDirection) => void;
  setPageFit: (fit: PageFit) => void;
  setPageAnimation: (anim: PageAnimation) => void;
  resetSettings: () => void;
};

export const ReaderSettingsPanel: React.FC<ReaderSettingsPanelProps> = ({
  t,
  theme,
  fontFamily,
  fontSize,
  lineHeight,
  maxWidth,
  effectiveReadingMode,
  canUseDoubleMode,
  isVisualContent,
  readingDirection,
  pageFit,
  pageAnimation,
  setTheme,
  setFontFamily,
  setFontSize,
  setLineHeight,
  setMaxWidth,
  setReadingMode,
  setReadingDirection,
  setPageFit,
  setPageAnimation,
  resetSettings,
}) => {
  const {
    customBg,
    customText,
    customAccent,
    customCss,
    setCustomThemeColors,
  } = useReaderStore();

  const { data: customFonts = [] } = useCustomFontsQuery();
  const { data: customThemes = [] } = useCustomThemesQuery();

  const maxContentWidth = typeof window !== "undefined"
    ? Math.max(600, Math.min(1800, Math.floor((window.innerWidth - (window.innerWidth < 640 ? 32 : 80)) / 50) * 50))
    : 1600;

  // Inject custom fonts dynamically into document head
  useEffect(() => {
    customFonts.forEach((f) => {
      if (f.source_type === "url" && f.font_url) {
        const linkId = `custom-font-link-${f.id}`;
        if (!document.getElementById(linkId)) {
          const link = document.createElement("link");
          link.id = linkId;
          link.rel = "stylesheet";
          link.href = f.font_url;
          document.head.appendChild(link);
        }
      } else if (f.file_url) {
        const styleId = `custom-font-face-${f.id}`;
        if (!document.getElementById(styleId)) {
          const style = document.createElement("style");
          style.id = styleId;
          style.textContent = `@font-face { font-family: '${f.font_family}'; src: url('${f.file_url}'); }`;
          document.head.appendChild(style);
        }
      }
    });
  }, [customFonts]);

  // Sync custom theme CSS variables to document root when custom theme is active
  useEffect(() => {
    if (theme === "custom") {
      document.documentElement.style.setProperty("--custom-reader-bg", customBg);
      document.documentElement.style.setProperty("--custom-reader-text", customText);
      document.documentElement.style.setProperty("--custom-reader-accent", customAccent);
    }
  }, [theme, customBg, customText, customAccent]);

  return (
    <div className="reader-settings-panel absolute right-0 top-full z-50 mt-2 max-h-[calc(100vh-5rem)] w-84 sm:w-96 md:w-[440px] overflow-y-auto rounded-2xl border p-5 shadow-2xl transition-colors duration-300">
      <h3 className="mb-4 text-xs font-bold uppercase tracking-wider opacity-50">
        {t("reader.settings", "Reader Settings")}
      </h3>

      {/* Themes Grid */}
      <div className="mb-5">
        <div className="mb-2 flex items-center justify-between">
          <span className="text-sm font-medium opacity-80">
            {t("reader.theme", "Theme & Appearance")}
          </span>
        </div>
        <div className="grid grid-cols-4 gap-2">
          <button
            onClick={() => setTheme("light")}
            className={`reader-theme-choice flex flex-col items-center justify-center gap-1 rounded-xl border py-2 px-1 transition-all ${
              theme === "light" ? "reader-theme-choice-active" : ""
            }`}
          >
            <Sun className="h-4 w-4" />
            <span className="text-[10px] font-medium">{t("reader.theme_light", "Light")}</span>
          </button>
          <button
            onClick={() => setTheme("sepia")}
            className={`reader-theme-choice flex flex-col items-center justify-center gap-1 rounded-xl border py-2 px-1 transition-all ${
              theme === "sepia" ? "reader-theme-choice-active" : ""
            }`}
          >
            <AlignLeft className="h-4 w-4" />
            <span className="text-[10px] font-medium">{t("reader.theme_sepia", "Sepia")}</span>
          </button>
          <button
            onClick={() => setTheme("warm")}
            className={`reader-theme-choice flex flex-col items-center justify-center gap-1 rounded-xl border py-2 px-1 transition-all ${
              theme === "warm" ? "reader-theme-choice-active" : ""
            }`}
          >
            <span className="text-xs font-bold text-amber-700 dark:text-amber-300">W</span>
            <span className="text-[10px] font-medium">{t("reader.theme_warm", "Warm")}</span>
          </button>
          <button
            onClick={() => setTheme("coffee")}
            className={`reader-theme-choice flex flex-col items-center justify-center gap-1 rounded-xl border py-2 px-1 transition-all ${
              theme === "coffee" ? "reader-theme-choice-active" : ""
            }`}
          >
            <span className="text-xs font-bold text-amber-900 dark:text-amber-200">C</span>
            <span className="text-[10px] font-medium">{t("common.coffee", "Coffee")}</span>
          </button>
          <button
            onClick={() => setTheme("dark")}
            className={`reader-theme-choice flex flex-col items-center justify-center gap-1 rounded-xl border py-2 px-1 transition-all ${
              theme === "dark" ? "reader-theme-choice-active" : ""
            }`}
          >
            <Moon className="h-4 w-4" />
            <span className="text-[10px] font-medium">{t("reader.theme_dark", "Dark")}</span>
          </button>
          <button
            onClick={() => setTheme("dim")}
            className={`reader-theme-choice flex flex-col items-center justify-center gap-1 rounded-xl border py-2 px-1 transition-all ${
              theme === "dim" ? "reader-theme-choice-active" : ""
            }`}
          >
            <span className="text-xs font-bold opacity-75">D</span>
            <span className="text-[10px] font-medium">{t("reader.theme_dim", "Dim")}</span>
          </button>
          <button
            onClick={() => setTheme("eink")}
            className={`reader-theme-choice flex flex-col items-center justify-center gap-1 rounded-xl border py-2 px-1 transition-all ${
              theme === "eink" ? "reader-theme-choice-active" : ""
            }`}
            title="1-bit Pure E-Ink high contrast mode (0ms transitions, no shadows)"
          >
            <Eye className="h-4 w-4" />
            <span className="text-[10px] font-medium">E-Ink</span>
          </button>
          <button
            onClick={() => setTheme("custom")}
            className={`reader-theme-choice flex flex-col items-center justify-center gap-1 rounded-xl border py-2 px-1 transition-all ${
              theme === "custom" ? "reader-theme-choice-active" : ""
            }`}
          >
            <Palette className="h-4 w-4" />
            <span className="text-[10px] font-medium">{t("reader.theme_custom", "Custom")}</span>
          </button>
        </div>

        {/* Custom Theme Customizer Drawer */}
        {theme === "custom" && (
          <div className="mt-3 p-3 rounded-xl bg-base-200/50 border border-current/10 space-y-3 animate-in fade-in duration-200">
            {customThemes.length > 0 && (
              <div>
                <label className="text-[11px] font-semibold opacity-70 block mb-1">
                  {t("reader.saved_themes", "Saved Custom Themes")}:
                </label>
                <div className="flex flex-wrap gap-1.5">
                  {customThemes.map((th) => (
                    <button
                      key={th.id}
                      type="button"
                      onClick={() =>
                        setCustomThemeColors(th.bg_color, th.text_color, th.accent_color, th.custom_css)
                      }
                      className="btn btn-xs btn-ghost border border-current/20 rounded-lg text-[11px]"
                      style={{
                        backgroundColor: th.bg_color,
                        color: th.text_color,
                        borderColor: th.accent_color,
                      }}
                    >
                      {th.name}
                    </button>
                  ))}
                </div>
              </div>
            )}

            <div className="grid grid-cols-3 gap-2">
              <div>
                <label className="text-[10px] font-semibold opacity-70 block mb-1">
                  {t("reader.bg_color", "Background")}
                </label>
                <div className="flex items-center gap-1">
                  <input
                    type="color"
                    value={customBg}
                    onChange={(e) => setCustomThemeColors(e.target.value, customText, customAccent, customCss)}
                    className="w-7 h-7 rounded border border-current/20 cursor-pointer p-0"
                  />
                  <span className="font-mono text-[10px] opacity-70 uppercase">{customBg}</span>
                </div>
              </div>
              <div>
                <label className="text-[10px] font-semibold opacity-70 block mb-1">
                  {t("reader.text_color", "Text")}
                </label>
                <div className="flex items-center gap-1">
                  <input
                    type="color"
                    value={customText}
                    onChange={(e) => setCustomThemeColors(customBg, e.target.value, customAccent, customCss)}
                    className="w-7 h-7 rounded border border-current/20 cursor-pointer p-0"
                  />
                  <span className="font-mono text-[10px] opacity-70 uppercase">{customText}</span>
                </div>
              </div>
              <div>
                <label className="text-[10px] font-semibold opacity-70 block mb-1">
                  {t("reader.accent_color", "Accent")}
                </label>
                <div className="flex items-center gap-1">
                  <input
                    type="color"
                    value={customAccent}
                    onChange={(e) => setCustomThemeColors(customBg, customText, e.target.value, customCss)}
                    className="w-7 h-7 rounded border border-current/20 cursor-pointer p-0"
                  />
                  <span className="font-mono text-[10px] opacity-70 uppercase">{customAccent}</span>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Font Family Selection */}
      <div className="mb-4">
        <div className="mb-2 flex items-center justify-between">
          <span className="text-sm font-medium opacity-80">
            {t("reader.font_family", "Font Family")}
          </span>
          <span className="text-[11px] opacity-50">
            {customFonts.length > 0
              ? t("reader.custom_fonts_available", "{{count}} custom fonts", { count: customFonts.length })
              : ""}
          </span>
        </div>
        <select
          className="reader-select select select-bordered select-sm w-full"
          value={fontFamily}
          onChange={(event) => setFontFamily(event.target.value)}
        >
          <optgroup label={t("reader.standard_fonts", "Standard Fonts")}>
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
          </optgroup>

          {customFonts.length > 0 && (
            <optgroup label={t("reader.custom_fonts", "Uploaded & Cloud Fonts")}>
              {customFonts.map((f) => (
                <option key={f.id} value={`'${f.font_family}', sans-serif`}>
                  {f.name} {f.is_system ? `(${t("common.system", "System")})` : ""}
                </option>
              ))}
            </optgroup>
          )}
        </select>
      </div>

      {/* Font Size */}
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

      {/* Line Height */}
      <div className="mb-4">
        <div className="mb-2 flex items-center justify-between">
          <span className="text-sm font-medium opacity-80">
            {t("reader.line_height", "Line Height")}
          </span>
          <span className="font-mono text-xs opacity-50">{lineHeight.toFixed(1)}</span>
        </div>
        <input
          type="range"
          min="1.2"
          max="2.5"
          step="0.1"
          value={lineHeight}
          onChange={(event) => setLineHeight(parseFloat(event.target.value))}
          className="range range-primary range-sm w-full"
        />
      </div>

      {/* Content Width */}
      <div>
        <div className="mb-2 flex items-center justify-between">
          <span className="text-sm font-medium opacity-80">
            {t("reader.content_width", "Content Width")}
          </span>
          <span className="font-mono text-xs opacity-50">
            {maxWidth >= maxContentWidth ? `${t("common.full", "Full")} (${maxContentWidth}px)` : `${maxWidth}px`}
          </span>
        </div>
        <input
          type="range"
          min="400"
          max={maxContentWidth}
          step="50"
          value={Math.min(maxWidth, maxContentWidth)}
          onChange={(event) => setMaxWidth(parseInt(event.target.value))}
          className="range range-primary range-sm w-full"
        />
      </div>

      {/* Reading Mode */}
      <div className="mt-4 border-t border-current/10 pt-4">
        <div className="mb-2 flex items-center justify-between">
          <span className="text-sm font-medium opacity-80">
            {t("reader.mode", "Reading Mode")}
          </span>
        </div>
        <div className="grid grid-cols-2 gap-1">
          <button
            onClick={() => setReadingMode("scroll")}
            className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${
              effectiveReadingMode === "scroll" ? "reader-segment-btn-active" : ""
            }`}
          >
            {t("reader.mode_scroll", "Scroll")}
          </button>
          <button
            onClick={() => setReadingMode("single")}
            className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${
              effectiveReadingMode === "single" ? "reader-segment-btn-active" : ""
            }`}
          >
            {t("reader.mode_single", "Single Page")}
          </button>
          <button
            disabled={!canUseDoubleMode}
            onClick={() => setReadingMode("double")}
            title={!canUseDoubleMode ? t("reader.double_page_unavailable") : undefined}
            className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${
              effectiveReadingMode === "double" ? "reader-segment-btn-active" : ""
            } ${!canUseDoubleMode ? "opacity-40" : ""}`}
          >
            {t("reader.mode_double", "Double Page")}
          </button>
          <button
            onClick={() => setReadingMode("webtoon")}
            className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${
              effectiveReadingMode === "webtoon" ? "reader-segment-btn-active" : ""
            }`}
          >
            {t("reader.mode_webtoon", "Webtoon")}
          </button>
        </div>
      </div>

      {/* Page Animation */}
      {(effectiveReadingMode === "single" || effectiveReadingMode === "double") && (
        <div className="mt-4 border-t border-current/10 pt-4">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-sm font-medium opacity-80">
              {t("reader.page_animation", "Page Animation")}
            </span>
          </div>
          <div className="grid grid-cols-2 gap-1">
            <button
              onClick={() => setPageAnimation("eink")}
              className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${
                pageAnimation === "eink" ? "reader-segment-btn-active" : ""
              }`}
            >
              {t("reader.animation_eink", "E-Ink Flash")}
            </button>
            <button
              onClick={() => setPageAnimation("none")}
              className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${
                pageAnimation === "none" ? "reader-segment-btn-active" : ""
              }`}
            >
              {t("reader.animation_none", "Instant / None")}
            </button>
            <button
              onClick={() => setPageAnimation("fade")}
              className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${
                pageAnimation === "fade" ? "reader-segment-btn-active" : ""
              }`}
            >
              {t("reader.animation_fade", "Fade")}
            </button>
            <button
              onClick={() => setPageAnimation("slide")}
              className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${
                pageAnimation === "slide" ? "reader-segment-btn-active" : ""
              }`}
            >
              {t("reader.animation_slide", "Slide")}
            </button>
          </div>
        </div>
      )}

      {/* Visual Content Direction & Fit */}
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
                className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${
                  readingDirection === "ltr" ? "reader-segment-btn-active" : ""
                }`}
              >
                {t("reader.direction_ltr", "Left to right")}
              </button>
              <button
                onClick={() => setReadingDirection("rtl")}
                className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${
                  readingDirection === "rtl" ? "reader-segment-btn-active" : ""
                }`}
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
                className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${
                  pageFit === "width" ? "reader-segment-btn-active" : ""
                }`}
              >
                {t("reader.fit_width", "Width")}
              </button>
              <button
                onClick={() => setPageFit("height")}
                className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${
                  pageFit === "height" ? "reader-segment-btn-active" : ""
                }`}
              >
                {t("reader.fit_height", "Height")}
              </button>
              <button
                onClick={() => setPageFit("original")}
                className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${
                  pageFit === "original" ? "reader-segment-btn-active" : ""
                }`}
              >
                {t("reader.fit_original", "Original")}
              </button>
            </div>
          </div>
        </>
      )}

      {/* Reset */}
      <div className="mt-5 border-t border-current/10 pt-4">
        <button
          type="button"
          onClick={resetSettings}
          className="reader-outline-btn btn btn-sm w-full animate-none gap-2 rounded-xl"
        >
          <RotateCcw className="h-4 w-4" />
          {t("reader.reset_settings", "Reset settings")}
        </button>
      </div>
    </div>
  );
};
