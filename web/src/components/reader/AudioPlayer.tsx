import { useTranslation } from "react-i18next";
import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Bookmark,
  BookmarkPlus,
  Check,
  Gauge,
  ListMusic,
  Moon,
  Pause,
  Play,
  Plus,
  RotateCcw,
  RotateCw,
  SkipBack,
  SkipForward,
  Trash2,
  Volume1,
  Volume2,
  VolumeX,
  X,
} from "lucide-react";

export interface AudioChapter {
  title: string;
  start_sec: number;
  end_sec?: number | null;
}

export interface AudioBookmark {
  id: string;
  time_sec: number;
  chapter_title?: string;
  note?: string;
  created_at: string;
}

interface AudioPlayerProps {
  rawUrl: string;
  initialTime?: number;
  seekToTime?: number | null;
  onTimeUpdate?: (time: number) => void;
  onBookmarksChange?: (bookmarks: AudioBookmark[]) => void;
  onRegisterDeleteBookmark?: (fn: (id: string) => void) => void;
  title?: string;
  author?: string;
  cover_url?: string;
  chapters?: AudioChapter[];
}

const SPEED_STEPS = [0.75, 1, 1.25, 1.5, 1.75, 2];
const SKIP_SECONDS = 15;
const PREFS_KEY = "reader-audio-prefs";

type AudioPrefs = { volume: number; rate: number };

function loadPrefs(): AudioPrefs {
  try {
    const raw = localStorage.getItem(PREFS_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<AudioPrefs>;
      return {
        volume: typeof parsed.volume === "number" ? Math.min(1, Math.max(0, parsed.volume)) : 1,
        rate: SPEED_STEPS.includes(parsed.rate as number) ? (parsed.rate as number) : 1,
      };
    }
  } catch {
    // corrupted prefs fall back to defaults
  }
  return { volume: 1, rate: 1 };
}

function formatTime(time: number): string {
  if (!Number.isFinite(time) || time < 0) return "0:00";
  const total = Math.floor(time);
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  if (h > 0) return `${h}:${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`;
  return `${m}:${s.toString().padStart(2, "0")}`;
}

