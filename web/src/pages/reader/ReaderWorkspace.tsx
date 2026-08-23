import { ComicReader, ReaderContent, ReaderImageToolbar, ReaderPageControls, ReaderSelectionToolbar, ReaderSidebar, ReaderTopBar } from "@/components/reader";
import { ReaderInBookSearch } from "@/components/reader/ReaderInBookSearch";
import { QuoteCardModal } from "@/components/common";
import { API_BASE, getMediaUrl } from "@/config/api";
import { getReaderThemeClasses } from "@/config/readerTheme";
import { offlineStore } from "@/lib/offlineStore";
import { useOfflineAssets } from "@/hooks/useOfflineAssets";
import { rawFileKey } from "@/hooks/useOfflineBook";
import { featureService, readerService } from "@/services";
import { useAuthStore, useReaderStore, useGuestStore } from "@/stores";
import type { ActiveImageTarget, AudioBookmark, Chapter, Highlight, ImageBookmark } from "@/types";
import React, { useCallback, useEffect, useMemo, useState, useRef } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";
import { useTTS } from "@/hooks/useTTS";
import { AudioPlayer } from "@/components/reader/AudioPlayer";
import { FastAverageColor } from "fast-average-color";
import { useHighlights } from "@/hooks/useHighlights";
import { useAutoScroll } from "@/hooks/useAutoScroll";
import { useReadingStats } from "@/hooks/useReadingStats";
import { useReaderNavigation } from "@/hooks/useReaderNavigation";
import { useReaderPaging } from "@/hooks/useReaderPaging";
import { useReaderSelection } from "@/hooks/useReaderSelection";
import { queryClient } from "@/config/queryClient";
import { applyUserHighlights, clearHighlight, highlightTextRangeFromNode, extractTextFromHtml, scrollToTextOffset, type TtsStartPoint, type SavedSelection } from "@/lib/readerHighlight";
import { generateCfi, resolveCfi } from "@/lib/epubCfi";
import { BookOpen, ChevronRight } from "lucide-react";

import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { useShallow } from "zustand/react/shallow";

import { MIN_DOUBLE_PAGE_WIDTH, READER_CONTENT_MEASURE, READER_PAGE_GAP, getSideTapRatio } from "@/constants";
import { usePublicSettings } from "@/hooks/useSettings";
import { useReadListNextQuery } from "@/hooks/useReadListQueries";
import { useBookSeriesQuery } from "@/hooks/useBooksQuery";
import { useAudiobookChaptersQuery } from "@/hooks/useAudiobookQueries";
import { useReaderETAQuery } from "@/hooks/useReadingStats";
import { hasPermission } from "@/utils/permission";
import { isNavOnlyChapter, isVisualChapter } from "@/utils/readerHtml";

const ambientColorCache = new Map<string, { hex: string; rgba: string }>();

