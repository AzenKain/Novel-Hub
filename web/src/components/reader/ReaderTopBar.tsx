import type { TFunction } from "i18next";
import { ChevronLeft, Menu, Settings, Play, Pause, Square, ArrowDown, Volume2, Search, Headphones, MoreHorizontal } from "lucide-react";
import React, { useState } from "react";
import { LanguageSwitcher } from "@/components/ui";
import type { PageAnimation, PageFit, ReaderTheme, ReadingDirection, ReadingMode } from "@/stores";
import { useSoundscapeStore } from "@/stores/soundscapeStore";
import { ReaderSettingsPanel } from "./ReaderSettingsPanel";
import { ReaderTtsSettingsPanel } from "./ReaderTtsSettingsPanel";
import { ReaderSoundscapePanel } from "./ReaderSoundscapePanel";

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
  pageAnimation: PageAnimation;
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
  setPageAnimation: (anim: PageAnimation) => void;
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
  nextTooltip?: string;
  /** Audio files have no text to search, paginate, or read aloud — text-reader
   *  controls (search, TTS, auto-scroll, reading-mode settings) stay hidden. */
  isAudio?: boolean;
};

export const ReaderTopBar: React.FC<ReaderTopBarProps> = ({
  t,
  title,
  headerBg,
  canGoPrev,
  canGoNext,
  settingsOpen,
  nextTooltip,
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
  setPageAnimation,
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
  isAudio = false,
}) => {
  const [ttsSettingsOpen, setTtsSettingsOpen] = useState(false);
  const [soundscapeOpen, setSoundscapeOpen] = useState(false);
  const [moreMenuOpen, setMoreMenuOpen] = useState(false);
  const { isPlaying: isSoundscapePlaying, activeTracks } = useSoundscapeStore();
  const activeSoundscapeCount = Object.keys(activeTracks).length;

  return (
    <header
      className={`relative z-10 flex h-14 w-full flex-none items-center justify-between border-b px-3 sm:px-4 ${headerBg} backdrop-blur-md`}
    >
      <div className="flex items-center gap-2 min-w-0">
        <div className="tooltip tooltip-bottom" data-tip={t("reader.toc", "Table of Contents")}>
          <label
            htmlFor="reader-drawer"
            className="reader-control-btn btn btn-square btn-sm cursor-pointer"
            aria-label={t("reader.toc", "Table of Contents")}
          >
            <Menu className="h-5 w-5" />
          </label>
        </div>
        <span className="truncate hidden max-w-xs text-sm font-medium opacity-50 sm:inline">
          {title || t("reader.reading", "Reading")}
        </span>
      </div>

      <div className="relative flex items-center gap-1 shrink-0">
        <div className="tooltip tooltip-bottom" data-tip={t("reader.prev_chapter", "Previous Chapter")}>
          <button
            onClick={onPrev}
            disabled={!canGoPrev}
            className="reader-control-btn btn btn-square btn-sm animate-none"
            aria-label={t("reader.prev_chapter", "Previous Chapter")}
          >
            <ChevronLeft className="h-5 w-5" />
          </button>
        </div>
        <div className="tooltip tooltip-bottom" data-tip={nextTooltip || t("reader.next_chapter", "Next Chapter")}>
          <button
            onClick={onNext}
            disabled={!canGoNext}
            className="reader-control-btn btn btn-square btn-sm animate-none"
            aria-label={nextTooltip || t("reader.next_chapter", "Next Chapter")}
          >
            <ChevronLeft className="h-5 w-5 rotate-180" />
          </button>
        </div>

        {onOpenSearch && !isAudio && (
          <div className="tooltip tooltip-bottom" data-tip={t("reader.in_book_search", "Search in Book")}>
            <button
              onClick={onOpenSearch}
              className="reader-control-btn btn btn-square btn-sm animate-none"
              aria-label={t("reader.in_book_search", "Search in Book")}
            >
              <Search className="h-5 w-5" />
            </button>
          </div>
        )}

        {/* Desktop Extra Controls (Hidden on Mobile) */}
        <div className="hidden md:flex items-center gap-1">
          {ttsSupported && !isAudio && (
            <div className="relative flex items-center gap-1 border-r border-[var(--reader-ui-border)] pr-2">
              <div
                className="tooltip tooltip-bottom"
                data-tip={
                  ttsPlaying
                    ? t("reader.tts_pause", "Pause Reading")
                    : t("reader.tts_play", "Read Aloud")
                }
              >
                <button
                  onClick={onTtsPlayPause}
                  className={`reader-control-btn btn btn-square btn-sm animate-none ${
                    ttsPlaying || ttsPaused ? "text-primary" : ""
                  }`}
                  aria-label={
                    ttsPlaying
                      ? t("reader.tts_pause", "Pause Reading")
                      : t("reader.tts_play", "Read Aloud")
                  }
                >
                  {ttsPlaying ? <Pause className="h-5 w-5" /> : <Play className="h-5 w-5" />}
                </button>
              </div>
              {(ttsPlaying || ttsPaused) && (
                <div className="tooltip tooltip-bottom" data-tip={t("reader.tts_stop", "Stop Reading")}>
                  <button
                    onClick={onTtsStop}
                    className="reader-control-btn btn btn-square btn-sm animate-none text-error"
                    aria-label={t("reader.tts_stop", "Stop Reading")}
                  >
                    <Square className="h-5 w-5" />
                  </button>
                </div>
              )}
              <div className="tooltip tooltip-bottom" data-tip={t("reader.tts_settings", "Voice & Speed Settings")}>
                <button
                  onClick={() => {
                    setTtsSettingsOpen(!ttsSettingsOpen);
                    if (settingsOpen) setSettingsOpen(false);
                    if (moreMenuOpen) setMoreMenuOpen(false);
                  }}
                  className={`reader-control-btn btn btn-square btn-sm animate-none ${
                    ttsSettingsOpen ? "reader-control-btn-active text-primary" : ""
                  }`}
                  aria-label={t("reader.tts_settings", "Voice & Speed Settings")}
                >
                  <Volume2 className="h-5 w-5" />
                </button>
              </div>
            </div>
          )}

          {!isAudio && (
            <div className="tooltip tooltip-bottom" data-tip={t("reader.auto_scroll", "Auto Scroll")}>
              <button
                onClick={onToggleAutoScroll}
                className={`reader-control-btn btn btn-square btn-sm animate-none ${
                  autoScrollActive ? "text-primary" : ""
                }`}
                aria-label={t("reader.auto_scroll", "Auto Scroll")}
              >
                <ArrowDown className="h-5 w-5" />
              </button>
            </div>
          )}

          {/* Ambient Soundscapes Mixer Button */}
          <div className="tooltip tooltip-bottom" data-tip={t("soundscape.ambient_sounds", "Ambient Sounds")}>
            <button
              onClick={() => {
                setSoundscapeOpen(!soundscapeOpen);
                if (settingsOpen) setSettingsOpen(false);
                if (ttsSettingsOpen) setTtsSettingsOpen(false);
                if (moreMenuOpen) setMoreMenuOpen(false);
              }}
              className={`reader-control-btn btn btn-square btn-sm animate-none relative ${
                soundscapeOpen || isSoundscapePlaying ? "text-primary" : ""
              }`}
              aria-label={t("soundscape.ambient_sounds", "Ambient Sounds")}
            >
              <Headphones className="h-5 w-5" />
              {activeSoundscapeCount > 0 && (
                <span className="absolute -top-1 -right-1 flex h-3 w-3">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-75"></span>
                  <span className="relative inline-flex rounded-full h-3 w-3 bg-primary text-[8px] text-primary-content items-center justify-center font-bold">
                    {activeSoundscapeCount}
                  </span>
                </span>
              )}
            </button>
          </div>

          <div className="tooltip tooltip-bottom" data-tip={t("common.language", "Language")}>
            <LanguageSwitcher
              className="dropdown-end"
              buttonClassName="reader-control-btn btn btn-sm gap-1 animate-none px-2"
            />
          </div>
        </div>

        {/* Mobile Mini TTS Controller (when actively playing or paused) */}
        {ttsSupported && !isAudio && (ttsPlaying || ttsPaused) && (
          <div className="flex md:hidden items-center gap-1">
            <button
              onClick={onTtsPlayPause}
              className="reader-control-btn btn btn-square btn-sm text-primary animate-none"
              aria-label="Play/Pause TTS"
            >
              {ttsPlaying ? <Pause className="h-5 w-5" /> : <Play className="h-5 w-5" />}
            </button>
            <button
              onClick={onTtsStop}
              className="reader-control-btn btn btn-square btn-sm text-error animate-none"
              aria-label="Stop TTS"
            >
              <Square className="h-5 w-5" />
            </button>
          </div>
        )}

        {/* Mobile More Tools Menu Button (Hidden on md+) */}
        <div className="relative md:hidden">
          <div className="tooltip tooltip-bottom" data-tip={t("reader.more_tools", "Reading Tools")}>
            <button
              onClick={() => {
                setMoreMenuOpen(!moreMenuOpen);
                if (settingsOpen) setSettingsOpen(false);
                if (ttsSettingsOpen) setTtsSettingsOpen(false);
                if (soundscapeOpen) setSoundscapeOpen(false);
              }}
              className={`reader-control-btn btn btn-square btn-sm animate-none relative ${
                moreMenuOpen || isSoundscapePlaying ? "text-primary" : ""
              }`}
              aria-label={t("reader.more_tools", "Reading Tools")}
            >
              <MoreHorizontal className="h-5 w-5" />
              {activeSoundscapeCount > 0 && (
                <span className="absolute -top-1 -right-1 flex h-2.5 w-2.5">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-75"></span>
                  <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-primary"></span>
                </span>
              )}
            </button>
          </div>

          {moreMenuOpen && (
            <div className="reader-settings-panel absolute right-0 top-full z-50 mt-2 w-64 rounded-2xl border p-3 shadow-2xl space-y-1.5 animate-in fade-in duration-150">
              <h4 className="text-[11px] font-bold uppercase tracking-wider opacity-50 px-2 pt-1 pb-1">
                {t("reader.tools", "Tools & Media")}
              </h4>

              {/* Soundscape Mixer Item */}
              <button
                type="button"
                onClick={() => {
                  setSoundscapeOpen(true);
                  setMoreMenuOpen(false);
                }}
                className="flex w-full items-center justify-between rounded-xl px-3 py-2 text-sm hover:bg-current/10 transition-colors"
              >
                <span className="flex items-center gap-2.5 font-medium">
                  <Headphones className="h-4 w-4" />
                  {t("soundscape.ambient_sounds", "Ambient Sounds")}
                </span>
                {activeSoundscapeCount > 0 ? (
                  <span className="badge badge-primary badge-xs">{activeSoundscapeCount}</span>
                ) : (
                  <span className="text-xs opacity-50">{t("common.off", "Off")}</span>
                )}
              </button>

              {/* TTS Read Aloud Item */}
              {ttsSupported && !isAudio && (
                <div className="flex items-center justify-between rounded-xl px-3 py-2 text-sm hover:bg-current/10 transition-colors">
                  <span className="flex items-center gap-2.5 font-medium">
                    <Volume2 className="h-4 w-4" />
                    {t("reader.tts_read_aloud", "Read Aloud")}
                  </span>
                  <div className="flex items-center gap-1">
                    <button
                      type="button"
                      onClick={() => {
                        onTtsPlayPause?.();
                      }}
                      className={`btn btn-circle btn-xs ${ttsPlaying ? "btn-primary" : "btn-ghost"}`}
                      aria-label="Play/Pause TTS"
                    >
                      {ttsPlaying ? <Pause className="h-3.5 w-3.5" /> : <Play className="h-3.5 w-3.5" />}
                    </button>
                    {(ttsPlaying || ttsPaused) && (
                      <button
                        type="button"
                        onClick={onTtsStop}
                        className="btn btn-circle btn-xs btn-ghost text-error"
                        aria-label="Stop TTS"
                      >
                        <Square className="h-3.5 w-3.5" />
                      </button>
                    )}
                    <button
                      type="button"
                      onClick={() => {
                        setTtsSettingsOpen(true);
                        setMoreMenuOpen(false);
                      }}
                      className="btn btn-circle btn-xs btn-ghost"
                      aria-label="TTS Settings"
                    >
                      <Settings className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
              )}

              {/* Auto Scroll Item */}
              {!isAudio && (
                <button
                  type="button"
                  onClick={() => {
                    onToggleAutoScroll?.();
                    setMoreMenuOpen(false);
                  }}
                  className="flex w-full items-center justify-between rounded-xl px-3 py-2 text-sm hover:bg-current/10 transition-colors"
                >
                  <span className="flex items-center gap-2.5 font-medium">
                    <ArrowDown className="h-4 w-4" />
                    {t("reader.auto_scroll", "Auto Scroll")}
                  </span>
                  <span className={`text-xs font-medium ${autoScrollActive ? "text-primary font-bold" : "opacity-50"}`}>
                    {autoScrollActive ? t("common.on", "On") : t("common.off", "Off")}
                  </span>
                </button>
              )}

              {/* Language Switcher Item */}
              <div className="flex w-full items-center justify-between rounded-xl px-3 py-2 text-sm border-t border-current/10 pt-2">
                <span className="flex items-center gap-2.5 font-medium">
                  {t("common.language", "Language")}
                </span>
                <LanguageSwitcher
                  className="dropdown-end"
                  buttonClassName="reader-control-btn btn btn-xs gap-1 animate-none"
                />
              </div>
            </div>
          )}
        </div>

        {/* Global Panels */}
        {ttsSettingsOpen && !isAudio && (
          <ReaderTtsSettingsPanel
            t={t}
            ttsVoices={ttsVoices}
            ttsSelectedVoice={ttsSelectedVoice}
            setTtsSelectedVoice={setTtsSelectedVoice}
            ttsRate={ttsRate}
            setTtsRate={setTtsRate}
          />
        )}

        {soundscapeOpen && (
          <ReaderSoundscapePanel onClose={() => setSoundscapeOpen(false)} />
        )}

        {!isAudio && (
          <div className="tooltip tooltip-bottom" data-tip={t("reader.open_settings", "Reader Settings")}>
            <button
              onClick={() => {
                setSettingsOpen(!settingsOpen);
                if (ttsSettingsOpen) setTtsSettingsOpen(false);
                if (soundscapeOpen) setSoundscapeOpen(false);
                if (moreMenuOpen) setMoreMenuOpen(false);
              }}
              className={`reader-control-btn btn btn-square btn-sm animate-none ${
                settingsOpen ? "reader-control-btn-active" : ""
              }`}
              aria-label={t("reader.open_settings", "Reader Settings")}
            >
              <Settings className="h-5 w-5" />
            </button>
          </div>
        )}

        {settingsOpen && !isAudio && (
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
            pageAnimation={pageAnimation}
            setTheme={setTheme}
            setFontFamily={setFontFamily}
            setFontSize={setFontSize}
            setLineHeight={setLineHeight}
            setMaxWidth={setMaxWidth}
            setReadingMode={setReadingMode}
            setReadingDirection={setReadingDirection}
            setPageFit={setPageFit}
            setPageAnimation={setPageAnimation}
            resetSettings={resetSettings}
          />
        )}
      </div>
    </header>
  );
};
