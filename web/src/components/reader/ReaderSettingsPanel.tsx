import type { TFunction } from "i18next";
import {
  AlignCenter,
  AlignLeft,
  AlignRight,
  BookText,
  Minus,
  Moon,
  Plus,
  RotateCcw,
  Sun,
  Palette,
  Eye,
  SunMoon,
} from "lucide-react";
import React, { useEffect } from "react";

import type { PageAnimation, PageFit, ReaderTheme, ReadingDirection, ReadingMode, TextAlignment } from "@/stores";
import { useReaderStore } from "@/stores/readerStore";
import { useCustomFontsQuery, useCustomThemesQuery } from "@/hooks/useCustomization";

type ReaderSettingsPanelProps = {
  t: TFunction;
  theme: ReaderTheme;
  fontFamily: string;
  fontSize: number;
  lineHeight: number;
  maxWidth: number;
  textAlign?: TextAlignment;
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
  setTextAlign?: (align: TextAlignment) => void;
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
  textAlign = "justify",
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
  setTextAlign,
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
    comicInvertColors,
    setComicInvertColors,
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

  // Visual/Comic Reader Settings (Comic / Manga / CBZ)
  if (isVisualContent) {
    return (
      <div className="reader-settings-panel absolute right-0 top-full z-50 mt-2 max-h-[calc(100vh-5rem)] w-84 sm:w-96 md:w-[440px] overflow-y-auto rounded-2xl border p-5 shadow-2xl transition-colors duration-300">
        <h3 className="mb-4 text-xs font-bold uppercase tracking-wider opacity-50">
          {t("reader.comic_settings", "Comic / Manga Settings")}
        </h3>

        {/* Themes Grid for Comic Reader Background */}
        <div className="mb-5">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-sm font-medium opacity-80">
              {t("reader.theme", "Theme & Background")}
            </span>
          </div>
          <div className="grid grid-cols-4 gap-2">
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
              <span className="text-xs font-bold">W</span>
              <span className="text-[10px] font-medium">{t("reader.theme_warm", "Warm")}</span>
            </button>
            <button
              onClick={() => setTheme("coffee")}
              className={`reader-theme-choice flex flex-col items-center justify-center gap-1 rounded-xl border py-2 px-1 transition-all ${
                theme === "coffee" ? "reader-theme-choice-active" : ""
              }`}
            >
              <span className="text-xs font-bold">C</span>
              <span className="text-[10px] font-medium">{t("common.coffee", "Coffee")}</span>
            </button>
            <button
              onClick={() => setTheme("eink")}
              className={`reader-theme-choice flex flex-col items-center justify-center gap-1 rounded-xl border py-2 px-1 transition-all ${
                theme === "eink" ? "reader-theme-choice-active" : ""
              }`}
              title={t("reader.eink_title")}
            >
              <Eye className="h-4 w-4" />
              <span className="text-[10px] font-medium">{t("reader.eink")}</span>
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
        </div>

        {/* Content Width / Page Width Slider for Manga */}
        <div className="mb-4">
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

        {/* Reading Mode for Comics */}
        <div className="mt-4 border-t border-current/10 pt-4">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-sm font-medium opacity-80">
              {t("reader.mode", "Reading Mode")}
            </span>
          </div>
          <div className="grid grid-cols-3 gap-1">
            <button
              onClick={() => setReadingMode("webtoon")}
              className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${
                effectiveReadingMode === "webtoon" || effectiveReadingMode === "scroll" ? "reader-segment-btn-active" : ""
              }`}
            >
              {t("reader.mode_webtoon", "Webtoon")}
            </button>
            <button
              onClick={() => setReadingMode("single")}
              className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${
                effectiveReadingMode === "single" ? "reader-segment-btn-active" : ""
              }`}
            >
              {t("reader.mode_single", "1 Page")}
            </button>
            <button
              disabled={!canUseDoubleMode}
              onClick={() => setReadingMode("double")}
              title={!canUseDoubleMode ? t("reader.double_page_unavailable") : undefined}
              className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight ${
                effectiveReadingMode === "double" ? "reader-segment-btn-active" : ""
              } ${!canUseDoubleMode ? "opacity-40" : ""}`}
            >
              {t("reader.mode_double", "2 Pages")}
            </button>
          </div>
        </div>

        {/* Invert Color (Night Manga reading) */}
        <div className="mt-4 border-t border-current/10 pt-4">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-sm font-medium opacity-80">
              {t("reader.night_mode", "Night Mode")}
            </span>
          </div>
          <button
            type="button"
            onClick={() => setComicInvertColors(!comicInvertColors)}
            className={`reader-segment-btn btn h-auto min-h-9 w-full flex items-center justify-between rounded-xl px-3 py-2 text-xs font-semibold ${
              comicInvertColors ? "btn-warning text-warning-content shadow-md" : ""
            }`}
          >
            <div className="flex items-center gap-2">
              <SunMoon className="h-4 w-4" />
              <span>{t("reader.invert_colors", "Invert Colors (Night Manga)")}</span>
            </div>
            <span className="badge badge-sm">{comicInvertColors ? t("common.on", "ON") : t("common.off", "OFF")}</span>
          </button>
        </div>

        {/* Reading Direction (Only for paged modes: single & double) */}
        {(effectiveReadingMode === "single" || effectiveReadingMode === "double") && (
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
        )}

        {/* Page Fit (For Single/Double Page Modes) */}
        {effectiveReadingMode !== "webtoon" && effectiveReadingMode !== "scroll" && (
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
        )}

        {/* Page Animation */}
        {effectiveReadingMode !== "webtoon" && effectiveReadingMode !== "scroll" && (
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
  }

  // Standard Text Novel / EBook Reader Settings
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
            <span className="text-xs font-bold">W</span>
            <span className="text-[10px] font-medium">{t("reader.theme_warm", "Warm")}</span>
          </button>
          <button
            onClick={() => setTheme("coffee")}
            className={`reader-theme-choice flex flex-col items-center justify-center gap-1 rounded-xl border py-2 px-1 transition-all ${
              theme === "coffee" ? "reader-theme-choice-active" : ""
            }`}
          >
            <span className="text-xs font-bold">C</span>
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
            title={t("reader.eink_title_fast")}
          >
            <Eye className="h-4 w-4" />
            <span className="text-[10px] font-medium">{t("reader.eink")}</span>
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

      {/* Font Family */}
      <div className="mb-4">
        <label className="mb-2 block text-sm font-medium opacity-80">
          <div className="flex items-center justify-between">
            <span>{t("reader.font_family", "Font Family")}</span>
            {customFonts.length > 0 && (
              <span className="text-[11px] opacity-50 font-normal">
                {t("reader.custom_fonts_count", "{{count}} custom fonts", { count: customFonts.length })}
              </span>
            )}
          </div>
        </label>
        <select
          value={fontFamily}
          onChange={(event) => setFontFamily(event.target.value)}
          className="reader-select select select-sm w-full rounded-xl"
        >
          <optgroup label={t("reader.system_fonts", "System & Standard Fonts")}>
            <option value="'Lora', serif">Lora (Serif)</option>
            <option value="'Merriweather', serif">Merriweather (Serif)</option>
            <option value="'Inter', sans-serif">Inter (Sans)</option>
            <option value="'Roboto', sans-serif">Roboto (Sans)</option>
            <option value="'OpenDyslexic', sans-serif">OpenDyslexic (Dyslexia-friendly)</option>
            <option value="'JetBrains Mono', monospace">JetBrains Mono (Monospace)</option>
          </optgroup>
          {customFonts.length > 0 && (
            <optgroup label={t("reader.custom_fonts", "Custom Fonts")}>
              {customFonts.map((f) => (
                <option key={f.id} value={`'${f.font_family}', sans-serif`}>
                  {f.name} ({f.font_family})
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
        <div className="flex items-center gap-3">
          <button
            onClick={() => setFontSize((prev) => Math.max(12, prev - 1))}
            className="reader-icon-btn btn btn-circle btn-xs"
            aria-label={t("reader.decrease_font_size", "Decrease font size")}
          >
            <Minus className="h-3.5 w-3.5" />
          </button>
          <input
            type="range"
            min="12"
            max="32"
            value={fontSize}
            onChange={(event) => setFontSize(parseInt(event.target.value))}
            className="range range-primary range-sm flex-1"
          />
          <button
            onClick={() => setFontSize((prev) => Math.min(32, prev + 1))}
            className="reader-icon-btn btn btn-circle btn-xs"
            aria-label={t("reader.increase_font_size", "Increase font size")}
          >
            <Plus className="h-3.5 w-3.5" />
          </button>
        </div>
      </div>

      {/* Line Height */}
      <div className="mb-4">
        <div className="mb-2 flex items-center justify-between">
          <span className="text-sm font-medium opacity-80">
            {t("reader.line_height", "Line Height")}
          </span>
          <span className="font-mono text-xs opacity-50">{lineHeight}</span>
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
      <div className="mb-4">
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

      {/* Text Alignment (Dàn trang / Căn lề) */}
      <div className="mb-4">
        <div className="mb-2 flex items-center justify-between">
          <span className="text-sm font-medium opacity-80">
            {t("reader.text_align", "Text Alignment")}
          </span>
        </div>
        <div className="grid grid-cols-4 gap-1">
          <button
            type="button"
            onClick={() => setTextAlign?.("original")}
            className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center gap-1 rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight cursor-pointer ${
              textAlign === "original" ? "reader-segment-btn-active" : ""
            }`}
            title={t("reader.align_original", "Publisher Default")}
          >
            <BookText className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">{t("reader.align_original_short", "Original")}</span>
          </button>
          <button
            type="button"
            onClick={() => setTextAlign?.("left")}
            className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center gap-1 rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight cursor-pointer ${
              textAlign === "left" ? "reader-segment-btn-active" : ""
            }`}
            title={t("reader.align_left", "Align Left")}
          >
            <AlignLeft className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">{t("reader.align_left_short", "Left")}</span>
          </button>
          <button
            type="button"
            onClick={() => setTextAlign?.("center")}
            className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center gap-1 rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight cursor-pointer ${
              textAlign === "center" ? "reader-segment-btn-active" : ""
            }`}
            title={t("reader.align_center", "Align Center")}
          >
            <AlignCenter className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">{t("reader.align_center_short", "Center")}</span>
          </button>
          <button
            type="button"
            onClick={() => setTextAlign?.("right")}
            className={`reader-segment-btn btn h-auto min-h-7.5 flex items-center justify-center gap-1 rounded-lg px-1 py-1.5 text-center text-[11px] leading-tight cursor-pointer ${
              textAlign === "right" ? "reader-segment-btn-active" : ""
            }`}
            title={t("reader.align_right", "Align Right")}
          >
            <AlignRight className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">{t("reader.align_right_short", "Right")}</span>
          </button>
        </div>
      </div>

      {/* Reading Mode for Novels */}
      <div className="mt-4 border-t border-current/10 pt-4">
        <div className="mb-2 flex items-center justify-between">
          <span className="text-sm font-medium opacity-80">
            {t("reader.mode", "Reading Mode")}
          </span>
        </div>
        <div className="grid grid-cols-3 gap-1">
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
        </div>
      </div>

      {/* Reading Direction (Only for paged modes: single & double) */}
      {(effectiveReadingMode === "single" || effectiveReadingMode === "double") && (
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
      )}

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
