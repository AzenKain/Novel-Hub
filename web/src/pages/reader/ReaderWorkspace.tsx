import { ReaderContent, ReaderPageControls, ReaderSelectionToolbar, ReaderSidebar, ReaderTopBar } from "@/components/reader";
import { ReaderInBookSearch } from "@/components/reader/ReaderInBookSearch";
import { getReaderThemeClasses } from "@/config/readerTheme";
import { featureService, readerService } from "@/services";
import { useAuthStore, useReaderStore } from "@/stores";
import type { Chapter } from "@/types";
import React, { useEffect, useState, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useTTS } from "@/hooks/useTTS";
import { AudioPlayer } from "@/components/reader/AudioPlayer";
import { FastAverageColor } from "fast-average-color";
import { useHighlights } from "@/hooks/useHighlights";
import { useAutoScroll } from "@/hooks/useAutoScroll";
import { useReadingStats } from "@/hooks/useReadingStats";
import { clearHighlight, highlightTextRange, highlightTextRangeFromNode, extractTextFromHtml, getSelectionInfo, saveSelection, getTextFromHereFromSaved, type TtsStartPoint, type SavedSelection } from "@/lib/readerHighlight";

import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { useShallow } from "zustand/react/shallow";

import { MIN_DOUBLE_PAGE_WIDTH, READER_CONTENT_MEASURE, READER_PAGE_GAP } from "@/constants";

