import type { TFunction } from "i18next";
import { ChevronLeft, Menu, Settings, Play, Pause, Square, ArrowDown, Volume2, Search } from "lucide-react";
import React, { useState } from "react";
import { LanguageSwitcher } from "@/components/ui";
import type { PageFit, ReaderTheme, ReadingDirection, ReadingMode } from "@/stores";
import { ReaderSettingsPanel } from "./ReaderSettingsPanel";
import { ReaderTtsSettingsPanel } from "./ReaderTtsSettingsPanel";

type ReaderTopBarProps = {
  t: TFunction;
  title: string;
  headerBg: string;
  canGoPrev: boolean;
  canGoNext: boolean;
  settingsOpen: boolean;
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
  onPrev: () => void;
  onNext: () => void;
  setSettingsOpen: (open: boolean) => void;
  setTheme: (theme: ReaderTheme) => void;
  setFontFamily: (family: string) => void;
  setFontSize: (size: number | ((prev: number) => number)) => void;
  setLineHeight: (height: number | ((prev: number) => number)) => void;
  setMaxWidth: (width: number | ((prev: number) => number)) => void;
  setReadingMode: (mode: ReadingMode) => void;
  setReadingDirection: (direction: ReadingDirection) => void;
  setPageFit: (fit: PageFit) => void;
  resetSettings: () => void;
  ttsSupported?: boolean;
  ttsPlaying?: boolean;
  ttsPaused?: boolean;
  onTtsPlayPause?: () => void;
  onTtsStop?: () => void;
  ttsVoices?: SpeechSynthesisVoice[];
  ttsSelectedVoice?: SpeechSynthesisVoice | null;
  setTtsSelectedVoice?: (voice: SpeechSynthesisVoice | null) => void;
  ttsRate?: number;
  setTtsRate?: (rate: number) => void;

  autoScrollActive?: boolean;
  onToggleAutoScroll?: () => void;
  onOpenSearch?: () => void;
};

