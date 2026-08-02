import { ComicReader, ReaderContent, ReaderPageControls, ReaderSelectionToolbar, ReaderSidebar, ReaderTopBar } from "@/components/reader";
import { ReaderInBookSearch } from "@/components/reader/ReaderInBookSearch";
import { API_BASE } from "@/config/api";
import { getReaderThemeClasses } from "@/config/readerTheme";
import { featureService, readerService } from "@/services";
import { useAuthStore, useReaderStore } from "@/stores";
import type { Chapter } from "@/types";
import React, { useEffect, useMemo, useState, useRef } from "react";
import { useTranslation } from "react-i18next";
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
import { clearHighlight, highlightTextRangeFromNode, extractTextFromHtml, scrollToTextOffset, type TtsStartPoint, type SavedSelection } from "@/lib/readerHighlight";

import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { useShallow } from "zustand/react/shallow";

import { MIN_DOUBLE_PAGE_WIDTH, READER_CONTENT_MEASURE, READER_PAGE_GAP } from "@/constants";
import { usePublicSettings } from "@/hooks/useSettings";
import { hasPermission } from "@/utils/permission";
import { isVisualChapter } from "@/utils/readerHtml";

export const ReaderWorkspace = () => {
  const { bookId } = useParams<{ bookId: string }>();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const fileId = searchParams.get("file_id") || undefined;
  
  const { t } = useTranslation();
  const [searchOpen, setSearchOpen] = useState(false);

  const ttsStartPointRef = useRef<TtsStartPoint | null>(null);
  const savedSelectionRef = useRef<SavedSelection | null>(null);

  // Declared up here because the TTS boundary callback and the extracted reader
  // hooks below all close over them.
  const contentRef = useRef<HTMLDivElement>(null);
  const columnsRef = useRef<HTMLDivElement>(null);
  const pageFrameRef = useRef<HTMLDivElement>(null);
  const sidebarRef = useRef<HTMLElement>(null);
  const pendingFragmentRef = useRef<string | null>(null);
  const pendingTextOffsetRef = useRef<number | null>(null);
  const lastFocusedControlRef = useRef<HTMLElement | null>(null);
  const ttsOffsetRef = useRef<number>(0);

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
    readingMode,
    readingDirection,
    pageFit,
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
    setReadingMode,
    setReadingDirection,
    setPageFit,
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
    readingMode: state.readingMode,
    readingDirection: state.readingDirection,
    pageFit: state.pageFit,
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
    setReadingMode: state.setReadingMode,
    setReadingDirection: state.setReadingDirection,
    setPageFit: state.setPageFit,
    setPageIndex: state.setPageIndex,
    setPageFrameWidth: state.setPageFrameWidth,
    setTtsVoiceName: state.setTtsVoiceName,
    setTtsRate: state.setTtsRate,
    resetSettings: state.resetSettings,
    reset: state.reset,
  })));


  useReadingStats(book?.id, !settingsOpen);

  const publicSettings = usePublicSettings();
  const guestPerms = publicSettings?.guest_permissions;
  const allowTTS = hasPermission(user, "book.tts", book?.libraryId, guestPerms);
  const allowHighlights = hasPermission(user, "book.highlight", book?.libraryId, guestPerms);
  const { highlights, addHighlight, updateHighlight, removeHighlight } = useHighlights(book?.id || '', currentChapter?.id, allowHighlights);

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
  });

  const [ambientColor, setAmbientColor] = useState<string>("transparent");

  useEffect(() => {
    if (!book || !book.coverUrl) {
      setAmbientColor("transparent");
      return;
    }
    const fac = new FastAverageColor();
    const img = new Image();
    img.crossOrigin = "Anonymous";
    img.src = book.coverUrl;
    img.onload = () => {
      try {
        const color = fac.getColor(img);
        setAmbientColor(theme === 'dark' ? color.hex : color.rgba);
      } catch (e) {
        setAmbientColor("transparent");
      }
    };
    return () => { fac.destroy(); };
  }, [book, theme]);

  

  

  const { isScrolling: autoScrollActive, toggleScroll: onToggleAutoScroll, updateSpeed } = useAutoScroll(contentRef);

  const doublePageWidth = pageFrameWidth > 0 ? Math.floor((pageFrameWidth - READER_PAGE_GAP) / 2) : 0;
  const canUseDoubleMode = pageFrameWidth === 0 || doublePageWidth >= MIN_DOUBLE_PAGE_WIDTH;
  const effectiveReadingMode = readingMode === "double" && !canUseDoubleMode ? "single" : readingMode;
  const scrollLayout = effectiveReadingMode === "scroll" || effectiveReadingMode === "webtoon";
  const isVisualContent = useMemo(() => isVisualChapter(htmlContent), [htmlContent]);
  const rtlPaging = isVisualContent && readingDirection === "rtl";
  const activeFile = fileId ? book?.files?.find(f => f.id === fileId) : book?.files?.[0];
  const isPdf = !!(activeFile?.format.match(/^pdf$/i) || currentChapter?.contentPath?.toLowerCase().endsWith(".pdf"));
  const isAudio = !!activeFile?.format.match(/^(mp3|m4a|m4b|flac)$/i);
  const isPdfAudio = isPdf || isAudio;
  const visiblePages = effectiveReadingMode === "double" ? 2 : 1;
  const pageWidth = scrollLayout || pageFrameWidth === 0
    ? 0
    : Math.max(1, Math.floor((pageFrameWidth - READER_PAGE_GAP * (visiblePages - 1)) / visiblePages));

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
    htmlContent,
    maxWidth,
    scrollLayout,
    effectiveReadingMode,
    rtlPaging,
    pageIndex,
    setPageIndex,
    setPageFrameWidth,
    onChapterNext: () => handleNext(),
    onChapterPrev: () => handlePrev(),
  });

  useEffect(() => {
    if (!sidebarOpen) return;

    const handlePointerDownOutside = (event: PointerEvent) => {
      const target = event.target;
      if (!(target instanceof Node)) return;
      if (sidebarRef.current?.contains(target)) return;
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
  });

  useEffect(() => {
    if (!bookId) return;
    
    const bootstrap = async () => {
      try {
        const [res, progressRes] = await Promise.allSettled([
          readerService.getBootstrap(bookId, fileId),
          user ? featureService.getReadingProgress(bookId) : Promise.reject("guest"),
        ]);

        if (res.status === "fulfilled" && res.value.status && res.value.data) {
          setBook(res.value.data.book);
          const sorted = [...res.value.data.chapters].sort((a, b) => a.chapterIndex - b.chapterIndex);
          setChapters(sorted);
          if (sorted.length > 0) {
            let targetChapter = sorted[0];
            let locationCfi: string | undefined = undefined;
            const startOver = searchParams.get("start_over") === "true";

            if (!startOver && progressRes.status === "fulfilled" && progressRes.value.status && progressRes.value.data) {
              const progress = progressRes.value.data;
              const found = sorted.find(ch => ch.id === progress.chapterId);
              if (found) {
                targetChapter = found;
                if (progress.locationType === "scroll" && progress.locationCfi) {
                  locationCfi = `scroll:${progress.locationCfi}`;
                } else if (progress.locationType === "page" && progress.locationCfi) {
                  locationCfi = `page:${progress.locationCfi}`;
                } else if (progress.locationType === "audio" && progress.locationCfi) {
                  locationCfi = `audio:${progress.locationCfi}`;
                } else if (progress.locationCfi) {
                  locationCfi = progress.locationCfi;
                }
              }
            }

            if (locationCfi) {
              pendingFragmentRef.current = locationCfi;
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
  }, [bookId, fileId]);

  const loadChapter = async (chapter: Chapter) => {
    if (!bookId) return;
    setCurrentChapter(chapter);
    setHtmlContent("");
    stop();
    try {
      const html = await readerService.getChapterHtml(bookId, chapter.id, fileId);
      setHtmlContent(html);
      if (contentRef.current) {
        contentRef.current.scrollTop = 0;
      }
    } catch (err) {
      console.error("Failed to load chapter content", err);
      setHtmlContent(`<div class='text-error p-4'>${t('common.error', 'Failed to load chapter content.')}</div>`);
    }
  };

  useEffect(() => {
    if (!htmlContent || !pendingFragmentRef.current) return;
    const fragment = pendingFragmentRef.current;
    pendingFragmentRef.current = null;
    requestAnimationFrame(() => {
      if (fragment.startsWith("scroll:") && contentRef.current) {
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
    if (!user || !currentChapter || !bookId) return;
    const progressPercent = computeProgressPercent();

    void featureService.recordReadingActivity({
      bookId,
      fileId,
      chapterId: currentChapter.id,
      chapterTitle: currentChapter.title,
      chapterIndex: currentChapter.chapterIndex,
      progressPercent,
      eventType: "chapter_open",
    }).then(() => {
      void queryClient.invalidateQueries({ queryKey: ["reading"] });
      void queryClient.invalidateQueries({ queryKey: ["books"] });
      void queryClient.invalidateQueries({ queryKey: ["trackerReadingProgress"] });
      void queryClient.invalidateQueries({ queryKey: ["bookUserState"] });
    }).catch((error) => {
      console.debug("Failed to record reading activity", error);
    });
  }, [user, bookId, fileId, currentChapter?.id, chapters.length]);

  const scrollTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleScroll = () => {
    if (!user || !scrollLayout || !contentRef.current || !currentChapter || !bookId) return;

    const scrollTop = contentRef.current.scrollTop;
    if (scrollTimeoutRef.current) {
      clearTimeout(scrollTimeoutRef.current);
    }

    scrollTimeoutRef.current = setTimeout(() => {
      const progressPercent = computeProgressPercent();

      void featureService.recordReadingActivity({
        bookId,
        fileId,
        chapterId: currentChapter.id,
        chapterTitle: currentChapter.title,
        chapterIndex: currentChapter.chapterIndex,
        progressPercent,
        locationCfi: String(scrollTop),
        locationType: "scroll",
        eventType: "progress_update",
      }).then(() => {
        void queryClient.invalidateQueries({ queryKey: ["reading"] });
        void queryClient.invalidateQueries({ queryKey: ["books"] });
        void queryClient.invalidateQueries({ queryKey: ["trackerReadingProgress"] });
        void queryClient.invalidateQueries({ queryKey: ["bookUserState"] });
      }).catch(console.debug);
    }, 2000);
  };

  useEffect(() => {
    if (!user || scrollLayout || !currentChapter || !bookId) return;

    const progressPercent = computeProgressPercent();

    void featureService.recordReadingActivity({
      bookId,
      fileId,
      chapterId: currentChapter.id,
      chapterTitle: currentChapter.title,
      chapterIndex: currentChapter.chapterIndex,
      progressPercent,
      locationCfi: String(pageIndex),
      locationType: "page",
      eventType: "progress_update",
    }).then(() => {
      void queryClient.invalidateQueries({ queryKey: ["reading"] });
      void queryClient.invalidateQueries({ queryKey: ["books"] });
      void queryClient.invalidateQueries({ queryKey: ["trackerReadingProgress"] });
      void queryClient.invalidateQueries({ queryKey: ["bookUserState"] });
    }).catch(console.debug);
  }, [user, pageIndex, effectiveReadingMode, currentChapter?.id, bookId, chapters.length]);

  const handleNext = () => {
    if (!currentChapter) return;
    const idx = chapters.findIndex(c => c.id === currentChapter.id);
    if (idx >= 0 && idx < chapters.length - 1) {
      loadChapter(chapters[idx + 1]);
    }
  };

  const handlePrev = () => {
    if (!currentChapter) return;
    const idx = chapters.findIndex(c => c.id === currentChapter.id);
    if (idx > 0) {
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

  const {
    readerBg,
    proseClass,
    sidebarBg,
    headerBg,
    linkColor,
    linkColorHover,
  } = getReaderThemeClasses(theme);
  const currentChapterIndex = chapters.findIndex(
    (chapter) => chapter.id === currentChapter?.id,
  );
  const canGoPrev = currentChapterIndex > 0;
  const canGoNext =
    currentChapterIndex >= 0 && currentChapterIndex < chapters.length - 1;

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
          readingDirection={readingDirection}
          pageFit={pageFit}
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
          setReadingMode={setReadingMode}
          setReadingDirection={setReadingDirection}
          setPageFit={setPageFit}
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
        />

        {/* Reader Scrollable Area */}
        {(() => {
          return (
            <div 
              ref={contentRef}
              className={`flex-1 ${
                isPdf
                  ? 'overflow-hidden flex flex-col pt-14 p-0'
                  : scrollLayout
                    ? 'overflow-y-auto pt-20 pb-24 px-4 sm:px-8' 
                    : 'overflow-hidden flex flex-col pt-14 pb-6 px-4 sm:px-20'
              } relative`}
              onClick={() => setSettingsOpen(false)}
              onScroll={handleScroll}
            >
              <div 
                ref={pageFrameRef}
                className={`w-full mx-auto ${isPdf ? 'h-full flex-1 min-h-0 flex flex-col' : scrollLayout ? 'h-auto' : 'flex-1 min-h-0 flex flex-col'}`}
                style={{ 
                  maxWidth: isPdf ? '100%' : (maxWidth >= 1600 ? '100%' : `${maxWidth}px`),
                  fontSize: `${fontSize}px`,
                  lineHeight: lineHeight,
                  "--reader-font-family": fontFamily,
                  "--reader-link-color": linkColor,
                  "--reader-link-color-hover": linkColorHover,
                  "--reader-page-gap": `${READER_PAGE_GAP}px`,
                  "--reader-page-width": pageWidth > 0 ? `${pageWidth}px` : undefined,
                  "--reader-line-height": lineHeight,
                  "--reader-content-measure": `${READER_CONTENT_MEASURE}ch`
                } as React.CSSProperties}
              >
                {isPdf ? (
                  <iframe
                    title={book.title}
                    src={`${API_BASE}/reader/${encodeURIComponent(bookId || "")}/file?file_id=${encodeURIComponent(activeFile?.id || fileId || "")}`}
                    className="reader-pdf-frame w-full h-full flex-1 border-0"
                  />
                ) : isAudio ? (
                  <div className="flex-1 w-full h-full pb-32">
                    <AudioPlayer 
                      rawUrl={`${API_BASE}/reader/${encodeURIComponent(bookId || "")}/file?file_id=${encodeURIComponent(activeFile?.id || fileId || "")}`}
                      title={book.title}
                      author={book.authorName || "Unknown"}
                      coverUrl={book.coverUrl || `/api/v1/books/${book.id}/cover`}
                      initialTime={pendingFragmentRef.current?.startsWith("audio:") ? parseFloat(pendingFragmentRef.current.slice(6)) : 0}
                      onTimeUpdate={(time) => {
                        pendingFragmentRef.current = `audio:${time}`;
                        const chapterPosition = chapters.findIndex((c) => c.id === currentChapter?.id);
                        const progressPercent = chapterPosition >= 0
                          ? Math.round(((chapterPosition + 1) / chapters.length) * 100)
                          : 0;
                  
                        void featureService.recordReadingActivity({
                          bookId: bookId || '',
                          fileId: fileId || '',
                          chapterId: currentChapter?.id || '',
                          chapterTitle: currentChapter?.title || '',
                          chapterIndex: currentChapter?.chapterIndex || 0,
                          progressPercent,
                          locationCfi: String(time),
                          locationType: "audio",
                          eventType: "progress_update",
                        }).catch(() => {});
                      }}
                    />
                  </div>
                ) : htmlContent && effectiveReadingMode === "webtoon" ? (
                  <ComicReader
                    htmlContent={htmlContent}
                    onContentClick={handleContentClick}
                  />
                ) : htmlContent ? (
                  <ReaderContent
                    htmlContent={htmlContent}
                    proseClass={proseClass}
                    effectiveReadingMode={effectiveReadingMode}
                    readingDirection={readingDirection}
                    pageFit={pageFit}
                    pageWidth={pageWidth}
                    columnsRef={columnsRef}
                    onContentClick={handleContentClick}
                  />
                ) : (
                  <div className="flex justify-center items-center h-64 opacity-50">
                    {t('common.loading', 'Loading content...')}
                  </div>
                )}
                
                {!isPdf && scrollLayout && (
                  <ReaderPageControls
                    t={t}
                    mode="footer"
                    canGoPrev={canGoPrev}
                    canGoNext={canGoNext}
                    onPrev={handlePrev}
                    onNext={handleNext}
                  />
                )}
              </div>

              {!isPdf && !scrollLayout && htmlContent && (
                <ReaderPageControls
                  t={t}
                  mode="floating"
                  canGoPrev
                  canGoNext
                  onPrev={handlePagePrev}
                  onNext={handlePageNext}
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
      />

      {selectionRange && (
        <ReaderSelectionToolbar
          t={t}
          toolbarPos={toolbarPos}
          isSupported={isSupported && allowTTS}
          onReadSelection={handleReadSelection}
          onReadFromHere={handleReadFromHere}
          onCopyText={handleCopyText}
          onHighlight={allowHighlights ? handleHighlight : undefined}
        />
      )}

      {searchOpen && bookId && (
        <div className="fixed top-16 right-6 z-50 animate-fade-in">
          <ReaderInBookSearch
            bookId={bookId}
            onClose={closeSearch}
            onSelectResult={(chapterId, offset) => {
              const ch = chapters.find((c) => c.id === chapterId);
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