export const ReaderWorkspace = () => {
  const { bookId } = useParams<{ bookId: string }>();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const fileId = searchParams.get("file_id") || undefined;
  
  const { t } = useTranslation();
  const [searchOpen, setSearchOpen] = useState(false);

  const ttsStartPointRef = useRef<TtsStartPoint | null>(null);
  const savedSelectionRef = useRef<SavedSelection | null>(null);

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
    setPageIndex: state.setPageIndex,
    setPageFrameWidth: state.setPageFrameWidth,
    setTtsVoiceName: state.setTtsVoiceName,
    setTtsRate: state.setTtsRate,
    resetSettings: state.resetSettings,
    reset: state.reset,
  })));


  const { highlights, addHighlight, removeHighlight } = useHighlights(book?.id || '', currentChapter?.id);
  useReadingStats(book?.id, !settingsOpen); 

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

  // Sync stored TTS voice and rate from Zustand store
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

  // Clear highlight on unmount or when stopping TTS manually
  useEffect(() => {
    if (!isPlaying && !isPaused) {
      clearHighlight();
    }
  }, [isPlaying, isPaused]);

  
  // Floating toolbar state
  const [selectionRange, setSelectionRange] = useState<Range | null>(null);
  const [toolbarPos, setToolbarPos] = useState({ top: 0, left: 0 });

  useEffect(() => {
    const handleSelection = (e: Event) => {
      const targetNode = e.target as Node | null;
      const targetElem = targetNode?.nodeType === Node.ELEMENT_NODE
        ? (targetNode as HTMLElement)
        : targetNode?.parentElement;
      const isToolbar = !!targetElem?.closest?.('[data-reader-toolbar="true"]');

      if (isToolbar) {
        return;
      }

      setTimeout(() => {
        const selection = window.getSelection();

        if (selection && selection.rangeCount > 0 && !selection.isCollapsed) {
          const range = selection.getRangeAt(0);
          const container = columnsRef.current || contentRef.current;
          const commonNode = range.commonAncestorContainer.nodeType === Node.TEXT_NODE
            ? range.commonAncestorContainer.parentNode
            : range.commonAncestorContainer;
          if (container && commonNode && container.contains(commonNode)) {
            const saved = saveSelection(container, range);
            if (saved) {
              savedSelectionRef.current = saved;
              setSelectionRange(range.cloneRange());
              const rect = range.getBoundingClientRect();
              setToolbarPos({ top: Math.max(10, rect.top - 40), left: rect.left + rect.width / 2 });
              return;
            }
          }
        }
        savedSelectionRef.current = null;
        setSelectionRange(null);
      }, 20);
    };

    document.addEventListener("mouseup", handleSelection);
    document.addEventListener("keyup", handleSelection);
    return () => {
      document.removeEventListener("mouseup", handleSelection);
      document.removeEventListener("keyup", handleSelection);
    };
  }, []);

  useEffect(() => {
    if (!CSS.highlights) return;
    const highlightRanges = highlights.map((h: any) => {
      return new Range(); 
    });
  }, [highlights]);

  const handleHighlight = async (color: string) => {
    if (selectionRange) {
      const text = selectionRange.toString();
      await addHighlight(text, 0, text.length, color);
      window.getSelection()?.removeAllRanges();
      savedSelectionRef.current = null;
      setSelectionRange(null);
    }
  };

  const handleReadSelection = () => {
    const container = columnsRef.current || contentRef.current;
    const saved = savedSelectionRef.current;
    if (container && saved && saved.selectedText) {
      ttsStartPointRef.current = { textNodeIndex: saved.textNodeIndex, offset: saved.offset };
      stop();
      speak(saved.selectedText);
      savedSelectionRef.current = null;
      setSelectionRange(null);
    }
  };

  const handleReadFromHere = () => {
    const container = columnsRef.current || contentRef.current;
    const saved = savedSelectionRef.current;
    if (container && saved) {
      const textFromHere = getTextFromHereFromSaved(container, saved);
      if (textFromHere) {
        ttsStartPointRef.current = { textNodeIndex: saved.textNodeIndex, offset: saved.offset };
        stop();
        speak(textFromHere);
      }
      savedSelectionRef.current = null;
      setSelectionRange(null);
    }
  };

  const handleCopyText = () => {
    const saved = savedSelectionRef.current;
    const textToCopy = saved?.selectedText || selectionRange?.toString();
    if (textToCopy) {
      void navigator.clipboard.writeText(textToCopy);
      window.getSelection()?.removeAllRanges();
      savedSelectionRef.current = null;
      setSelectionRange(null);
    }
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

  

  

  const contentRef = useRef<HTMLDivElement>(null);
  const { isScrolling: autoScrollActive, toggleScroll: onToggleAutoScroll, updateSpeed } = useAutoScroll(contentRef);



  const columnsRef = useRef<HTMLDivElement>(null);
  const pageFrameRef = useRef<HTMLDivElement>(null);
  const sidebarRef = useRef<HTMLElement>(null);
  const pendingFragmentRef = useRef<string | null>(null);
  const ttsOffsetRef = useRef<number>(0);

  const doublePageWidth = pageFrameWidth > 0 ? Math.floor((pageFrameWidth - READER_PAGE_GAP) / 2) : 0;
  const canUseDoubleMode = pageFrameWidth === 0 || doublePageWidth >= MIN_DOUBLE_PAGE_WIDTH;
  const effectiveReadingMode = readingMode === "double" && !canUseDoubleMode ? "single" : readingMode;
  const visiblePages = effectiveReadingMode === "double" ? 2 : 1;
  const pageWidth = effectiveReadingMode === "scroll" || pageFrameWidth === 0
    ? 0
    : Math.max(1, Math.floor((pageFrameWidth - READER_PAGE_GAP * (visiblePages - 1)) / visiblePages));

  useEffect(() => {
    setPageIndex(0);
    if (contentRef.current) {
      contentRef.current.scrollLeft = 0;
      contentRef.current.scrollTop = 0;
    }
    if (columnsRef.current) {
      columnsRef.current.scrollLeft = 0;
      const body = columnsRef.current.querySelector("body");
      if (body) {
        body.scrollLeft = 0;
      }
    }
  }, [effectiveReadingMode, maxWidth, fontSize, fontFamily, lineHeight, htmlContent]);

  useEffect(() => {
    if (readingMode === "scroll") {
      setPageFrameWidth(0);
      return;
    }

    const frame = pageFrameRef.current;
    if (!frame) return;

    const updatePageFrameWidth = () => setPageFrameWidth(frame.clientWidth);

    updatePageFrameWidth();

    const resizeObserver = new ResizeObserver(updatePageFrameWidth);
    resizeObserver.observe(frame);
    window.addEventListener("resize", updatePageFrameWidth);

    return () => {
      resizeObserver.disconnect();
      window.removeEventListener("resize", updatePageFrameWidth);
    };
  }, [readingMode, maxWidth]);

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

  const getPagedScrollContainer = () => {
    const readerContent = columnsRef.current;
    if (!readerContent) return null;
    return readerContent.querySelector("body") || readerContent;
  };

  const getPagedScrollMetrics = () => {
    const container = getPagedScrollContainer();
    if (!container) return;

    const scrollStep = container.clientWidth + READER_PAGE_GAP;
    const maxIndex = Math.max(0, Math.ceil((container.scrollWidth - container.clientWidth) / scrollStep));
    return { container, scrollStep, maxIndex };
  };

  const scrollToPageIndex = (targetIndex: number) => {
    const metrics = getPagedScrollMetrics();
    if (!metrics) return;

    const { container, scrollStep, maxIndex } = metrics;
    const nextIndex = Math.min(Math.max(targetIndex, 0), maxIndex);

    container.scrollTo({
      left: nextIndex * scrollStep,
      behavior: "smooth",
    });
    setPageIndex(nextIndex);
  };

  const handlePageNext = () => {
    const metrics = getPagedScrollMetrics();
    if (metrics && pageIndex >= metrics.maxIndex) {
      handleNext();
      return;
    }
    scrollToPageIndex(pageIndex + 1);
  };

  const handlePagePrev = () => {
    if (pageIndex <= 0) {
      handlePrev();
      return;
    }
    scrollToPageIndex(pageIndex - 1);
  };

  const handleContentClick = (e: React.MouseEvent<HTMLDivElement>) => {
    const target = e.target as HTMLElement;
    const anchor = target.closest("a");
    if (anchor) {
      const href = anchor.getAttribute("href");
      if (href) {
        if (href.startsWith("#")) {
          e.preventDefault();
          scrollToFragment(href.slice(1));
          return;
        }
        if (href.startsWith("section:")) {
          e.preventDefault();
          const [sectionPath, fragment = ""] = href.split("#");
          const found = chapters.find(ch => ch.contentPath === sectionPath);
          if (found) {
            pendingFragmentRef.current = fragment || null;
            void loadChapter(found);
            return;
          }
        }
        if (href.includes("/api/v1/reader/") && href.includes("/chapter/")) {
          e.preventDefault();
          const parts = href.split("/chapter/");
          if (parts.length > 1) {
            const chId = parts[1].split("#")[0];
            const found = chapters.find(ch => ch.id === chId);
            if (found) {
              void loadChapter(found);
              return;
            }
          }
        }
        if (href.includes("/api/v1/reader/") && href.includes("/asset/")) {
          e.preventDefault();
          const parts = href.split("/asset/");
          if (parts.length > 1) {
            const resolvedPath = decodeURIComponent(parts[1].split("#")[0].split("?")[0]);
            const targetPath = resolvedPath.toLowerCase().replace(/^\/+/, "");
            const found = chapters.find(ch => {
              const chPath = ch.contentPath?.toLowerCase().replace(/^\/+/, "");
              return chPath === targetPath || (chPath && targetPath.endsWith(chPath)) || (chPath && chPath.endsWith(targetPath));
            });
            if (found) {
              void loadChapter(found);
              return;
            }
          }
        }
      }
    }
  };

  const scrollToFragment = (fragment: string) => {
    const normalized = fragment.trim();
    if (!normalized) return;
    const root = columnsRef.current;
    if (!root) return;
    const escaped = typeof CSS !== "undefined" && typeof CSS.escape === "function"
      ? CSS.escape(normalized)
      : normalized.replace(/["\\.#:[\]>+~()]/g, "\\$&");
    const target = root.querySelector<HTMLElement>(`#${escaped}`);
    if (!target) return;
    target.scrollIntoView({ behavior: "smooth", block: "start", inline: "start" });
  };

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
          // Sort chapters by index
          const sorted = [...res.value.data.chapters].sort((a, b) => a.chapterIndex - b.chapterIndex);
          setChapters(sorted);
          if (sorted.length > 0) {
            let targetChapter = sorted[0];
            let locationCfi: string | undefined = undefined;

            if (progressRes.status === "fulfilled" && progressRes.value.status && progressRes.value.data) {
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
    setHtmlContent(""); // clear while loading
    stop(); // stop TTS if playing
    try {
      const html = await readerService.getChapterHtml(bookId, chapter.id, fileId);
      setHtmlContent(html);
      
      // Scroll to top
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

  useEffect(() => {
    if (!user || !currentChapter || !bookId) return;
    const chapterPosition = chapters.findIndex((chapter) => chapter.id === currentChapter.id);
    const progressPercent = chapterPosition >= 0
      ? Math.round(((chapterPosition + 1) / chapters.length) * 100)
      : 0;

    void featureService.recordReadingActivity({
      bookId,
      fileId,
      chapterId: currentChapter.id,
      chapterTitle: currentChapter.title,
      chapterIndex: currentChapter.chapterIndex,
      progressPercent,
      eventType: "chapter_open",
    }).catch((error) => {
      console.debug("Failed to record reading activity", error);
    });
  }, [user, bookId, fileId, currentChapter?.id, chapters.length]);

  const scrollTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleScroll = () => {
    if (!user || effectiveReadingMode !== "scroll" || !contentRef.current || !currentChapter || !bookId) return;
    
    const scrollTop = contentRef.current.scrollTop;
    if (scrollTimeoutRef.current) {
      clearTimeout(scrollTimeoutRef.current);
    }
    
    scrollTimeoutRef.current = setTimeout(() => {
      const chapterPosition = chapters.findIndex((c) => c.id === currentChapter.id);
      const progressPercent = chapterPosition >= 0
        ? Math.round(((chapterPosition + 1) / chapters.length) * 100)
        : 0;

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
      }).catch(console.debug);
    }, 2000);
  };

  useEffect(() => {
    if (!user || effectiveReadingMode === "scroll" || !currentChapter || !bookId) return;

    const chapterPosition = chapters.findIndex((c) => c.id === currentChapter.id);
    const progressPercent = chapterPosition >= 0
      ? Math.round(((chapterPosition + 1) / chapters.length) * 100)
      : 0;

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
          title={currentChapter?.title || "Reading"}
          headerBg={headerBg}
          canGoPrev={canGoPrev}
          canGoNext={canGoNext}
          settingsOpen={settingsOpen}
          theme={theme}
          fontFamily={fontFamily}
          fontSize={fontSize}
          maxWidth={maxWidth}
          effectiveReadingMode={effectiveReadingMode}
          canUseDoubleMode={canUseDoubleMode}
          onPrev={handlePrev}
          onNext={handleNext}
          setSettingsOpen={setSettingsOpen}
          setTheme={setTheme}
          setFontFamily={setFontFamily}
          setFontSize={setFontSize}
          setMaxWidth={setMaxWidth}
          setReadingMode={setReadingMode}
          resetSettings={resetSettings}
          ttsSupported={isSupported}
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
          onOpenSearch={() => setSearchOpen(true)}
        />

        {/* Reader Scrollable Area */}
        <div 
          ref={contentRef}
          className={`flex-1 ${
            effectiveReadingMode === 'scroll' 
              ? 'overflow-y-auto pt-20 pb-24 px-4 sm:px-8' 
              : 'overflow-hidden flex flex-col pt-14 pb-6 px-4 sm:px-20'
          } relative`}
          onClick={() => setSettingsOpen(false)}
          onScroll={handleScroll}
        >
          <div 
            ref={pageFrameRef}
            className={`w-full mx-auto ${effectiveReadingMode === 'scroll' ? 'h-auto' : 'flex-1 min-h-0 flex flex-col'}`}
            style={{ 
              maxWidth: maxWidth >= 1600 ? '100%' : `${maxWidth}px`,
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
            {book?.files?.find(f => f.id === fileId)?.format.match(/^(mp3|m4a|m4b|flac)$/i) ? (
              <div className="flex-1 w-full h-full pb-32">
                <AudioPlayer 
                  rawUrl={`/api/v1/reader/${bookId}/file?file_id=${fileId}`}
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
            ) : htmlContent ? (
              <ReaderContent
                htmlContent={htmlContent}
                proseClass={proseClass}
                effectiveReadingMode={effectiveReadingMode}
                pageWidth={pageWidth}
                columnsRef={columnsRef}
                onContentClick={handleContentClick}
              />
            ) : (
              <div className="flex justify-center items-center h-64 opacity-50">
                {t('common.loading', 'Loading content...')}
              </div>
            )}
            
            {effectiveReadingMode === "scroll" && (
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

          {effectiveReadingMode !== "scroll" && htmlContent && (
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
      </div>
      
      <ReaderSidebar
        t={t}
        book={book}
        chapters={chapters}
        currentChapter={currentChapter}
        sidebarBg={sidebarBg}
        sidebarRef={sidebarRef}
        onClose={() => setSidebarOpen(false)}
        onBack={handleReaderBack}
        onSelectChapter={(chapter) => {
          void loadChapter(chapter);
          setSidebarOpen(false);
        }}
      />

      {selectionRange && (
        <ReaderSelectionToolbar
          t={t}
          toolbarPos={toolbarPos}
          isSupported={isSupported}
          onReadSelection={handleReadSelection}
          onReadFromHere={handleReadFromHere}
          onCopyText={handleCopyText}
          onHighlight={handleHighlight}
        />
      )}

      {searchOpen && bookId && (
        <div className="fixed top-16 right-6 z-50 animate-fade-in">
          <ReaderInBookSearch
            bookId={bookId}
            onClose={() => setSearchOpen(false)}
            onSelectResult={(chapterId) => {
              const ch = chapters.find((c) => c.id === chapterId);
              if (ch) {
                void loadChapter(ch);
              }
              setSearchOpen(false);
            }}
          />
        </div>
      )}
    </div>
  );
};
