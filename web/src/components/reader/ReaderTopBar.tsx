import type { TFunction } from "i18next";
import { ChevronLeft, Menu, Settings, Play, Pause, Square, ArrowDown, Volume2, Search, Headphones, MoreHorizontal, BookmarkPlus, Copy, Check, Loader2, Sparkles, MessageSquarePlus, X, Maximize, Minimize } from "lucide-react";
import React, { useState, useEffect } from "react";
import { toast } from "react-toastify";
import { copyImageToClipboard } from "@/utils/clipboard";
import { LanguageSwitcher } from "@/components/ui";
import type { PageAnimation, PageFit, ReaderTheme, ReadingDirection, ReadingMode, TextAlignment } from "@/stores";
import { useSoundscapeStore } from "@/stores/soundscapeStore";
import { ReaderSettingsPanel } from "./ReaderSettingsPanel";
import { ReaderTtsSettingsPanel } from "./ReaderTtsSettingsPanel";
import { ReaderSoundscapePanel } from "./ReaderSoundscapePanel";
import type { ActiveImageTarget, ImageBookmark } from "@/types";

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
  textAlign?: TextAlignment;
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
  setTextAlign?: (align: TextAlignment) => void;
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
  isAudio?: boolean;
  isComic?: boolean;
  comicCurrentPage?: number;
  comicTotalPages?: number;
  onComicPageJump?: (page: number) => void;

  activeImageTarget?: ActiveImageTarget | null;
  onSaveImageBookmark?: (bookmark: Omit<ImageBookmark, "id" | "created_at">) => void;
  onOpenQuoteCard?: (text?: string, imageUrl?: string) => void;
  onCloseImageTarget?: () => void;
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
  textAlign,
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
  setTextAlign,
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
  isComic = false,
  comicCurrentPage = 0,
  comicTotalPages = 0,
  onComicPageJump,
  activeImageTarget,
  onSaveImageBookmark,
  onOpenQuoteCard,
  onCloseImageTarget,
}) => {
  const [ttsSettingsOpen, setTtsSettingsOpen] = useState(false);
  const [soundscapeOpen, setSoundscapeOpen] = useState(false);
  const [moreMenuOpen, setMoreMenuOpen] = useState(false);
  const [imageDropdownOpen, setImageDropdownOpen] = useState(false);
  const [imageNote, setImageNote] = useState("");
  const [copyingImage, setCopyingImage] = useState(false);
  const [copiedImage, setCopiedImage] = useState(false);
  const [pageInputValue, setPageInputValue] = useState<string>(String(comicCurrentPage + 1));
  const { isPlaying: isSoundscapePlaying, activeTracks } = useSoundscapeStore();
  const activeSoundscapeCount = Object.keys(activeTracks).length;

  const [isFullscreen, setIsFullscreen] = useState(false);

  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(!!document.fullscreenElement);
    };
    document.addEventListener("fullscreenchange", handleFullscreenChange);
    return () => document.removeEventListener("fullscreenchange", handleFullscreenChange);
  }, []);

  const toggleFullscreen = async () => {
    try {
      if (!document.fullscreenElement) {
        await document.documentElement.requestFullscreen();
      } else {
        await document.exitFullscreen();
      }
    } catch {
      // Ignore fullscreen errors
    }
  };

  React.useEffect(() => {
    setPageInputValue(String(comicCurrentPage + 1));
  }, [comicCurrentPage]);

  useEffect(() => {
    if (!activeImageTarget) {
      setImageDropdownOpen(false);
    }
  }, [activeImageTarget]);

  return (
    <header
      className={`relative z-50 flex h-14 w-full flex-none items-center justify-between border-b px-3 sm:px-4 ${headerBg} backdrop-blur-md`}
    >
      <div className="flex items-center gap-2 min-w-0">
        <div className="tooltip tooltip-bottom" data-tip={t("reader.toc", "Contents")}>
          <label
            htmlFor="reader-drawer"
            className="reader-control-btn btn btn-square btn-sm cursor-pointer"
            aria-label={t("reader.toc", "Contents")}
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
            className={`reader-control-btn btn btn-square btn-sm animate-none ${!canGoPrev ? "opacity-30 cursor-not-allowed" : ""}`}
            aria-label={t("reader.prev_chapter", "Previous Chapter")}
          >
            <ChevronLeft className="h-5 w-5" />
          </button>
        </div>
        <div className="tooltip tooltip-bottom" data-tip={nextTooltip || t("reader.next_chapter", "Next Chapter")}>
          <button
            onClick={onNext}
            disabled={!canGoNext}
            className={`reader-control-btn btn btn-square btn-sm animate-none ${!canGoNext ? "opacity-30 cursor-not-allowed" : ""}`}
            aria-label={nextTooltip || t("reader.next_chapter", "Next Chapter")}
          >
            <ChevronLeft className="h-5 w-5 rotate-180" />
          </button>
        </div>

        {isComic && comicTotalPages > 0 && (
          <div
            className="flex items-center gap-1 bg-base-content/10 px-2 py-0.5 rounded-md text-xs font-mono font-bold select-none"
            title={t("reader.jump_to_page", "Jump to page")}
          >
            <input
              type="text"
              inputMode="numeric"
              pattern="[0-9]*"
              value={pageInputValue}
              onChange={(e) => setPageInputValue(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  const p = parseInt(pageInputValue, 10);
                  if (!isNaN(p) && p >= 1 && p <= comicTotalPages) {
                    onComicPageJump?.(p - 1);
                    (e.target as HTMLInputElement).blur();
                  } else {
                    setPageInputValue(String(comicCurrentPage + 1));
                  }
                }
              }}
              onBlur={() => {
                const p = parseInt(pageInputValue, 10);
                if (!isNaN(p) && p >= 1 && p <= comicTotalPages) {
                  onComicPageJump?.(p - 1);
                } else {
                  setPageInputValue(String(comicCurrentPage + 1));
                }
              }}
              className="w-8 sm:w-10 text-center font-bold bg-transparent border-b border-base-content/40 focus:border-primary focus:outline-hidden px-0.5 py-0"
              aria-label={t("reader.page_number", "Page Number")}
            />
            <span className="opacity-60">/ {comicTotalPages}</span>
          </div>
        )}

        {onOpenSearch && !isAudio && !isComic && (
          <div className="tooltip tooltip-bottom" data-tip={t("reader.in_book_search", "Search")}>
            <button
              onClick={onOpenSearch}
              className="reader-control-btn btn btn-square btn-sm animate-none"
              aria-label={t("reader.in_book_search", "Search")}
            >
              <Search className="h-5 w-5" />
            </button>
          </div>
        )}

        {/* Active Image Actions Button & Dropdown */}
        {activeImageTarget && (
          <div className="relative">
            <div className="tooltip tooltip-bottom" data-tip={t("reader.bookmark_image", "Bookmark image")}>
              <button
                type="button"
                onClick={() => {
                  setImageDropdownOpen(!imageDropdownOpen);
                  if (settingsOpen) setSettingsOpen(false);
                  if (ttsSettingsOpen) setTtsSettingsOpen(false);
                  if (soundscapeOpen) setSoundscapeOpen(false);
                  if (moreMenuOpen) setMoreMenuOpen(false);
                }}
                className={`reader-control-btn btn btn-square btn-sm animate-none relative ${
                  imageDropdownOpen ? "text-amber-400 reader-control-btn-active" : "text-amber-400"
                }`}
                aria-label={t("reader.bookmark_image", "Bookmark image")}
              >
                <BookmarkPlus className="h-5 w-5" />
                <span className="absolute -top-0.5 -right-0.5 flex h-2 w-2">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-amber-400 opacity-75"></span>
                  <span className="relative inline-flex rounded-full h-2 w-2 bg-amber-500"></span>
                </span>
              </button>
            </div>

            {/* Dropdown Panel with Image Preview & Bookmark/Quote Actions */}
            {imageDropdownOpen && (
              <div
                className="absolute right-0 mt-2 z-50 flex flex-col gap-2.5 rounded-2xl border border-(--reader-ui-border) bg-(--reader-ui-surface-strong) p-3 shadow-2xl backdrop-blur-md animate-in fade-in zoom-in-95 duration-100 max-w-[calc(100vw-32px)] w-80"
                onClick={(e) => e.stopPropagation()}
              >
                {/* Small Image Preview */}
                <div className="w-full flex items-center gap-2.5 p-2 rounded-xl bg-(--reader-ui-soft) border border-(--reader-ui-border)">
                  <img
                    src={activeImageTarget.image_url}
                    alt={t("common.preview")}
                    className="h-16 w-16 object-cover rounded-lg border border-(--reader-ui-border) shrink-0 bg-black/20"
                  />
                  <div className="flex-1 min-w-0 text-xs">
                    <p className="font-semibold text-(--reader-ui-text) truncate">
                      {activeImageTarget.chapter_title || title || t("reader.image_action", "Illustration image")}
                    </p>
                    {activeImageTarget.page_index !== undefined && (
                      <p className="text-[11px] text-(--reader-ui-muted) opacity-80 mt-0.5">
                        {t("reader.page_number", "Trang {{page}}", { page: activeImageTarget.page_index + 1 })}
                      </p>
                    )}
                  </div>
                  <button
                    type="button"
                    onClick={() => {
                      setImageDropdownOpen(false);
                      onCloseImageTarget?.();
                    }}
                    className="btn btn-ghost btn-xs btn-circle text-(--reader-ui-muted) hover:text-(--reader-ui-text)"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>

                {/* Actions Row */}
                <div className="flex items-center gap-1.5 w-full">
                  <button
                    type="button"
                    onClick={() => {
                      onSaveImageBookmark?.({
                        image_url: activeImageTarget.image_url,
                        chapter_id: activeImageTarget.chapter_id,
                        chapter_title: activeImageTarget.chapter_title,
                        page_index: activeImageTarget.page_index,
                        note: imageNote.trim() || undefined,
                      });
                      setImageNote("");
                      setImageDropdownOpen(false);
                    }}
                    className="btn btn-ghost btn-xs h-7 flex-1 min-w-0 px-2 rounded-xl border border-(--reader-ui-border) bg-(--reader-ui-soft) text-(--reader-ui-text) hover:bg-(--reader-ui-hover) gap-1 text-[11px] font-medium transition-colors"
                  >
                    <BookmarkPlus className="h-3.5 w-3.5 text-primary shrink-0" />
                    <span className="truncate">{t("reader.bookmark", "Bookmark")}</span>
                  </button>

                  <button
                    type="button"
                    onClick={async (e) => {
                      e.stopPropagation();
                      e.preventDefault();
                      setCopyingImage(true);
                      const success = await copyImageToClipboard(activeImageTarget.image_url);
                      setCopyingImage(false);
                      if (success) {
                        setCopiedImage(true);
                        toast.success(t("reader.image_copied", "Image copied to clipboard!"));
                        setTimeout(() => setCopiedImage(false), 2000);
                      } else {
                        toast.error(t("reader.image_copy_failed", "Failed to copy image"));
                      }
                    }}
                    disabled={copyingImage}
                    className="btn btn-ghost btn-xs h-7 flex-1 min-w-0 px-2 rounded-xl border border-(--reader-ui-border) bg-(--reader-ui-soft) text-(--reader-ui-text) hover:bg-(--reader-ui-hover) gap-1 text-[11px] font-medium transition-colors"
                  >
                    {copyingImage ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin shrink-0" />
                    ) : copiedImage ? (
                      <Check className="h-3.5 w-3.5 text-success shrink-0" />
                    ) : (
                      <Copy className="h-3.5 w-3.5 text-amber-500 shrink-0" />
                    )}
                    <span className="truncate">{copiedImage ? t("common.copied", "Copied") : t("common.copy", "Copy")}</span>
                  </button>

                  {onOpenQuoteCard && (
                    <button
                      type="button"
                      onClick={() => {
                        onOpenQuoteCard(imageNote.trim() || undefined, activeImageTarget.image_url);
                        setImageDropdownOpen(false);
                      }}
                      className="btn btn-ghost btn-xs h-7 flex-1 min-w-0 px-2 rounded-xl border border-(--reader-ui-border) bg-(--reader-ui-soft) text-(--reader-ui-text) hover:bg-(--reader-ui-hover) gap-1 text-[11px] font-medium transition-colors"
                    >
                      <Sparkles className="h-3.5 w-3.5 text-purple-400 shrink-0" />
                      <span className="truncate">{t("reader.quote", "Quote")}</span>
                    </button>
                  )}
                </div>

                {/* Note Input */}
                <div className="flex items-center gap-1.5 w-full bg-(--reader-ui-soft) rounded-xl px-2.5 py-1 border border-(--reader-ui-border)">
                  <MessageSquarePlus className="h-3.5 w-3.5 text-(--reader-ui-muted) shrink-0" />
                  <input
                    type="text"
                    value={imageNote}
                    onChange={(e) => setImageNote(e.target.value)}
                    placeholder={t("reader.add_image_note_placeholder", "Note for this image...")}
                    className="w-full bg-transparent text-xs text-(--reader-ui-text) placeholder:text-(--reader-ui-muted)/60 focus:outline-hidden py-0.5"
                  />
                </div>
              </div>
            )}
          </div>
        )}

        {/* Desktop Extra Controls (Hidden on Mobile) */}
        <div className="hidden md:flex items-center gap-1">
          {ttsSupported && !isAudio && !isComic && (
            <div className="relative flex items-center gap-1 border-r border-(--reader-ui-border) pr-2">
              <div
                className="tooltip tooltip-bottom"
                data-tip={
                  ttsPlaying
                    ? t("reader.tts_pause", "Pause")
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
                      ? t("reader.tts_pause", "Pause")
                      : t("reader.tts_play", "Read Aloud")
                  }
                >
                  {ttsPlaying ? <Pause className="h-5 w-5" /> : <Play className="h-5 w-5" />}
                </button>
              </div>
              {(ttsPlaying || ttsPaused) && (
                <div className="tooltip tooltip-bottom" data-tip={t("reader.tts_stop", "Stop")}>
                  <button
                    onClick={onTtsStop}
                    className="reader-control-btn btn btn-square btn-sm animate-none text-error"
                    aria-label={t("reader.tts_stop", "Stop")}
                  >
                    <Square className="h-5 w-5" />
                  </button>
                </div>
              )}
              <div className="tooltip tooltip-bottom" data-tip={t("reader.tts_settings", "Voice & Speed")}>
                <button
                  onClick={() => {
                    setTtsSettingsOpen(!ttsSettingsOpen);
                    if (settingsOpen) setSettingsOpen(false);
                    if (moreMenuOpen) setMoreMenuOpen(false);
                  }}
                  className={`reader-control-btn btn btn-square btn-sm animate-none ${
                    ttsSettingsOpen ? "reader-control-btn-active text-primary" : ""
                  }`}
                  aria-label={t("reader.tts_settings", "Voice & Speed")}
                >
                  <Volume2 className="h-5 w-5" />
                </button>
              </div>
            </div>
          )}

          {!isAudio && (effectiveReadingMode === "scroll" || effectiveReadingMode === "webtoon") && (
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
              aria-label={t("reader.tts_play_pause")}
            >
              {ttsPlaying ? <Pause className="h-5 w-5" /> : <Play className="h-5 w-5" />}
            </button>
            <button
              onClick={onTtsStop}
              className="reader-control-btn btn btn-square btn-sm text-error animate-none"
              aria-label={t("reader.tts_stop")}
            >
              <Square className="h-5 w-5" />
            </button>
          </div>
        )}

        {/* Mobile More Tools Menu Button (Hidden on md+) */}
        <div className="relative md:hidden">
          <div className="tooltip tooltip-bottom" data-tip={t("reader.more_tools", "Tools")}>
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
              aria-label={t("reader.more_tools", "Tools")}
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
              {ttsSupported && !isAudio && !isComic && (
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
                      aria-label={t("reader.tts_play_pause")}
                    >
                      {ttsPlaying ? <Pause className="h-3.5 w-3.5" /> : <Play className="h-3.5 w-3.5" />}
                    </button>
                    {(ttsPlaying || ttsPaused) && (
                      <button
                        type="button"
                        onClick={onTtsStop}
                        className="btn btn-circle btn-xs btn-ghost text-error"
                        aria-label={t("reader.tts_stop")}
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
                      aria-label={t("reader.tts_settings")}
                    >
                      <Settings className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
              )}

              {/* Auto Scroll Item */}
              {!isAudio && (effectiveReadingMode === "scroll" || effectiveReadingMode === "webtoon") && (
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

              {/* Fullscreen Toggle Item */}
              <button
                type="button"
                onClick={() => {
                  toggleFullscreen();
                  setMoreMenuOpen(false);
                }}
                className="flex w-full items-center justify-between rounded-xl px-3 py-2 text-sm hover:bg-current/10 transition-colors"
              >
                <span className="flex items-center gap-2.5 font-medium">
                  {isFullscreen ? <Minimize className="h-4 w-4" /> : <Maximize className="h-4 w-4" />}
                  {isFullscreen ? t("reader.exit_fullscreen", "Exit Fullscreen") : t("reader.enter_fullscreen", "Fullscreen")}
                </span>
                <span className={`text-xs font-medium ${isFullscreen ? "text-primary font-bold" : "opacity-50"}`}>
                  {isFullscreen ? t("common.on", "On") : t("common.off", "Off")}
                </span>
              </button>

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

        {/* Fullscreen Toggle Button */}
        <div
          className="tooltip tooltip-bottom"
          data-tip={isFullscreen ? t("reader.exit_fullscreen", "Exit Fullscreen") : t("reader.enter_fullscreen", "Fullscreen")}
        >
          <button
            type="button"
            onClick={toggleFullscreen}
            className={`reader-control-btn btn btn-square btn-sm animate-none ${
              isFullscreen ? "text-primary reader-control-btn-active" : ""
            }`}
            aria-label={isFullscreen ? t("reader.exit_fullscreen", "Exit Fullscreen") : t("reader.enter_fullscreen", "Fullscreen")}
          >
            {isFullscreen ? <Minimize className="h-5 w-5" /> : <Maximize className="h-5 w-5" />}
          </button>
        </div>

        {!isAudio && (
          <div className="tooltip tooltip-bottom" data-tip={t("reader.open_settings", "Settings")}>
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
              aria-label={t("reader.open_settings", "Settings")}
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
            textAlign={textAlign}
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
            setTextAlign={setTextAlign}
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