export function AudioPlayer({
  rawUrl,
  initialTime = 0,
  seekToTime,
  onTimeUpdate,
  onBookmarksChange,
  onRegisterDeleteBookmark,
  title,
  author,
  cover_url,
  chapters,
}: AudioPlayerProps) {
  const { t } = useTranslation();
  const audioRef = useRef<HTMLAudioElement>(null);
  const barRef = useRef<HTMLDivElement>(null);
  const [prefs, setPrefs] = useState<AudioPrefs>(loadPrefs);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isBuffering, setIsBuffering] = useState(false);
  const [loadError, setLoadError] = useState(false);
  const [currentTime, setCurrentTime] = useState(initialTime);
  const [duration, setDuration] = useState(0);
  const [chaptersOpen, setChaptersOpen] = useState(false);
  const [drawerTab, setDrawerTab] = useState<"chapters" | "bookmarks">("chapters");
  const [speedOpen, setSpeedOpen] = useState(false);
  const [sleepOpen, setSleepOpen] = useState(false);
  const speedRef = useRef<HTMLDivElement>(null);
  const sleepRef = useRef<HTMLDivElement>(null);

  // Audio Bookmarks state
  const [bookmarks, setBookmarks] = useState<AudioBookmark[]>(() => {
    try {
      const raw = localStorage.getItem(`audio-bm-${rawUrl}`);
      return raw ? JSON.parse(raw) : [];
    } catch {
      return [];
    }
  });
  const [showAddBookmarkModal, setShowAddBookmarkModal] = useState(false);
  const [bookmarkNote, setBookmarkNote] = useState("");

  const saveBookmarks = (list: AudioBookmark[]) => {
    setBookmarks(list);
    try {
      localStorage.setItem(`audio-bm-${rawUrl}`, JSON.stringify(list));
    } catch {}
  };

  const handleAddBookmark = (noteText?: string) => {
    const newBm: AudioBookmark = {
      id: Date.now().toString(),
      time_sec: currentTime,
      chapter_title: currentChapter?.title,
      note: noteText?.trim() || undefined,
      created_at: new Date().toISOString(),
    };
    const updated = [...bookmarks, newBm].sort((a, b) => a.time_sec - b.time_sec);
    saveBookmarks(updated);
    setShowAddBookmarkModal(false);
    setBookmarkNote("");
    setChaptersOpen(true);
    setDrawerTab("bookmarks");
  };

  const handleDeleteBookmark = (id: string) => {
    const updated = bookmarks.filter((b) => b.id !== id);
    saveBookmarks(updated);
  };

  // Sync bookmarks upward whenever they change or component mounts
  useEffect(() => {
    onBookmarksChange?.(bookmarks);
  }, [bookmarks, onBookmarksChange]);

  // Register delete handler with parent
  useEffect(() => {
    onRegisterDeleteBookmark?.(handleDeleteBookmark);
  }, [bookmarks, onRegisterDeleteBookmark]);

  // Seek audio when parent requests seeking
  useEffect(() => {
    if (typeof seekToTime === "number" && audioRef.current) {
      audioRef.current.currentTime = seekToTime;
      setCurrentTime(seekToTime);
    }
  }, [seekToTime]);

  // Sleep timer: remaining seconds (>0 means active) or -1 for end_of_chapter
  const [sleepRemaining, setSleepRemaining] = useState<number | null>(null);
  const [sleepMode, setSleepMode] = useState<string>("off");
  const sleepChapterRef = useRef<number | null>(null);

  const sortedChapters = useMemo(
    () => [...(chapters ?? [])].sort((a, b) => a.start_sec - b.start_sec),
    [chapters]
  );

  const currentChapterIndex = useMemo(() => {
    for (let i = sortedChapters.length - 1; i >= 0; i--) {
      if (currentTime >= sortedChapters[i].start_sec) return i;
    }
    return -1;
  }, [sortedChapters, currentTime]);

  const currentChapter = currentChapterIndex >= 0 ? sortedChapters[currentChapterIndex] : undefined;

  // Countdown timer effect
  useEffect(() => {
    if (!isPlaying || sleepRemaining === null) return;
    if (sleepRemaining <= 0) {
      // Timer finished: smooth fade out and pause
      const el = audioRef.current;
      if (el && !el.paused) {
        let vol = el.volume;
        const fade = setInterval(() => {
          vol = Math.max(0, vol - 0.2);
          el.volume = vol;
          if (vol <= 0) {
            clearInterval(fade);
            el.pause();
            el.volume = prefs.volume;
            setIsPlaying(false);
            setSleepRemaining(null);
            setSleepMode("off");
          }
        }, 150);
      }
      return;
    }

    const interval = setInterval(() => {
      setSleepRemaining((prev) => (prev !== null && prev > 0 ? prev - 1 : 0));
    }, 1000);

    return () => clearInterval(interval);
  }, [isPlaying, sleepRemaining, prefs.volume]);

  // End of chapter sleep detector
  useEffect(() => {
    if (!isPlaying || sleepMode !== "end_of_chapter") return;
    if (
      sleepChapterRef.current !== null &&
      currentChapterIndex >= 0 &&
      currentChapterIndex !== sleepChapterRef.current
    ) {
      const el = audioRef.current;
      if (el && !el.paused) {
        el.pause();
        setIsPlaying(false);
        setSleepRemaining(null);
        setSleepMode("off");
        sleepChapterRef.current = null;
      }
    }
  }, [isPlaying, sleepMode, currentChapterIndex]);

  const setSleepTimer = (mode: string, seconds?: number) => {
    setSleepMode(mode);
    setSleepOpen(false);
    if (mode === "off") {
      setSleepRemaining(null);
      sleepChapterRef.current = null;
    } else if (mode === "end_of_chapter") {
      setSleepRemaining(null);
      sleepChapterRef.current = currentChapterIndex;
    } else if (seconds) {
      setSleepRemaining(seconds);
      sleepChapterRef.current = null;
    }
  };

  // Close menus on outside click
  useEffect(() => {
    if (!speedOpen && !sleepOpen) return;
    const onClick = (e: MouseEvent) => {
      if (speedRef.current && !speedRef.current.contains(e.target as Node)) setSpeedOpen(false);
      if (sleepRef.current && !sleepRef.current.contains(e.target as Node)) setSleepOpen(false);
    };
    window.addEventListener("mousedown", onClick);
    return () => window.removeEventListener("mousedown", onClick);
  }, [speedOpen, sleepOpen]);

  // ---- controls -----------------------------------------------------------
  const togglePlay = useCallback(() => {
    const el = audioRef.current;
    if (!el) return;
    if (el.paused) el.play().catch(console.error);
    else el.pause();
  }, []);

  const skip = useCallback((seconds: number) => {
    const el = audioRef.current;
    if (!el) return;
    el.currentTime = Math.min(Math.max(0, el.currentTime + seconds), el.duration || Infinity);
  }, []);

  const jumpChapter = useCallback(
    (dir: 1 | -1) => {
      const el = audioRef.current;
      if (!el || sortedChapters.length === 0) return;
      const idx = currentChapterIndex;
      if (dir === -1) {
        // Restart the chapter if we're >3s into it (podcast-app convention).
        if (idx >= 0 && el.currentTime - sortedChapters[idx].start_sec > 3) {
          el.currentTime = sortedChapters[idx].start_sec;
        } else if (idx > 0) {
          el.currentTime = sortedChapters[idx - 1].start_sec;
        }
      } else if (idx >= 0 && idx + 1 < sortedChapters.length) {
        el.currentTime = sortedChapters[idx + 1].start_sec;
      }
    },
    [sortedChapters, currentChapterIndex]
  );

  const setSpeed = (rate: number) => {
    setPrefs((p) => ({ ...p, rate }));
    setSpeedOpen(false);
  };

  const setVolume = (value: number) => {
    setPrefs((p) => ({ ...p, volume: Math.min(1, Math.max(0, value)) }));
  };

  const seekToRatio = useCallback((ratio: number) => {
    const el = audioRef.current;
    if (!el || !Number.isFinite(el.duration) || el.duration <= 0) return;
    const clamped = Math.min(1, Math.max(0, ratio));
    const time = clamped * el.duration;
    el.currentTime = time;
    setCurrentTime(time);
  }, []);

  // Pointer scrubbing on the progress bar.
  const ratioFromEvent = (e: React.PointerEvent) => {
    const bar = barRef.current;
    if (!bar) return 0;
    const rect = bar.getBoundingClientRect();
    return (e.clientX - rect.left) / rect.width;
  };
  const onBarPointerDown = (e: React.PointerEvent) => {
    e.currentTarget.setPointerCapture(e.pointerId);
    seekToRatio(ratioFromEvent(e));
  };
  const onBarPointerMove = (e: React.PointerEvent) => {
    if (e.buttons & 1) seekToRatio(ratioFromEvent(e));
  };

  // Keyboard: space toggles play, arrows seek.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      if (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable) return;
      if (e.code === "Space") {
        e.preventDefault();
        togglePlay();
      } else if (e.key === "ArrowLeft") {
        skip(-SKIP_SECONDS);
      } else if (e.key === "ArrowRight") {
        skip(SKIP_SECONDS);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [togglePlay, skip]);

  // Lock-screen / OS media controls.
  useEffect(() => {
    if (!("mediaSession" in navigator)) return;
    navigator.mediaSession.metadata = new MediaMetadata({
      title: title || t("reader.audiobook"),
      artist: author || "",
      artwork: cover_url ? [{ src: cover_url, sizes: "512x512" }] : [],
    });
    navigator.mediaSession.setActionHandler("play", togglePlay);
    navigator.mediaSession.setActionHandler("pause", togglePlay);
    navigator.mediaSession.setActionHandler("seekbackward", () => skip(-SKIP_SECONDS));
    navigator.mediaSession.setActionHandler("seekforward", () => skip(SKIP_SECONDS));
    navigator.mediaSession.setActionHandler("previoustrack", () => jumpChapter(-1));
    navigator.mediaSession.setActionHandler("nexttrack", () => jumpChapter(1));
  }, [title, author, cover_url, togglePlay, skip, jumpChapter, t]);

  const progress = duration > 0 ? currentTime / duration : 0;
  const remaining = duration > 0 ? duration - currentTime : 0;

  return (
    <div className="flex h-full w-full flex-col overflow-y-auto bg-[var(--reader-ui-surface)] text-[var(--reader-ui-text)]">
      <audio
        ref={audioRef}
        src={rawUrl}
        preload="metadata"
        onPlay={() => setIsPlaying(true)}
        onPause={() => setIsPlaying(false)}
        onWaiting={() => setIsBuffering(true)}
        onPlaying={() => setIsBuffering(false)}
        onCanPlay={() => setIsBuffering(false)}
        onError={() => {
          setLoadError(true);
          setIsPlaying(false);
        }}
        onLoadedMetadata={(e) => setDuration(e.currentTarget.duration)}
        onDurationChange={(e) => setDuration(e.currentTarget.duration)}
        onTimeUpdate={(e) => {
          const time = e.currentTarget.currentTime;
          setCurrentTime(time);
          onTimeUpdate?.(time);
        }}
        onEnded={() => setIsPlaying(false)}
      />

      {loadError ? (
        <div className="flex flex-1 items-center justify-center p-8">
          <div className="rounded-2xl border border-[var(--reader-ui-border)] bg-[var(--reader-ui-surface-strong)] px-8 py-10 text-center">
            <p className="font-semibold">{t("reader.audio_load_failed", "Could not load this audio file")}</p>
            <p className="mt-1 text-sm opacity-60">{t("reader.audio_load_failed_hint", "Check the file format and try again.")}</p>
          </div>
        </div>
      ) : (
        <>
          {/* Now-playing header: cover + meta, centered */}
          <div className="flex flex-1 flex-col items-center justify-center gap-5 px-6 pt-6 sm:pt-10">
            <div
              className={`relative aspect-square w-44 shrink-0 overflow-hidden rounded-xl bg-base-300 shadow-lg transition-shadow sm:w-56 ${isPlaying ? "shadow-2xl" : ""}`}
            >
              {cover_url ? (
                <img src={cover_url} alt={t("reader.cover_art")} className="h-full w-full object-cover" />
              ) : (
                <div className="flex h-full w-full items-center justify-center">
                  <Play className="h-12 w-12 opacity-25" />
                </div>
              )}
              {isBuffering && (
                <div className="absolute inset-0 grid place-items-center bg-black/30">
                  <span className="loading loading-spinner loading-md text-white" />
                </div>
              )}
            </div>
            <div className="max-w-xl text-center">
              <h2 className="line-clamp-2 text-xl font-bold sm:text-2xl">{title || t("reader.audiobook")}</h2>
              <p className="mt-1 truncate text-sm opacity-60">{author || t("common.unknown")}</p>
              {currentChapter && (
                <p className="mt-2 truncate text-sm font-medium text-[var(--reader-ui-accent,#7c9885)]">
                  {currentChapter.title}
                </p>
              )}
            </div>
          </div>

          {/* Chapters & Bookmarks list panel */}
          {chaptersOpen && (
            <div className="mx-auto w-full max-w-4xl overflow-hidden rounded-t-2xl border-t border-x border-[var(--reader-ui-border)] bg-[var(--reader-ui-surface-strong)] shadow-xl animate-in slide-in-from-bottom-2 duration-150">
              {/* Header with Tabs */}
              <div className="flex items-center justify-between border-b border-[var(--reader-ui-border)] px-4 py-2 bg-base-200/40">
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => setDrawerTab("chapters")}
                    className={`btn btn-xs rounded-xl gap-1.5 font-bold ${
                      drawerTab === "chapters"
                        ? "btn-primary"
                        : "btn-ghost text-base-content/60"
                    }`}
                  >
                    <ListMusic size={14} />
                    <span>{t("reader.chapters_list", "Chương")} ({sortedChapters.length})</span>
                  </button>

                  <button
                    type="button"
                    onClick={() => setDrawerTab("bookmarks")}
                    className={`btn btn-xs rounded-xl gap-1.5 font-bold ${
                      drawerTab === "bookmarks"
                        ? "btn-primary"
                        : "btn-ghost text-base-content/60"
                    }`}
                  >
                    <Bookmark size={14} />
                    <span>{t("reader.bookmarks", "Dấu trang")} ({bookmarks.length})</span>
                  </button>
                </div>

                <div className="flex items-center gap-1">
                  {drawerTab === "bookmarks" && (
                    <button
                      type="button"
                      onClick={() => setShowAddBookmarkModal(true)}
                      className="btn btn-xs btn-outline btn-primary rounded-xl gap-1.5 font-semibold inline-flex items-center"
                      title={t("reader.add_bookmark", "Đánh dấu mốc hiện tại")}
                    >
                      <Plus size={13} className="shrink-0" />
                      <span className="inline-flex items-center gap-1 leading-none">
                        <span className="hidden sm:inline">{t("reader.add_bookmark", "Lưu mốc này")}</span>
                        <span className="font-mono text-xs tabular-nums font-normal">{formatTime(currentTime)}</span>
                      </span>
                    </button>
                  )}
                  <button
                    className="btn btn-circle btn-ghost btn-xs ml-1"
                    onClick={() => setChaptersOpen(false)}
                    aria-label={t("common.cancel", "Cancel")}
                  >
                    <X size={14} />
                  </button>
                </div>
              </div>

              {/* Tab 1: Chapters list */}
              {drawerTab === "chapters" && (
                <ul className="max-h-64 overflow-y-auto divide-y divide-base-200/50">
                  {sortedChapters.length === 0 ? (
                    <li className="p-6 text-center text-xs opacity-50">
                      {t("reader.no_chapters", "Không có danh sách chương")}
                    </li>
                  ) : (
                    sortedChapters.map((ch, i) => (
                      <li key={`${ch.start_sec}-${i}`}>
                        <button
                          className={`flex w-full items-baseline justify-between gap-4 px-4 py-2.5 text-left text-sm transition-colors hover:bg-[var(--reader-ui-hover,rgba(127,127,127,0.12))] ${
                            i === currentChapterIndex ? "font-bold text-[var(--reader-ui-accent,#7c9885)] bg-[var(--reader-ui-hover,rgba(127,127,127,0.08))]" : ""
                          }`}
                          onClick={() => {
                            const el = audioRef.current;
                            if (el) el.currentTime = ch.start_sec;
                            setChaptersOpen(false);
                          }}
                        >
                          <span className="min-w-0 truncate">{ch.title}</span>
                          <span className="shrink-0 text-xs tabular-nums opacity-60">{formatTime(ch.start_sec)}</span>
                        </button>
                      </li>
                    ))
                  )}
                </ul>
              )}

              {/* Tab 2: Audio Bookmarks list */}
              {drawerTab === "bookmarks" && (
                <div className="max-h-64 overflow-y-auto">
                  {bookmarks.length === 0 ? (
                    <div className="py-8 px-4 text-center space-y-2">
                      <Bookmark className="w-8 h-8 mx-auto opacity-30 text-primary" />
                      <p className="text-xs opacity-60 font-medium">
                        {t("reader.no_bookmarks", "Chưa có mốc thời gian nào được lưu.")}
                      </p>
                      <button
                        type="button"
                        onClick={() => setShowAddBookmarkModal(true)}
                        className="btn btn-xs btn-primary rounded-xl gap-1"
                      >
                        <Plus size={13} />
                        {t("reader.bookmark_current_pos", "Lưu mốc hiện tại")} ({formatTime(currentTime)})
                      </button>
                    </div>
                  ) : (
                    <ul className="divide-y divide-base-200/50">
                      {bookmarks.map((bm) => (
                        <li
                          key={bm.id}
                          className="flex items-center justify-between gap-3 px-4 py-2.5 transition-colors hover:bg-[var(--reader-ui-hover,rgba(127,127,127,0.08))]"
                        >
                          <button
                            type="button"
                            className="flex flex-1 items-start gap-3 text-left min-w-0"
                            onClick={() => {
                              const el = audioRef.current;
                              if (el) el.currentTime = bm.time_sec;
                              setChaptersOpen(false);
                            }}
                          >
                            <span className="px-2 py-0.5 rounded-lg bg-primary/10 text-primary font-mono text-xs font-bold shrink-0">
                              {formatTime(bm.time_sec)}
                            </span>
                            <div className="min-w-0 flex-1">
                              <p className="text-xs font-semibold text-[var(--reader-ui-text)] truncate">
                                {bm.note || bm.chapter_title || t("reader.bookmark", "Dấu trang")}
                              </p>
                              {bm.note && bm.chapter_title && (
                                <p className="text-[10px] text-base-content/50 truncate mt-0.5">
                                  {bm.chapter_title}
                                </p>
                              )}
                            </div>
                          </button>

                          <button
                            type="button"
                            onClick={() => handleDeleteBookmark(bm.id)}
                            className="btn btn-ghost btn-circle btn-xs text-base-content/40 hover:text-error shrink-0"
                            title={t("common.delete", "Xóa")}
                          >
                            <Trash2 size={13} />
                          </button>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              )}
            </div>
          )}

          {/* Transport dock, pinned to the bottom, full width */}
          <div className="sticky bottom-0 w-full border-t border-[var(--reader-ui-border)] bg-[var(--reader-ui-surface-strong)] px-4 pb-4 pt-3 backdrop-blur sm:px-8">
            <div className="mx-auto w-full max-w-4xl">
              {/* Timeline */}
              <div className="flex select-none items-center justify-between gap-3 text-xs font-medium tabular-nums opacity-80">
                <span>{formatTime(currentTime)}</span>
                {duration > 0 && <span className="opacity-60">−{formatTime(remaining)}</span>}
              </div>
              <div
                ref={barRef}
                role="slider"
                aria-label={t("reader.seek", "Seek")}
                aria-valuemin={0}
                aria-valuemax={Math.round(duration)}
                aria-valuenow={Math.round(currentTime)}
                tabIndex={0}
                className="group relative -my-1 cursor-pointer py-2"
                onPointerDown={onBarPointerDown}
                onPointerMove={onBarPointerMove}
                onKeyDown={(e) => {
                  if (e.key === "ArrowLeft") skip(-5);
                  if (e.key === "ArrowRight") skip(5);
                }}
              >
                <div className="relative h-1.5 rounded-full bg-current/20">
                  <div
                    className="absolute inset-y-0 left-0 rounded-full bg-[var(--reader-ui-accent,#7c9885)]"
                    style={{ width: `${progress * 100}%` }}
                  />
                  {sortedChapters.map((ch, i) =>
                    duration > 0 && ch.start_sec > 0 ? (
                      <div
                        key={`${ch.start_sec}-${i}`}
                        className="absolute top-1/2 h-2.5 w-0.5 -translate-y-1/2 rounded-full bg-current/40"
                        style={{ left: `${(ch.start_sec / duration) * 100}%` }}
                      />
                    ) : null
                  )}
                  <div
                    className="absolute top-1/2 h-3.5 w-3.5 -translate-x-1/2 -translate-y-1/2 rounded-full bg-[var(--reader-ui-accent,#7c9885)] opacity-0 shadow transition-opacity group-hover:opacity-100"
                    style={{ left: `${progress * 100}%` }}
                  />
                </div>
              </div>

              {/* Controls row */}
              <div className="mt-1 flex flex-wrap items-center justify-center gap-2 sm:justify-between">
                <div className="hidden min-w-28 items-center gap-2 sm:flex">
                  <button
                    className="btn btn-circle btn-ghost btn-xs"
                    onClick={() => setVolume(prefs.volume === 0 ? 1 : 0)}
                    aria-label={prefs.volume === 0 ? t("reader.unmute") : t("reader.mute")}
                  >
                    {prefs.volume === 0 ? <VolumeX size={18} /> : prefs.volume < 0.5 ? <Volume1 size={18} /> : <Volume2 size={18} />}
                  </button>
                  <input
                    type="range"
                    min={0}
                    max={1}
                    step={0.05}
                    value={prefs.volume}
                    onChange={(e) => setVolume(parseFloat(e.target.value))}
                    className="range range-xs w-24"
                    aria-label={t("reader.volume", "Volume")}
                  />
                </div>

                <div className="flex items-center gap-1 sm:gap-2">
                  <button
                    className="btn btn-circle btn-ghost btn-sm"
                    onClick={() => jumpChapter(-1)}
                    disabled={sortedChapters.length === 0}
                    aria-label={t("reader.prev_chapter", "Previous Chapter")}
                    title={t("reader.prev_chapter", "Previous Chapter")}
                  >
                    <SkipBack size={18} />
                  </button>
                  <button
                    className="btn btn-circle btn-ghost btn-sm"
                    onClick={() => skip(-SKIP_SECONDS)}
                    aria-label={t("reader.skip_back_15_seconds")}
                    title={t("reader.skip_back_15_seconds")}
                  >
                    <div className="flex flex-col items-center">
                      <RotateCcw size={20} />
                      <span className="text-[8px] font-bold leading-none">15</span>
                    </div>
                  </button>
                  <button
                    className="btn btn-circle btn-primary mx-1 h-12 w-12 min-h-0"
                    onClick={togglePlay}
                    aria-label={t("reader.play_pause")}
                  >
                    {isPlaying ? <Pause size={22} /> : <Play size={22} className="ml-0.5" />}
                  </button>
                  <button
                    className="btn btn-circle btn-ghost btn-sm"
                    onClick={() => skip(SKIP_SECONDS)}
                    aria-label={t("reader.skip_forward_15_seconds")}
                    title={t("reader.skip_forward_15_seconds")}
                  >
                    <div className="flex flex-col items-center">
                      <RotateCw size={20} />
                      <span className="text-[8px] font-bold leading-none">15</span>
                    </div>
                  </button>
                  <button
                    className="btn btn-circle btn-ghost btn-sm"
                    onClick={() => jumpChapter(1)}
                    disabled={sortedChapters.length === 0}
                    aria-label={t("reader.next_chapter", "Next Chapter")}
                    title={t("reader.next_chapter", "Next Chapter")}
                  >
                    <SkipForward size={18} />
                  </button>
                </div>

                <div className="flex min-w-28 items-center justify-end gap-1">
                  {/* Bookmark Button */}
                  <button
                    className={`btn btn-ghost btn-sm min-h-0 rounded-full px-2.5 font-medium ${
                      drawerTab === "bookmarks" && chaptersOpen ? "text-primary bg-primary/10" : ""
                    }`}
                    onClick={() => {
                      if (chaptersOpen && drawerTab === "bookmarks") {
                        setChaptersOpen(false);
                      } else {
                        setDrawerTab("bookmarks");
                        setChaptersOpen(true);
                        setSpeedOpen(false);
                        setSleepOpen(false);
                      }
                    }}
                    title={t("reader.bookmarks", "Dấu trang")}
                  >
                    <Bookmark size={15} />
                    {bookmarks.length > 0 && (
                      <span className="text-[11px] ml-0.5 tabular-nums font-bold">{bookmarks.length}</span>
                    )}
                  </button>

                  {/* Chapters Button */}
                  {sortedChapters.length > 0 && (
                    <button
                      className={`btn btn-ghost btn-sm min-h-0 rounded-full px-2.5 ${
                        drawerTab === "chapters" && chaptersOpen ? "text-primary bg-primary/10" : ""
                      }`}
                      onClick={() => {
                        if (chaptersOpen && drawerTab === "chapters") {
                          setChaptersOpen(false);
                        } else {
                          setDrawerTab("chapters");
                          setChaptersOpen(true);
                          setSpeedOpen(false);
                          setSleepOpen(false);
                        }
                      }}
                      aria-label={t("reader.chapters_list", "Chapters")}
                    >
                      <ListMusic size={16} />
                      <span className="hidden text-xs sm:inline">{t("reader.chapters_list", "Chapters")}</span>
                    </button>
                  )}

                  {/* Sleep timer dropdown */}
                  <div className="relative" ref={sleepRef}>
                    <button
                      className={`btn btn-ghost btn-sm min-h-0 rounded-full px-2.5 font-medium tabular-nums ${
                        sleepMode !== "off" ? "text-primary font-bold bg-primary/10" : ""
                      }`}
                      onClick={() => {
                        setSleepOpen((v) => !v);
                        setSpeedOpen(false);
                        setChaptersOpen(false);
                      }}
                      aria-haspopup="menu"
                      aria-expanded={sleepOpen}
                      title={t("reader.sleep_timer", "Hẹn giờ tắt")}
                    >
                      <Moon size={15} />
                      {sleepMode === "end_of_chapter" ? (
                        <span className="text-[11px] ml-0.5">{t("reader.end_of_chapter", "Hết chương")}</span>
                      ) : sleepRemaining !== null && sleepRemaining > 0 ? (
                        <span className="text-[11px] ml-0.5 font-mono">{formatTime(sleepRemaining)}</span>
                      ) : null}
                    </button>
                    {sleepOpen && (
                      <div
                        role="menu"
                        className="absolute bottom-11 right-0 z-20 min-w-44 overflow-hidden rounded-xl border border-[var(--reader-ui-border)] bg-[var(--reader-ui-surface-strong)] py-1 shadow-xl text-xs"
                      >
                        <div className="px-3 py-1.5 font-bold text-base-content/60 border-b border-base-200">
                          {t("reader.sleep_timer", "Hẹn giờ tắt")}
                        </div>
                        {[
                          { key: "off", label: t("common.off", "Tắt") },
                          { key: "5m", label: "5 phút", sec: 300 },
                          { key: "15m", label: "15 phút", sec: 900 },
                          { key: "30m", label: "30 phút", sec: 1800 },
                          { key: "45m", label: "45 phút", sec: 2700 },
                          { key: "60m", label: "60 phút", sec: 3600 },
                          { key: "end_of_chapter", label: t("reader.sleep_end_chapter", "Khi hết chương này") },
                        ].map((opt) => (
                          <button
                            key={opt.key}
                            role="menuitemradio"
                            aria-checked={sleepMode === opt.key}
                            className={`flex w-full items-center justify-between gap-2 px-3 py-2 text-left transition-colors hover:bg-[var(--reader-ui-hover,rgba(127,127,127,0.12))] ${
                              sleepMode === opt.key ? "font-bold text-[var(--reader-ui-accent,#7c9885)]" : ""
                            }`}
                            onClick={() => setSleepTimer(opt.key, opt.sec)}
                          >
                            <span>{opt.label}</span>
                            {sleepMode === opt.key && <Check size={14} />}
                          </button>
                        ))}
                      </div>
                    )}
                  </div>

                  {/* YouTube-style speed dropdown */}
                  <div className="relative" ref={speedRef}>
                    <button
                      className={`btn btn-ghost btn-sm min-h-0 rounded-full px-3 font-semibold tabular-nums ${speedOpen ? "text-primary" : ""}`}
                      onClick={() => {
                        setSpeedOpen((v) => !v);
                        setChaptersOpen(false);
                        setSleepOpen(false);
                      }}
                      aria-haspopup="menu"
                      aria-expanded={speedOpen}
                      aria-label={t("reader.playback_speed", "Playback speed")}
                    >
                      <Gauge size={15} />
                      {prefs.rate}×
                    </button>
                    {speedOpen && (
                      <div
                        role="menu"
                        className="absolute bottom-11 right-0 z-20 min-w-32 overflow-hidden rounded-xl border border-[var(--reader-ui-border)] bg-[var(--reader-ui-surface-strong)] py-1 shadow-xl"
                      >
                        {SPEED_STEPS.map((step) => (
                          <button
                            key={step}
                            role="menuitemradio"
                            aria-checked={prefs.rate === step}
                            className={`flex w-full items-center justify-between gap-2 px-3 py-1.5 text-left text-sm tabular-nums transition-colors hover:bg-[var(--reader-ui-hover,rgba(127,127,127,0.12))] ${
                              prefs.rate === step ? "font-semibold text-[var(--reader-ui-accent,#7c9885)]" : ""
                            }`}
                            onClick={() => setSpeed(step)}
                          >
                            {step}×
                            {prefs.rate === step && <Check size={14} />}
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Add Bookmark Modal Dialog */}
          {showAddBookmarkModal && (
            <dialog className="modal modal-open z-60 bg-black/60 backdrop-blur-xs animate-in fade-in duration-150">
              <div className="modal-box max-w-sm p-5 rounded-2xl border border-[var(--reader-ui-border,rgba(255,255,255,0.12))] shadow-2xl bg-[var(--reader-ui-surface-strong,#1e202b)] text-[var(--reader-ui-text,#e2e8f0)]">
                <div className="flex items-center justify-between pb-2 mb-3 border-b border-[var(--reader-ui-border,rgba(255,255,255,0.12))]">
                  <div className="flex items-center gap-2 font-bold text-sm text-[var(--reader-ui-text)]">
                    <BookmarkPlus className="w-4 h-4 text-[var(--reader-ui-accent,#38bdf8)]" />
                    <span>{t("reader.add_bookmark", "Lưu mốc dấu trang")}</span>
                  </div>
                  <button
                    onClick={() => setShowAddBookmarkModal(false)}
                    className="btn btn-xs btn-circle bg-[var(--reader-ui-soft,rgba(255,255,255,0.06))] hover:bg-[var(--reader-ui-hover,rgba(255,255,255,0.1))] text-[var(--reader-ui-text)] border border-[var(--reader-ui-border)]"
                  >
                    ✕
                  </button>
                </div>

                <div className="p-3 bg-[var(--reader-ui-soft,rgba(255,255,255,0.06))] border border-[var(--reader-ui-border)] rounded-xl mb-3 flex items-center justify-between">
                  <span className="text-xs opacity-70">{t("reader.timestamp", "Thời điểm")}:</span>
                  <span className="font-mono font-bold text-[var(--reader-ui-accent,#38bdf8)] text-sm">{formatTime(currentTime)}</span>
                </div>

                <div className="space-y-1.5 mb-4">
                  <label className="text-xs font-semibold opacity-80">
                    {t("reader.bookmark_note", "Ghi chú (tùy chọn)")}:
                  </label>
                  <textarea
                    autoFocus
                    rows={3}
                    value={bookmarkNote}
                    onChange={(e) => setBookmarkNote(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
                        e.preventDefault();
                        handleAddBookmark(bookmarkNote);
                      }
                    }}
                    placeholder={t("reader.bookmark_note_ph", "Nhập ghi chú mốc này...")}
                    className="textarea textarea-bordered textarea-sm w-full rounded-xl text-xs resize-none bg-[var(--reader-ui-soft)] border-[var(--reader-ui-border)] text-[var(--reader-ui-text)] focus:border-[var(--reader-ui-accent)]"
                  />
                </div>

                <div className="flex items-center justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setShowAddBookmarkModal(false)}
                    className="btn btn-sm rounded-xl bg-[var(--reader-ui-soft)] hover:bg-[var(--reader-ui-hover)] text-[var(--reader-ui-text)] border border-[var(--reader-ui-border)]"
                  >
                    {t("common.cancel", "Hủy")}
                  </button>
                  <button
                    type="button"
                    onClick={() => handleAddBookmark(bookmarkNote)}
                    className="btn btn-sm rounded-xl font-bold gap-1 bg-[var(--reader-ui-accent,#38bdf8)] text-[var(--reader-ui-accent-text,#08111d)] border-0 hover:opacity-90"
                  >
                    <Check size={14} />
                    {t("common.save", "Lưu mốc")}
                  </button>
                </div>
              </div>
            </dialog>
          )}
        </>
      )}
    </div>
  );
}