const ReaderWorkspaceInner = () => {
  const { book_id } = useParams<{ book_id: string }>();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const file_id = searchParams.get("file_id") || undefined;
  const readListId = searchParams.get("readlist") || undefined;
  
  const { t } = useTranslation();
  const [searchOpen, setSearchOpen] = useState(false);
  const [quoteModalOpen, setQuoteModalOpen] = useState(false);
  const [quoteModalText, setQuoteModalText] = useState("");
  const [quoteModalImageUrl, setQuoteModalImageUrl] = useState<string | undefined>(undefined);

  const ttsStartPointRef = useRef<TtsStartPoint | null>(null);
  const savedSelectionRef = useRef<SavedSelection | null>(null);

  // Declared up here because the TTS boundary callback and the extracted reader
  // hooks below all close over them.
  const contentRef = useRef<HTMLDivElement>(null);
  const columnsRef = useRef<HTMLDivElement>(null);
  const pageFrameRef = useRef<HTMLDivElement>(null);
  const sidebarRef = useRef<HTMLElement>(null);
  const pendingFragmentRef = useRef<string | null>(null);
  const pendingLandingRef = useRef<string | null>(null);
  const pendingTextOffsetRef = useRef<number | null>(null);
  const lastFocusedControlRef = useRef<HTMLElement | null>(null);
  const ttsOffsetRef = useRef<number>(0);
  const { resolveHTML, resolveBlobURL } = useOfflineAssets(book_id);
  const [offlineRawUrl, setOfflineRawUrl] = useState<string | undefined>(undefined);

  const { user } = useAuthStore(useShallow((state) => ({ user: state.user })));

  const {
    book,
    chapters,
    currentChapter,
    htmlContent,
    loading,
    sidebarOpen,
    settingsOpen,
    fontSize,
    fontFamily,
    theme,
    lineHeight,
    maxWidth,
    textAlign,
    readingMode,
    readingDirection,
    pageFit,
    pageAnimation,
    pageIndex,
    pageFrameWidth,
    setBook,
    setChapters,
    setCurrentChapter,
    setHtmlContent,
    setLoading,
    setSidebarOpen,
    setSettingsOpen,
    setFontSize,
    setFontFamily,
    setTheme,
    setLineHeight,
    setMaxWidth,
    setTextAlign,
    setReadingMode,
    setReadingDirection,
    setPageFit,
    setPageAnimation,
    setPageIndex,
    setPageFrameWidth,
    ttsVoiceName,
    ttsRate,
    setTtsVoiceName,
    setTtsRate,
    resetSettings,
    reset,
  } = useReaderStore(useShallow((state) => ({
    book: state.book,
    chapters: state.chapters,
    currentChapter: state.currentChapter,
    htmlContent: state.htmlContent,
    loading: state.loading,
    sidebarOpen: state.sidebarOpen,
    settingsOpen: state.settingsOpen,
    fontSize: state.fontSize,
    fontFamily: state.fontFamily,
    theme: state.theme,
    lineHeight: state.lineHeight,
    maxWidth: state.maxWidth,
    textAlign: state.textAlign,
    readingMode: state.readingMode,
    readingDirection: state.readingDirection,
    pageFit: state.pageFit,
    pageAnimation: state.pageAnimation,
    pageIndex: state.pageIndex,
    pageFrameWidth: state.pageFrameWidth,
    ttsVoiceName: state.ttsVoiceName,
    ttsRate: state.ttsRate,
    setBook: state.setBook,
    setChapters: state.setChapters,
    setCurrentChapter: state.setCurrentChapter,
    setHtmlContent: state.setHtmlContent,
    setLoading: state.setLoading,
    setSidebarOpen: state.setSidebarOpen,
    setSettingsOpen: state.setSettingsOpen,
    setFontSize: state.setFontSize,
    setFontFamily: state.setFontFamily,
    setTheme: state.setTheme,
    setLineHeight: state.setLineHeight,
    setMaxWidth: state.setMaxWidth,
    setTextAlign: state.setTextAlign,
    setReadingMode: state.setReadingMode,
    setReadingDirection: state.setReadingDirection,
    setPageFit: state.setPageFit,
    setPageAnimation: state.setPageAnimation,
    setPageIndex: state.setPageIndex,
    setPageFrameWidth: state.setPageFrameWidth,
    setTtsVoiceName: state.setTtsVoiceName,
    setTtsRate: state.setTtsRate,
    resetSettings: state.resetSettings,
    reset: state.reset,
  })));

  const {
    readerBg,
    proseClass,
    sidebarBg,
    headerBg,
    linkColor,
    linkColorHover,
  } = getReaderThemeClasses(theme);

  useReadingStats(book?.id, !settingsOpen);

  const publicSettings = usePublicSettings();
  const guestPerms = publicSettings?.guest_permissions;
  const allowTTS = hasPermission(user, "book.tts", book?.library_id, guestPerms);
  const allowHighlights = hasPermission(user, "book.highlight", book?.library_id, guestPerms);
  const currentChapterExists = Boolean(currentChapter && chapters.some((c) => c.id === currentChapter.id));
  const { data: readerETA } = useReaderETAQuery(user && hasPermission(user, "user.stats.read") ? book_id : undefined);
  const { highlights, addHighlight, updateHighlight, removeHighlight } = useHighlights(book?.id || '', currentChapter?.id, allowHighlights && currentChapterExists);

  useEffect(() => {
    if (book?.title) {
      const siteTitle = publicSettings?.site?.title || "NovelHub";
      document.title = `${t('reader.reading', 'Reading')}: ${book.title} | ${siteTitle}`;
    }
  }, [book?.title, publicSettings?.site?.title, t]);

  useEffect(() => {
    return () => {
      reset();
    };
  }, [reset]);

  const { isSupported, isPlaying, isPaused, speak, pause, resume, stop, voices, selectedVoice, setSelectedVoice, rate, setRate } = useTTS({
    onEnd: () => clearHighlight(),
    onBoundary: (e) => {
      if (columnsRef.current && (e.name === 'word' || !e.name)) {
        const textToSearch = e.utterance?.text || "";
        const wordLen = e.charLength || (textToSearch ? (textToSearch.slice(e.charIndex).match(/^\S+/)?.[0]?.length || 1) : 1);
        highlightTextRangeFromNode(columnsRef.current, ttsStartPointRef.current, e.charIndex, wordLen);
      }
    }
  });

  useEffect(() => {
    if (ttsRate && ttsRate !== rate) {
      setRate(ttsRate);
    }
  }, [ttsRate]);

  useEffect(() => {
    if (voices.length > 0 && ttsVoiceName) {
      const found = voices.find(v => v.name === ttsVoiceName);
      if (found && selectedVoice?.name !== found.name) {
        setSelectedVoice(found);
      }
    }
  }, [voices, ttsVoiceName]);

  const handleTtsVoiceChange = (voice: SpeechSynthesisVoice | null) => {
    setSelectedVoice(voice);
    setTtsVoiceName(voice?.name || null);
  };

  const handleTtsRateChange = (newRate: number) => {
    setRate(newRate);
    setTtsRate(newRate);
  };

  const handleTtsPlayPause = () => {
    if (isPlaying) {
      pause();
    } else if (isPaused) {
      resume();
    } else if (htmlContent) {
      const text = extractTextFromHtml(htmlContent);
      if (text.trim()) {
        ttsOffsetRef.current = 0;
        speak(text);
      }
    }
  };

  useEffect(() => {
    if (!isPlaying && !isPaused) {
      clearHighlight();
    }
  }, [isPlaying, isPaused]);

  const currentChapterIndex = chapters.findIndex(
    (chapter) => chapter.id === currentChapter?.id,
  );

  const [audioBookmarks, setAudioBookmarks] = useState<AudioBookmark[]>([]);
  const [audioSeekTime, setAudioSeekTime] = useState<number | null>(null);
  const audioDeleteBookmarkRef = useRef<((id: string) => void) | null>(null);
  const lastAudioProgressRef = useRef<number>(0);

  // Image Bookmarks State
  const [imageBookmarks, setImageBookmarks] = useState<ImageBookmark[]>(() => {
    try {
      const raw = localStorage.getItem(`img-bm-${book_id}`);
      return raw ? JSON.parse(raw) : [];
    } catch {
      return [];
    }
  });
  const [activeImageTarget, setActiveImageTarget] = useState<ActiveImageTarget | null>(null);

  const saveImageBookmarks = (list: ImageBookmark[]) => {
    setImageBookmarks(list);
    try {
      localStorage.setItem(`img-bm-${book_id}`, JSON.stringify(list));
    } catch {}
  };

  const handleSaveImageBookmark = (bm: Omit<ImageBookmark, "id" | "created_at">) => {
    const newBm: ImageBookmark = {
      ...bm,
      id: Date.now().toString(),
      created_at: new Date().toISOString(),
    };
    const updated = [newBm, ...imageBookmarks];
    saveImageBookmarks(updated);
    toast.success(t("reader.image_bookmarked", "Image bookmarked!"));
  };

  const handleDeleteImageBookmark = (id: string) => {
    const updated = imageBookmarks.filter((b) => b.id !== id);
    saveImageBookmarks(updated);
  };

  const handleSelectImageBookmark = (bm: ImageBookmark) => {
    if (bm.chapter_id && bm.chapter_id !== currentChapter?.id) {
      const ch = chapters.find((c) => c.id === bm.chapter_id);
      if (ch) {
        void loadChapter(ch);
      }
    }
    if (isVisualContent && bm.page_index !== undefined) {
      setComicPage(bm.page_index);
    } else {
      setTimeout(() => {
        const container = columnsRef.current || contentRef.current;
        if (container) {
          const imgs = Array.from(container.querySelectorAll("img"));
          const matched = imgs.find((img) => img.src === bm.image_url || img.getAttribute("src") === bm.image_url);
          if (matched) {
            const isPaged = container.classList?.contains("reader-mode-single") || container.classList?.contains("reader-mode-double");
            const pagedContainer = (container.querySelector("body") || container) as HTMLElement;
            if (isPaged && pagedContainer.clientWidth > 0) {
              const rangeRect = matched.getBoundingClientRect();
              const containerRect = pagedContainer.getBoundingClientRect();
              const pageGap = 40;
              const scrollStep = pagedContainer.clientWidth + pageGap;
              const currentScrollLeft = pagedContainer.scrollLeft;
              const relativeLeft = (rangeRect.left - containerRect.left) + currentScrollLeft;
              const pageIndex = Math.max(0, Math.floor(relativeLeft / scrollStep));
              pagedContainer.scrollTo({
                left: pageIndex * scrollStep,
                behavior: "smooth",
              });
            } else {
              matched.scrollIntoView({ block: "center", behavior: "smooth" });
            }
          }
        }
      }, 250);
    }
    setSidebarOpen(false);
    restoreFocus();
  };

  const handleContentClickCapture = (e: React.MouseEvent) => {
    if (isComicFormat) return;

    const target = e.target as HTMLElement;
    if (target && target.tagName.toLowerCase() === "img") {
      const img = target as HTMLImageElement;
      const src = img.currentSrc || img.src;
      if (src) {
        // If clicking in the side tap zone on mobile/tablet, let page turning happen instead of selecting image
        const screenWidth = typeof window !== "undefined" ? window.innerWidth : 1024;
        const sideRatio = getSideTapRatio(screenWidth);
        if (sideRatio > 0) {
          const clickRatio = e.clientX / screenWidth;
          if (clickRatio < sideRatio || clickRatio > 1 - sideRatio) {
            return;
          }
        }
        setActiveImageTarget({
          image_url: src,
          chapter_id: currentChapter?.id,
          chapter_title: currentChapter?.title,
          x: e.clientX,
          y: e.clientY,
        });
        return;
      }
    }
    if (activeImageTarget) {
      setActiveImageTarget(null);
    }
  };

  const {
    selectionRange,
    setSelectionRange,
    toolbarPos,
    handleHighlight,
    handleReadSelection,
    handleReadFromHere,
    handleCopyText,
  } = useReaderSelection({
    columnsRef,
    contentRef,
    savedSelectionRef,
    ttsStartPointRef,
    addHighlight,
    speak,
    stop,
    chapterIndex: currentChapterIndex >= 0 ? currentChapterIndex : 0,
    chapterId: currentChapter?.id,
  });

  const handleSelectHighlight = (highlight: Highlight) => {
    if (highlight.chapter_id && highlight.chapter_id !== currentChapter?.id) {
      const ch = chapters.find((c) => c.id === highlight.chapter_id);
      if (ch) {
        pendingTextOffsetRef.current = highlight.start_index;
        void loadChapter(ch);
        setSidebarOpen(false);
        restoreFocus();
        return;
      }
    }
    const container = columnsRef.current || contentRef.current;
    if (container && highlight.start_index >= 0) {
      scrollToTextOffset(container, highlight.start_index);
      setSidebarOpen(false);
      restoreFocus();
    }
  };

  const [ambientColor, setAmbientColor] = useState<string>("transparent");

  useEffect(() => {
    if (!book || !book.cover_url) {
      setAmbientColor("transparent");
      return;
    }
    const coverUrl = book.cover_url;
    if (ambientColorCache.has(coverUrl)) {
      const cached = ambientColorCache.get(coverUrl)!;
      setAmbientColor(theme === 'dark' ? cached.hex : cached.rgba);
      return;
    }

    const fac = new FastAverageColor();
    const img = new Image();
    img.crossOrigin = "Anonymous";
    img.src = coverUrl;
    img.onload = () => {
      try {
        const color = fac.getColor(img);
        ambientColorCache.set(coverUrl, { hex: color.hex, rgba: color.rgba });
        setAmbientColor(theme === 'dark' ? color.hex : color.rgba);
      } catch (e) {
        setAmbientColor("transparent");
      }
    };
    return () => { fac.destroy(); };
  }, [book?.cover_url, theme]);

  const { isScrolling: autoScrollActive, toggleScroll: onToggleAutoScroll, updateSpeed } = useAutoScroll(contentRef);
  const [comicPage, setComicPage] = useState(0);
  const [comicTotalPages, setComicTotalPages] = useState(0);

  const isMobileScreen = typeof window !== "undefined" && (window.innerWidth < 768 || window.screen.width < 768);
  const doublePageWidth = pageFrameWidth > 0 ? Math.floor((pageFrameWidth - READER_PAGE_GAP) / 2) : 0;
  const canUseDoubleMode = !isMobileScreen && (pageFrameWidth === 0 ? (typeof window !== "undefined" && window.innerWidth >= 768) : doublePageWidth >= MIN_DOUBLE_PAGE_WIDTH);
  const effectiveReadingMode = readingMode === "double" && !canUseDoubleMode ? "single" : readingMode;

  const scrollLayout = effectiveReadingMode === "scroll" || effectiveReadingMode === "webtoon";
  const activeFile = file_id ? book?.files?.find(f => f.id === file_id) : book?.files?.[0];
  const isComicFormat = !!activeFile?.format.match(/^(cbz|cbr|cbt|cb7)$/i);
  const isVisualContent = isComicFormat || isVisualChapter(htmlContent);
  const rtlPaging = isVisualContent && readingDirection === "rtl";
  const isPdf = !!(activeFile?.format.match(/^pdf$/i) || currentChapter?.content_path?.toLowerCase().endsWith(".pdf"));
  const isAudio = !!activeFile?.format.match(/^(mp3|m4a|m4b|flac)$/i);
  const isPdfAudio = isPdf || isAudio;
  const rawFileUrl = `${API_BASE}/reader/${encodeURIComponent(book_id || "")}/file?file_id=${encodeURIComponent(activeFile?.id || file_id || "")}`;

  const comicImageTarget = useMemo(() => {
    if (!isComicFormat || !htmlContent) return null;
    const div = document.createElement("div");
    div.innerHTML = htmlContent;
    const imgs = Array.from(div.querySelectorAll("img"));
    const urls = imgs.map((img) => img.getAttribute("src") || "").filter(Boolean);
    const url = urls[comicPage] || urls[0];
    if (!url) return null;
    return {
      image_url: url,
      chapter_id: currentChapter?.id,
      chapter_title: currentChapter?.title,
      page_index: comicPage,
    };
  }, [isComicFormat, htmlContent, comicPage, currentChapter]);

  const effectiveActiveImageTarget = activeImageTarget || comicImageTarget;

  useEffect(() => {
    setActiveImageTarget(null);
  }, [comicPage]);

  const comicStep = effectiveReadingMode === "double" ? 2 : 1;

  const handleComicPagePrev = () => {
    if (comicPage > 0) {
      setComicPage(Math.max(0, comicPage - comicStep));
    } else if (canGoPrev) {
      void handlePrev(true);
    }
  };

  const handleComicPageNext = () => {
    if (comicPage + comicStep < comicTotalPages) {
      setComicPage(comicPage + comicStep);
    } else if (canGoNext) {
      void handleNext();
    }
  };

  // A HEAD probe rather than waiting for the player to fail: an <audio> or <iframe> pointed at
  // an unreachable URL shows its own broken state and never tells us to fall back.
  useEffect(() => {
    if (!isPdfAudio || !activeFile) {
      setOfflineRawUrl(undefined);
      return;
    }
    let active = true;
    void fetch(rawFileUrl, { method: "HEAD", credentials: "include" })
      .then((res) => (res.ok ? undefined : Promise.reject(new Error("unreachable"))))
      .catch(() => resolveBlobURL(rawFileKey(activeFile.id)))
      .then((url) => {
        if (active) setOfflineRawUrl(url || undefined);
      })
      .catch(() => active && setOfflineRawUrl(undefined));
    return () => {
      active = false;
    };
  }, [isPdfAudio, activeFile, rawFileUrl, resolveBlobURL]);

  const visiblePages = effectiveReadingMode === "double" ? 2 : 1;
  const rawFrameWidth = pageFrameWidth > 0 ? pageFrameWidth : (typeof window !== "undefined" ? Math.min(window.innerWidth - 64, maxWidth) : 920);
  const pageWidth = scrollLayout
    ? 0
    : Math.max(1, Math.floor((rawFrameWidth - READER_PAGE_GAP * (visiblePages - 1)) / visiblePages));

  useEffect(() => {
    const container = columnsRef.current || contentRef.current;
    if (!container || !highlights) return;

    let rafId: number;
    let timerId1: ReturnType<typeof setTimeout>;
    let timerId2: ReturnType<typeof setTimeout>;
    let timerId3: ReturnType<typeof setTimeout>;
    const reapply = () => {
      cancelAnimationFrame(rafId);
      rafId = requestAnimationFrame(() => {
        const target = columnsRef.current || contentRef.current;
        if (target) {
          const currentChapterHighlights = highlights.filter(
            (h) => !h.chapter_id || h.chapter_id === currentChapter?.id
          );
          applyUserHighlights(target, currentChapterHighlights);
        }
      });
    };

    reapply();
    timerId1 = setTimeout(reapply, 60);
    timerId2 = setTimeout(reapply, 200);
    timerId3 = setTimeout(reapply, 600);

    const observer = new ResizeObserver(() => {
      reapply();
    });

    observer.observe(container);

    return () => {
      cancelAnimationFrame(rafId);
      clearTimeout(timerId1);
      clearTimeout(timerId2);
      clearTimeout(timerId3);
      observer.disconnect();
    };
  }, [
    highlights,
    htmlContent,
    currentChapter?.id,
    theme,
    effectiveReadingMode,
    pageWidth,
    fontSize,
    lineHeight,
    fontFamily,
    proseClass,
    readerBg,
    readingDirection,
    pageFit,
    maxWidth,
    sidebarOpen,
    settingsOpen,
  ]);

  useEffect(() => {
    const container = columnsRef.current || contentRef.current;
    if (container && highlights && highlights.length > 0) {
      applyUserHighlights(container, highlights);
    }
  }, [selectionRange, isPlaying, isPaused, highlights, scrollLayout, autoScrollActive]);
  const {
    getPagedScrollMetrics,
    scrollToPageIndex,
    getLocationFraction,
    handlePageNext,
    handlePagePrev,
  } = useReaderPaging({
    contentRef,
    columnsRef,
    pageFrameRef,
    pendingLandingRef,
    htmlContent,
    maxWidth,
    scrollLayout,
    effectiveReadingMode,
    rtlPaging,
    pageAnimation,
    pageIndex,
    setPageIndex,
    setPageFrameWidth,
    onChapterNext: () => handleNext(),
    onChapterPrev: () => handlePrev(true),
  });

  useEffect(() => {
    if (!sidebarOpen) return;

    const handlePointerDownOutside = (event: PointerEvent) => {
      const target = event.target;
      if (!(target instanceof Node)) return;
      if (sidebarRef.current?.contains(target)) return;
      if (target instanceof Element && target.closest('.modal, [role="dialog"], [data-reader-modal="true"]')) return;
      setSidebarOpen(false);
    };

    document.addEventListener("pointerdown", handlePointerDownOutside, true);
    return () => {
      document.removeEventListener("pointerdown", handlePointerDownOutside, true);
    };
  }, [sidebarOpen, setSidebarOpen]);

  const computeProgressPercent = (): number => {
    const chapterPosition = chapters.findIndex((c) => c.id === currentChapter?.id);
    if (chapterPosition < 0 || chapters.length === 0) return 0;
    const fraction = getLocationFraction();
    return Math.min(100, Math.round(((chapterPosition + fraction) / chapters.length) * 100));
  };

  const { handleContentClick, scrollToFragment } = useReaderNavigation({
    columnsRef,
    pendingFragmentRef,
    chapters,
    scrollLayout,
    loadChapter: (chapter) => loadChapter(chapter),
    getPagedScrollMetrics,
    scrollToPageIndex,
    onPagePrev: handlePagePrev,
    onPageNext: handlePageNext,
    rtlPaging,
    onCenterClick: () => {
      setSettingsOpen(false);
      setSidebarOpen(false);
    },
    onImageClick: (target) => {
      setActiveImageTarget({
        image_url: target.image_url,
        chapter_id: currentChapter?.id,
        chapter_title: currentChapter?.title,
        x: target.x,
        y: target.y,
      });
    },
  });

  useEffect(() => {
    if (!book_id) return;

    reset();
    setLoading(true);
    
    const bootstrap = async () => {
      try {
        const [res, progressRes] = await Promise.allSettled([
          readerService.getBootstrap(book_id, file_id),
          user ? featureService.getReadingProgress(book_id) : Promise.reject("guest"),
        ]);

        const offline = res.status === "fulfilled" && res.value.status && res.value.data
          ? null
          : await offlineStore.getBook(book_id).catch(() => undefined);
        const loaded = offline
          ? { book: offline.book, chapters: offline.chapters }
          : res.status === "fulfilled" && res.value.status ? res.value.data : null;

        if (loaded) {
          setBook(loaded.book);
          const sorted = [...loaded.chapters].sort((a, b) => a.chapter_index - b.chapter_index);
          setChapters(sorted);
          if (sorted.length > 0) {
            let targetChapter = sorted.find((ch) => !isNavOnlyChapter(ch.title)) || sorted[0];
            let location_cfi: string | undefined = undefined;
            const startOver = searchParams.get("start_over") === "true";

            if (!startOver) {
              if (progressRes.status === "fulfilled" && progressRes.value.status && progressRes.value.data) {
                const progress = progressRes.value.data;
                const found = sorted.find(ch => ch.id === progress.chapter_id);
                if (found && !isNavOnlyChapter(found.title)) {
                  targetChapter = found;
                  if (progress.location_cfi && progress.location_cfi.startsWith("epubcfi(")) {
                    location_cfi = progress.location_cfi;
                  } else if (progress.location_type === "scroll" && progress.location_cfi) {
                    location_cfi = `scroll:${progress.location_cfi}`;
                  } else if (progress.location_type === "page" && progress.location_cfi) {
                    location_cfi = `page:${progress.location_cfi}`;
                  } else if (progress.location_type === "audio" && progress.location_cfi) {
                    location_cfi = `audio:${progress.location_cfi}`;
                  } else if (progress.location_cfi) {
                    location_cfi = progress.location_cfi;
                  }
                }
              } else if (!user) {
                const guestHist = useGuestStore.getState().getReadingHistory().find(h => h.book_id === book_id);
                if (guestHist) {
                  const found = sorted.find(ch => ch.id === guestHist.chapter_id);
                  if (found && !isNavOnlyChapter(found.title)) {
                    targetChapter = found;
                  }
                }
              }
            }

            if (location_cfi) {
              pendingFragmentRef.current = location_cfi;
            }
            loadChapter(targetChapter);
          }
        }
      } catch (err) {
        console.error("Failed to load book", err);
      } finally {
        setLoading(false);
      }
    };
    bootstrap();
  }, [book_id, file_id]);

  const loadChapter = async (chapter: Chapter) => {
    if (!book_id) return;
    setCurrentChapter(chapter);
    setHtmlContent("");
    stop();
    try {
      const html = await readerService.getChapterHtml(book_id, chapter.id, file_id);
      setHtmlContent(await resolveHTML(html));
      if (contentRef.current) {
        contentRef.current.scrollTop = 0;
      }
      const idx = chapters.findIndex((c) => c.id === chapter.id);
      if (idx >= 0 && idx < chapters.length - 1) {
        readerService.prefetchChapter(book_id, chapters[idx + 1].id, file_id);
      }
    } catch (err) {
      console.error("Failed to load chapter content", err);
      const stored = await offlineStore.getChapter(book_id, chapter.id).catch(() => undefined);
      if (stored) {
        setHtmlContent(await resolveHTML(stored));
        if (contentRef.current) {
          contentRef.current.scrollTop = 0;
        }
        return;
      }
      setHtmlContent(`<div class='text-error p-4'>${t('offline.chapter_unavailable', 'This chapter is not available offline. Save the book for offline reading while you are connected.')}</div>`);
    }
  };

  useEffect(() => {
    if (!htmlContent || !pendingFragmentRef.current) return;
    const fragment = pendingFragmentRef.current;
    pendingFragmentRef.current = null;
    requestAnimationFrame(() => {
      if (fragment.startsWith("epubcfi(")) {
        const container = columnsRef.current || contentRef.current;
        if (container) {
          const resolved = resolveCfi(container, fragment);
          if (resolved) {
            const el = resolved.node.nodeType === Node.ELEMENT_NODE
              ? (resolved.node as HTMLElement)
              : resolved.node.parentElement;
            if (el) {
              if (scrollLayout) {
                el.scrollIntoView({ block: "center", behavior: "auto" });
              } else {
                const scrollStep = container.clientWidth + READER_PAGE_GAP;
                if (scrollStep > 0) {
                  let offsetLeft = el.offsetLeft;
                  let parent = el.offsetParent as HTMLElement;
                  while (parent && container.contains(parent) && parent !== container) {
                    offsetLeft += parent.offsetLeft;
                    parent = parent.offsetParent as HTMLElement;
                  }
                  const pIndex = Math.floor(offsetLeft / scrollStep);
                  scrollToPageIndex(pIndex, true);
                }
              }
            }
          }
        }
      } else if (fragment.startsWith("scroll:") && contentRef.current) {
        contentRef.current.scrollTop = parseInt(fragment.slice(7), 10) || 0;
      } else if (fragment.startsWith("page:")) {
        const pIndex = parseInt(fragment.slice(5), 10) || 0;
        scrollToPageIndex(pIndex);
      } else {
        scrollToFragment(fragment);
      }
    });
  }, [htmlContent]);

  // Resolve in-book search offset to a DOM range and scroll it into view.
  useEffect(() => {
    if (!htmlContent || pendingTextOffsetRef.current == null) return;
    const offset = pendingTextOffsetRef.current;
    pendingTextOffsetRef.current = null;
    requestAnimationFrame(() => {
      const container = columnsRef.current || contentRef.current;
      if (container && scrollToTextOffset(container, offset)) return;
      // ponytail: backend offset is currently always 0 (FTS snippet() only),
      // so deep-link can't resolve — fall back to chapter top. BE contract gap.
      if (contentRef.current) contentRef.current.scrollTop = 0;
    });
  }, [htmlContent]);

  const invalidateProgressQueries = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["reading"] });
    void queryClient.invalidateQueries({ queryKey: ["trackerReadingProgress"] });
    void queryClient.invalidateQueries({ queryKey: ["bookUserState", book_id] });
  }, [queryClient, book_id]);

  const progressWarnedRef = useRef(false);
  const reportProgressFailure = useCallback((error: unknown) => {
    console.debug("Failed to record reading activity", error);
    if (progressWarnedRef.current) return;
    progressWarnedRef.current = true;
    toast.warning(t("reader.progress_sync_failed", "Reading progress is not being saved right now"));
  }, [t]);

  useEffect(() => {
    return () => {
      void queryClient.invalidateQueries({ queryKey: ["reading"] });
      void queryClient.invalidateQueries({ queryKey: ["books"] });
      void queryClient.invalidateQueries({ queryKey: ["library"] });
      void queryClient.invalidateQueries({ queryKey: ["trackerReadingProgress"] });
      void queryClient.invalidateQueries({ queryKey: ["bookUserState"] });
    };
  }, []);

  useEffect(() => {
    if (!currentChapter || !book_id) return;
    const progress_percent = computeProgressPercent();

    if (!user) {
      useGuestStore.getState().recordReading({
        book_id,
        file_id,
        chapter_id: currentChapter.id,
        chapter_title: currentChapter.title,
        chapter_index: currentChapter.chapter_index,
        progress_percent,
        book_title: book?.title,
        author_name: (book as any)?.author_name,
        cover_url: book?.cover_url,
        updated_at: new Date().toISOString(),
      });
      return;
    }

    void featureService.recordReadingActivity({
      book_id,
      file_id,
      chapter_id: currentChapter.id,
      chapter_title: currentChapter.title,
      chapter_index: currentChapter.chapter_index,
      progress_percent,
      event_type: "chapter_open",
    }).then(() => {
      invalidateProgressQueries();
    }).catch(reportProgressFailure);
  }, [user, book_id, file_id, currentChapter?.id, chapters.length, book]);

  const getVisibleCfi = (): string => {
    const container = scrollLayout ? contentRef.current : columnsRef.current;
    if (!container || !currentChapter) return "";

    const children = Array.from(container.querySelectorAll("p, h1, h2, h3, h4, h5, h6, li, figure, img, div"));
    const containerRect = container.getBoundingClientRect();

    let visibleNode: Node | null = null;
    for (const child of children) {
      if (child.tagName === "DIV" && child.querySelector("p, h1, h2, h3, h4, h5, h6, li, figure, img")) {
        continue;
      }
      const rect = child.getBoundingClientRect();
      if (!scrollLayout) {
        if (rect.right > containerRect.left && rect.left < containerRect.right) {
          visibleNode = child;
          break;
        }
      } else {
        if (rect.bottom > containerRect.top && rect.top < containerRect.bottom) {
          visibleNode = child;
          break;
        }
      }
    }

    if (!visibleNode) {
      const treeWalker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT, null);
      visibleNode = treeWalker.nextNode() || container;
    }

    let targetNode = visibleNode;
    if (visibleNode.nodeType === Node.ELEMENT_NODE) {
      const treeWalker = document.createTreeWalker(visibleNode, NodeFilter.SHOW_TEXT, null);
      const firstText = treeWalker.nextNode();
      if (firstText) {
        targetNode = firstText;
      }
    }

    const currentChapterIndex = chapters.findIndex((c) => c.id === currentChapter.id);
    return generateCfi(container, targetNode, 0, currentChapterIndex >= 0 ? currentChapterIndex : 0, currentChapter.id);
  };

  const scrollTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleScroll = () => {
    if (!scrollLayout || !contentRef.current || !currentChapter || !book_id) return;

    if (scrollTimeoutRef.current) {
      clearTimeout(scrollTimeoutRef.current);
    }

    scrollTimeoutRef.current = setTimeout(() => {
      const progress_percent = computeProgressPercent();

      if (!user) {
        useGuestStore.getState().recordReading({
          book_id,
          file_id,
          chapter_id: currentChapter.id,
          chapter_title: currentChapter.title,
          chapter_index: currentChapter.chapter_index,
          progress_percent,
          book_title: book?.title,
          author_name: (book as any)?.author_name,
          cover_url: book?.cover_url,
          updated_at: new Date().toISOString(),
        });
        return;
      }

      void featureService.recordReadingActivity({
        book_id,
        file_id,
        chapter_id: currentChapter.id,
        chapter_title: currentChapter.title,
        chapter_index: currentChapter.chapter_index,
        progress_percent,
        location_cfi: getVisibleCfi(),
        location_type: "scroll",
        event_type: "progress_update",
      }).then(() => {
        invalidateProgressQueries();
      }).catch(reportProgressFailure);
    }, 2000);
  };

  useEffect(() => {
    if (scrollLayout || !currentChapter || !book_id) return;

    const progress_percent = computeProgressPercent();

    if (!user) {
      useGuestStore.getState().recordReading({
        book_id,
        file_id,
        chapter_id: currentChapter.id,
        chapter_title: currentChapter.title,
        chapter_index: currentChapter.chapter_index,
        progress_percent,
        book_title: book?.title,
        author_name: (book as any)?.author_name,
        cover_url: book?.cover_url,
        updated_at: new Date().toISOString(),
      });
      return;
    }

    void featureService.recordReadingActivity({
      book_id,
      file_id,
      chapter_id: currentChapter.id,
      chapter_title: currentChapter.title,
      chapter_index: currentChapter.chapter_index,
      progress_percent,
      location_cfi: getVisibleCfi(),
      location_type: "page",
      event_type: "progress_update",
    }).then(() => {
      invalidateProgressQueries();
    }).catch(reportProgressFailure);
  }, [user, pageIndex, effectiveReadingMode, currentChapter?.id, book_id, chapters.length, book]);
  const { data: seriesContext } = useBookSeriesQuery(book_id || "");
  const { data: audiobookChapters } = useAudiobookChaptersQuery(book_id || "");
  const readListNext = useReadListNextQuery(readListId, book_id).data;

  const canGoPrev = currentChapterIndex > 0;
  const hasNextChapter =
    currentChapterIndex >= 0 && currentChapterIndex < chapters.length - 1;

  const nextInReadList = !hasNextChapter && readListNext?.has_next ? readListNext.book : undefined;
  const nextInSeries = !hasNextChapter && !nextInReadList && seriesContext?.next ? seriesContext.next : undefined;
  const canGoNext = hasNextChapter || !!nextInReadList || !!nextInSeries;

  const nextTooltip = useMemo(() => {
    if (hasNextChapter) return t("reader.next_chapter", "Next Chapter");
    if (nextInReadList) return t("reader.readlist_next", "Next: {{title}}", { title: nextInReadList.title });
    if (nextInSeries) return t("reader.readlist_next", "Next: {{title}}", { title: nextInSeries.title });
    return undefined;
  }, [hasNextChapter, nextInReadList, nextInSeries, t]);

  const goToNextInReadList = () => {
    if (!readListNext || !readListNext.book) return;
    navigate(`/reader/${readListNext.book.id}?readlist=${readListId}`);
  };

  const goToNextInSeries = () => {
    if (!seriesContext?.next) return;
    navigate(`/reader/${seriesContext.next.book_id}`);
  };

  const handleNext = () => {
    if (!currentChapter) return;
    const idx = chapters.findIndex(c => c.id === currentChapter.id);
    if (idx >= 0 && idx < chapters.length - 1) {
      loadChapter(chapters[idx + 1]);
    } else if (nextInReadList) {
      goToNextInReadList();
    } else if (nextInSeries) {
      goToNextInSeries();
    }
  };

  const handlePrev = (startAtEnd = false) => {
    if (!currentChapter) return;
    const idx = chapters.findIndex(c => c.id === currentChapter.id);
    if (idx > 0) {
      const isPagedMode = effectiveReadingMode === "single" || effectiveReadingMode === "double";
      if (startAtEnd || isPagedMode) {
        pendingLandingRef.current = "end";
      } else {
        pendingLandingRef.current = null;
      }
      loadChapter(chapters[idx - 1]);
    }
  };

  const handleReaderBack = () => {
    if (window.history.length > 1) {
      navigate(-1);
      return;
    }
    navigate("/");
  };

  const restoreFocus = () => {
    const el = lastFocusedControlRef.current;
    if (el && document.contains(el)) {
      el.focus();
      lastFocusedControlRef.current = null;
    }
  };

  const closeSearch = () => {
    setSearchOpen(false);
    restoreFocus();
  };

  const handleKeyDown = (e: KeyboardEvent) => {
    const target = e.target as HTMLElement | null;
    if (target && (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.tagName === "SELECT" || target.isContentEditable)) {
      return;
    }
    if (e.key === "Escape") {
      if (selectionRange) {
        setSelectionRange(null);
        window.getSelection()?.removeAllRanges();
        return;
      }
      if (searchOpen) { closeSearch(); return; }
      if (settingsOpen) { setSettingsOpen(false); restoreFocus(); return; }
      if (sidebarOpen) { setSidebarOpen(false); restoreFocus(); return; }
      return;
    }
    if (scrollLayout || isPdfAudio) return;
    if (e.key === "ArrowLeft") {
      e.preventDefault();
      rtlPaging ? handlePageNext() : handlePagePrev();
    } else if (e.key === "ArrowRight") {
      e.preventDefault();
      rtlPaging ? handlePagePrev() : handlePageNext();
    }
  };

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
    // ponytail: handler closes over live state; rebind on relevant deps
  }, [selectionRange, searchOpen, settingsOpen, sidebarOpen, scrollLayout, isPdfAudio, rtlPaging, pageIndex]);

  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center bg-base-100 text-base-content">
        <span className="loading loading-spinner loading-lg text-primary"></span>
      </div>
    );
  }

  if (!book) {
    return (
      <div className="flex flex-col h-screen items-center justify-center bg-base-100 text-base-content">
        <h2 className="text-2xl font-bold mb-4">{t('reader.book_not_found', 'Book not found')}</h2>
        <button className="btn btn-primary" onClick={() => navigate("/")}>{t('reader.go_back', 'Go Back')}</button>
      </div>
    );
  }

  return (
    <div className={`reader-ui reader-theme-${theme} drawer h-screen overflow-hidden ${readerBg}`}>
      <input id="reader-drawer" type="checkbox" className="drawer-toggle" checked={sidebarOpen} onChange={(e) => setSidebarOpen(e.target.checked)} />

      <div className="drawer-content flex flex-col h-screen overflow-hidden relative">
        <ReaderTopBar
          t={t}
          title={currentChapter?.title || t("reader.reading", "Reading")}
          headerBg={headerBg}
          canGoPrev={canGoPrev}
          canGoNext={canGoNext}
          settingsOpen={settingsOpen}
          theme={theme}
          fontFamily={fontFamily}
          fontSize={fontSize}
          lineHeight={lineHeight}
          maxWidth={maxWidth}
          effectiveReadingMode={effectiveReadingMode}
          canUseDoubleMode={canUseDoubleMode}
          isVisualContent={isVisualContent}
          textAlign={textAlign}
          readingDirection={readingDirection}
          pageFit={pageFit}
          pageAnimation={pageAnimation}
          onPrev={handlePrev}
          onNext={handleNext}
          setSettingsOpen={(open) => {
            if (open) lastFocusedControlRef.current = document.activeElement as HTMLElement | null;
            setSettingsOpen(open);
            if (!open) restoreFocus();
          }}
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
          ttsSupported={isSupported && allowTTS}
          ttsPlaying={isPlaying}
          ttsPaused={isPaused}
          onTtsPlayPause={handleTtsPlayPause}
          onTtsStop={stop}
          ttsVoices={voices}
          ttsSelectedVoice={selectedVoice}
          setTtsSelectedVoice={handleTtsVoiceChange}
          ttsRate={rate}
          setTtsRate={handleTtsRateChange}
          autoScrollActive={autoScrollActive}
          onToggleAutoScroll={onToggleAutoScroll}
          onOpenSearch={() => {
            lastFocusedControlRef.current = document.activeElement as HTMLElement | null;
            setSearchOpen(true);
          }}
          isAudio={isAudio}
          nextTooltip={nextTooltip}
          isComic={isComicFormat}
          comicCurrentPage={comicPage}
          comicTotalPages={comicTotalPages}
          onComicPageJump={(p) => setComicPage(p)}
          activeImageTarget={effectiveActiveImageTarget}
          onSaveImageBookmark={handleSaveImageBookmark}
          onOpenQuoteCard={(text, imageUrl) => {
            setQuoteModalText(text || "");
            setQuoteModalImageUrl(imageUrl);
            setQuoteModalOpen(true);
          }}
          onCloseImageTarget={() => setActiveImageTarget(null)}
        />

        {/* Reader Scrollable Area */}
        {(() => {
          return (
            <div 
              ref={contentRef}
              className={`flex-1 min-h-0 ${
                isPdfAudio
                  ? 'overflow-hidden flex flex-col p-0'
                  : effectiveReadingMode === "webtoon"
                    ? 'overflow-y-auto p-0'
                    : scrollLayout
                      ? 'overflow-y-auto pt-6 pb-24 px-4 sm:px-8'
                      : isComicFormat
                        ? 'overflow-hidden flex flex-col p-0'
                        : 'overflow-hidden flex flex-col pt-4 pb-6 px-4 sm:px-20'
              } relative`}
              onClick={() => setSettingsOpen(false)}
              onClickCapture={handleContentClickCapture}
              onScroll={handleScroll}
            >
              <div
                ref={pageFrameRef}
                data-align={textAlign}
                className={`w-full mx-auto ${isPdfAudio ? 'h-full flex-1 min-h-0 flex flex-col' : scrollLayout ? 'h-auto' : 'flex-1 min-h-0 flex flex-col'}`}
                style={{
                  maxWidth: isPdfAudio || effectiveReadingMode === "webtoon" || isVisualContent ? '100%' : (maxWidth >= 1600 ? '100%' : `${maxWidth}px`),
                  fontSize: `${fontSize}px`,
                  lineHeight: lineHeight,
                  "--reader-font-family": fontFamily,
                  "--reader-text-align": textAlign === "original" ? undefined : textAlign,
                  "--reader-link-color": linkColor,
                  "--reader-link-color-hover": linkColorHover,
                  "--reader-page-gap": `${READER_PAGE_GAP}px`,
                  "--reader-page-width": pageWidth > 0 ? `${pageWidth}px` : undefined,
                  "--reader-page-height": "calc(100vh - 100px)",
                  "--reader-line-height": lineHeight,
                  "--reader-content-measure": `${READER_CONTENT_MEASURE}ch`
                } as React.CSSProperties}
              >
                {isPdf ? (
                  <iframe
                    title={book.title}
                    src={offlineRawUrl || rawFileUrl}
                    className="reader-pdf-frame w-full h-full flex-1 border-0"
                  />
                ) : isAudio ? (
                  <div className="flex-1 w-full h-full">
                    <AudioPlayer
                      rawUrl={offlineRawUrl || rawFileUrl}
                      title={book.title}
                      author={book.author_name || "Unknown"}
                      cover_url={book.cover_url || `${API_BASE}/books/${book.id}/cover`}
                      chapters={audiobookChapters?.map((c) => ({ title: c.title, start_sec: c.start_sec, end_sec: c.end_sec }))}
                      initialTime={pendingFragmentRef.current?.startsWith("audio:") ? parseFloat(pendingFragmentRef.current.slice(6)) : 0}
                      seekToTime={audioSeekTime}
                      onBookmarksChange={setAudioBookmarks}
                      onRegisterDeleteBookmark={(fn) => {
                        audioDeleteBookmarkRef.current = fn;
                      }}
                      onTimeUpdate={(time) => {
                        pendingFragmentRef.current = `audio:${time}`;
                        const now = Date.now();
                        if (now - lastAudioProgressRef.current < 10_000) return;
                        lastAudioProgressRef.current = now;
                        const chapterPosition = chapters.findIndex((c) => c.id === currentChapter?.id);
                        const progress_percent = chapterPosition >= 0
                          ? Math.round(((chapterPosition + 1) / chapters.length) * 100)
                          : 0;

                        void featureService.recordReadingActivity({
                          book_id: book_id || '',
                          file_id: file_id || '',
                          chapter_id: currentChapter?.id || '',
                          chapter_title: currentChapter?.title || '',
                          chapter_index: currentChapter?.chapter_index || 0,
                          progress_percent,
                          location_cfi: String(time),
                          location_type: "audio",
                          event_type: "progress_update",
                        }).catch(() => {});
                      }}
                    />
                    {readerETA && readerETA.eta_minutes > 0 && (
                      <p className="text-center text-xs text-(--reader-ui-muted) mt-2">
                        {t("reader.eta_left", "~{{minutes}} min left", { minutes: Math.max(1, readerETA.eta_minutes) })}
                      </p>
                    )}
                  </div>
                ) : htmlContent && isComicFormat ? (
                  <ComicReader
                    htmlContent={htmlContent}
                    onContentClick={handleContentClick}
                    onImageClick={(target) => {
                      setActiveImageTarget({
                        ...target,
                        chapter_id: currentChapter?.id,
                        chapter_title: currentChapter?.title,
                      });
                    }}
                    onPrevChapter={canGoPrev ? () => handlePrev(true) : undefined}
                    onNextChapter={canGoNext ? handleNext : undefined}
                    canGoPrevChapter={canGoPrev}
                    canGoNextChapter={canGoNext}
                    initialLanding={pendingLandingRef.current}
                    currentPage={comicPage}
                    onPageChange={setComicPage}
                    onTotalPagesChange={setComicTotalPages}
                  />
                ) : htmlContent ? (
                  <ReaderContent
                    htmlContent={htmlContent}
                    proseClass={proseClass}
                    textAlign={textAlign}
                    effectiveReadingMode={effectiveReadingMode}
                    readingDirection={readingDirection}
                    pageWidth={pageWidth}
                    columnsRef={columnsRef}
                    onContentClick={handleContentClick}
                  />
                ) : (
                  <div className="flex justify-center items-center h-64 opacity-50">
                    {t('common.loading', 'Loading content...')}
                  </div>
                )}
                
                {!isPdf && !isAudio && scrollLayout && nextInReadList && (
                  <button
                    className="reader-action-btn btn btn-sm mx-auto mt-6 flex gap-1.5 rounded-xl"
                    onClick={goToNextInReadList}
                  >
                    {t("reader.readlist_next", "Next: {{title}}", { title: nextInReadList.title })}
                  </button>
                )}

                {!isPdf && !isAudio && scrollLayout && !nextInReadList && nextInSeries && (
                  <div className="mx-auto mt-8 flex max-w-md flex-col items-center gap-3 rounded-2xl border border-(--reader-ui-border) bg-(--reader-ui-surface-strong) text-(--reader-ui-text) p-5 shadow-sm text-center">
                    <p className="text-xs font-bold uppercase tracking-wider text-(--reader-ui-accent)">
                      {t("reader.series_next_label", "Next Volume in Series")}
                    </p>
                    <div className="flex items-center gap-3.5 w-full text-left bg-(--reader-ui-soft) p-3 rounded-xl border border-(--reader-ui-border)">
                      {nextInSeries.cover_url ? (
                        <img
                          src={getMediaUrl(nextInSeries.cover_url)}
                          alt={nextInSeries.title}
                          className="h-16 aspect-[3/4.2] rounded-lg object-cover bg-black/10 shrink-0 shadow-2xs"
                        />
                      ) : (
                        <div className="h-16 aspect-[3/4.2] rounded-lg bg-(--reader-ui-accent-soft) grid place-items-center text-(--reader-ui-accent) shrink-0">
                          <BookOpen className="w-6 h-6" />
                        </div>
                      )}
                      <div className="min-w-0 flex-1">
                        <p className="text-sm font-bold text-(--reader-ui-text) truncate">
                          {nextInSeries.title}
                        </p>
                        <p className="text-xs text-(--reader-ui-muted) truncate mt-0.5">
                          {nextInSeries.series_name} {nextInSeries.series_index ? `#${nextInSeries.series_index}` : ""}
                        </p>
                      </div>
                    </div>
                    <button
                      className="reader-action-btn btn btn-sm w-full gap-2 rounded-xl mt-1"
                      onClick={goToNextInSeries}
                    >
                      <BookOpen className="w-4 h-4" />
                      {t("reader.read_next_volume", "Read Next Volume")}
                      <ChevronRight className="w-4 h-4" />
                    </button>
                  </div>
                )}

                {!isPdf && !isAudio && scrollLayout && effectiveReadingMode !== "webtoon" && !isVisualContent && (
                  <ReaderPageControls
                    t={t}
                    mode="footer"
                    canGoPrev={canGoPrev}
                    canGoNext={canGoNext}
                    onPrev={handlePrev}
                    onNext={handleNext}
                  />
                )}
                {!isPdf && !isAudio && scrollLayout && readerETA && readerETA.eta_minutes > 0 && (
                  <p className="text-center text-xs text-(--reader-ui-muted) mt-2">
                    {t("reader.eta_left", "~{{minutes}} min left", { minutes: Math.max(1, readerETA.eta_minutes) })}
                  </p>
                )}
              </div>

              {!isPdf && !isAudio && !scrollLayout && htmlContent && (
                <ReaderPageControls
                  t={t}
                  mode="floating"
                  canGoPrev={isComicFormat ? (comicPage > 0 || canGoPrev) : true}
                  canGoNext={isComicFormat ? (comicPage + comicStep < comicTotalPages || canGoNext) : true}
                  onPrev={isComicFormat ? (rtlPaging ? handleComicPageNext : handleComicPagePrev) : handlePagePrev}
                  onNext={isComicFormat ? (rtlPaging ? handleComicPagePrev : handleComicPageNext) : handlePageNext}
                />
              )}
            </div>
          );
        })()}
      </div>
      
      <ReaderSidebar
        t={t}
        book={book}
        chapters={chapters}
        currentChapter={currentChapter}
        sidebarBg={sidebarBg}
        sidebarRef={sidebarRef}
        onClose={() => { setSidebarOpen(false); restoreFocus(); }}
        onBack={handleReaderBack}
        onSelectChapter={(chapter) => {
          pendingTextOffsetRef.current = null;
          void loadChapter(chapter);
          setSidebarOpen(false);
          restoreFocus();
        }}
        highlights={allowHighlights ? highlights : undefined}
        onUpdateHighlight={allowHighlights ? (id, color, note) => void updateHighlight(id, color, note) : undefined}
        onDeleteHighlight={allowHighlights ? (id) => void removeHighlight(id) : undefined}
        onSelectHighlight={handleSelectHighlight}
        isAudio={isAudio}
        audioBookmarks={audioBookmarks}
        onSelectAudioBookmark={(time) => {
          setAudioSeekTime(time);
          setTimeout(() => setAudioSeekTime(null), 100);
        }}
        onDeleteAudioBookmark={(id) => {
          audioDeleteBookmarkRef.current?.(id);
        }}
        isVisualContent={isVisualContent}
        imageBookmarks={imageBookmarks}
        onSelectImageBookmark={handleSelectImageBookmark}
        onDeleteImageBookmark={handleDeleteImageBookmark}
        onOpenQuoteCard={(text, imageUrl) => {
          setQuoteModalText(text || "");
          setQuoteModalImageUrl(imageUrl);
          setQuoteModalOpen(true);
        }}
        nextInSeries={seriesContext?.next}
        nextInReadList={readListNext?.has_next ? readListNext.book : undefined}
        onGoToNextInSeries={goToNextInSeries}
        onGoToNextInReadList={goToNextInReadList}
      />

      {selectionRange && (
        <ReaderSelectionToolbar
          t={t}
          toolbarPos={toolbarPos}
          isSupported={isSupported && allowTTS}
          selectedText={savedSelectionRef.current?.selectedText || selectionRange?.toString() || window.getSelection()?.toString() || ""}
          onReadSelection={handleReadSelection}
          onReadFromHere={handleReadFromHere}
          onCopyText={handleCopyText}
          onHighlight={allowHighlights ? handleHighlight : undefined}
          onOpenQuoteCard={(text) => {
            const txt = (text || savedSelectionRef.current?.selectedText || selectionRange?.toString() || window.getSelection()?.toString() || "").trim();
            if (txt) {
              setQuoteModalText(txt);
              setQuoteModalImageUrl(undefined);
              setQuoteModalOpen(true);
            }
          }}
        />
      )}

      <QuoteCardModal
        open={quoteModalOpen}
        quote={quoteModalText}
        imageUrl={quoteModalImageUrl}
        bookTitle={book?.title}
        bookAuthor={book?.author_name}
        bookCover={book?.cover_url ? getMediaUrl(book.cover_url) : undefined}
        chapterTitle={currentChapter?.title}
        onClose={() => {
          setQuoteModalOpen(false);
          setQuoteModalImageUrl(undefined);
        }}
      />

      {searchOpen && book_id && (
        <div className="fixed top-16 right-6 z-50 animate-fade-in">
          <ReaderInBookSearch
            book_id={book_id}
            onClose={closeSearch}
            onSelectResult={(chapter_id, offset) => {
              const ch = chapters.find((c) => c.id === chapter_id);
              if (ch) {
                pendingTextOffsetRef.current = offset > 0 ? offset : null;
                void loadChapter(ch);
              }
              closeSearch();
            }}
          />
        </div>
      )}
    </div>
  );
};

export const ReaderWorkspace = () => {
  const { book_id } = useParams<{ book_id: string }>();
  return <ReaderWorkspaceInner key={book_id} />;
};