export const ReaderTopBar: React.FC<ReaderTopBarProps> = ({
  t,
  title,
  headerBg,
  canGoPrev,
  canGoNext,
  settingsOpen,
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
  onPrev,
  onNext,
  setSettingsOpen,
  setTheme,
  setFontFamily,
  setFontSize,
  setLineHeight,
  setMaxWidth,
  setReadingMode,
  setReadingDirection,
  setPageFit,
  resetSettings,
  ttsSupported,
  ttsPlaying,
  ttsPaused,
  onTtsPlayPause,
  onTtsStop,
  ttsVoices,
  ttsSelectedVoice,
  setTtsSelectedVoice,
  ttsRate,
  setTtsRate,
  autoScrollActive,
  onToggleAutoScroll,
  onOpenSearch,
}) => {
  const [ttsSettingsOpen, setTtsSettingsOpen] = useState(false);

  return (
    <header
      className={`relative z-10 flex h-14 w-full flex-none items-center justify-between border-b px-4 ${headerBg} backdrop-blur-md`}
    >
      <div className="flex items-center gap-2">
        <label
          htmlFor="reader-drawer"
          className="reader-control-btn btn btn-square btn-sm cursor-pointer"
        >
          <Menu className="h-5 w-5" />
        </label>
        <span className="line-clamp-1 hidden max-w-xs text-sm font-medium opacity-50 sm:inline">
          {title || t("reader.reading", "Reading")}
        </span>
      </div>

      <div className="relative flex items-center gap-1">
        <button
          onClick={onPrev}
          disabled={!canGoPrev}
          className="reader-control-btn btn btn-square btn-sm animate-none"
          title={t("reader.prev_chapter", "Previous Chapter")}
        >
          <ChevronLeft className="h-5 w-5" />
        </button>
        <button
          onClick={onNext}
          disabled={!canGoNext}
          className="reader-control-btn btn btn-square btn-sm animate-none"
          title={t("reader.next_chapter", "Next Chapter")}
        >
          <ChevronLeft className="h-5 w-5 rotate-180" />
        </button>

        {onOpenSearch && (
          <button
            onClick={onOpenSearch}
            className="reader-control-btn btn btn-square btn-sm animate-none"
            title={t("reader.in_book_search", "Search in Book")}
          >
            <Search className="h-5 w-5" />
          </button>
        )}

        {ttsSupported && (
          <div className="relative flex items-center gap-1 border-r border-[var(--reader-ui-border)] pr-2">
            <button
              onClick={onTtsPlayPause}
              className={`reader-control-btn btn btn-square btn-sm animate-none ${
                ttsPlaying || ttsPaused ? "text-primary" : ""
              }`}
              title={
                ttsPlaying
                  ? t("reader.tts_pause", "Pause Reading")
                  : t("reader.tts_play", "Read Aloud")
              }
            >
              {ttsPlaying ? <Pause className="h-5 w-5" /> : <Play className="h-5 w-5" />}
            </button>
            {(ttsPlaying || ttsPaused) && (
              <button
                onClick={onTtsStop}
                className="reader-control-btn btn btn-square btn-sm animate-none text-error"
                title={t("reader.tts_stop", "Stop Reading")}
              >
                <Square className="h-5 w-5" />
              </button>
            )}
            <button
              onClick={() => {
                setTtsSettingsOpen(!ttsSettingsOpen);
                if (settingsOpen) setSettingsOpen(false);
              }}
              className={`reader-control-btn btn btn-square btn-sm animate-none ${
                ttsSettingsOpen ? "reader-control-btn-active text-primary" : ""
              }`}
              title={t("reader.tts_settings", "Voice & Speed Settings")}
            >
              <Volume2 className="h-5 w-5" />
            </button>

            {ttsSettingsOpen && (
              <ReaderTtsSettingsPanel
                t={t}
                ttsVoices={ttsVoices}
                ttsSelectedVoice={ttsSelectedVoice}
                setTtsSelectedVoice={setTtsSelectedVoice}
                ttsRate={ttsRate}
                setTtsRate={setTtsRate}
              />
            )}
          </div>
        )}

        <button
          onClick={onToggleAutoScroll}
          className={`reader-control-btn btn btn-square btn-sm animate-none ${
            autoScrollActive ? "text-primary" : ""
          }`}
          title={t("reader.auto_scroll", "Auto Scroll")}
        >
          <ArrowDown className="h-5 w-5" />
        </button>

        <LanguageSwitcher className="dropdown-end" />

        <button
          onClick={() => {
            setSettingsOpen(!settingsOpen);
            if (ttsSettingsOpen) setTtsSettingsOpen(false);
          }}
          className={`reader-control-btn btn btn-square btn-sm animate-none ${
            settingsOpen ? "reader-control-btn-active" : ""
          }`}
          title={t("reader.open_settings")}
          aria-label={t("reader.open_settings")}
        >
          <Settings className="h-5 w-5" />
        </button>

        {settingsOpen && (
          <ReaderSettingsPanel
            t={t}
            theme={theme}
            fontFamily={fontFamily}
            fontSize={fontSize}
            lineHeight={lineHeight}
            maxWidth={maxWidth}
            effectiveReadingMode={effectiveReadingMode}
            canUseDoubleMode={canUseDoubleMode}
            isVisualContent={isVisualContent}
            readingDirection={readingDirection}
            pageFit={pageFit}
            setTheme={setTheme}
            setFontFamily={setFontFamily}
            setFontSize={setFontSize}
            setLineHeight={setLineHeight}
            setMaxWidth={setMaxWidth}
            setReadingMode={setReadingMode}
            setReadingDirection={setReadingDirection}
            setPageFit={setPageFit}
            resetSettings={resetSettings}
          />
        )}
      </div>
    </header>
  );
};

