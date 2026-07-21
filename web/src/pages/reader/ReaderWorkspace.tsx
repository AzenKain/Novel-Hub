import { ReaderContent, ReaderPageControls, ReaderSidebar, ReaderTopBar } from "@/components/reader";
import { getReaderThemeClasses } from "@/config/readerTheme";
import { featureService, readerService } from "@/services";
import { useAuthStore, useReaderStore } from "@/stores";
import type { Chapter } from "@/types";
import React, { useEffect, useState, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useTTS } from "@/hooks/useTTS";
import { FastAverageColor } from "fast-average-color";
import { useHighlights } from "@/hooks/useHighlights";
import { useAutoScroll } from "@/hooks/useAutoScroll";
import { useReadingStats } from "@/hooks/useReadingStats";


const clearHighlight = () => {
  const selection = window.getSelection();
  if (selection) {
    selection.removeAllRanges();
  }
};

const highlightTextRange = (container: HTMLElement, startChar: number, length: number) => {
  const treeWalker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT, null);
  let currentOffset = 0;
  let startNode: Node | null = null;
  let startNodeOffset = 0;
  let endNode: Node | null = null;
  let endNodeOffset = 0;

  while (treeWalker.nextNode()) {
    const node = treeWalker.currentNode;
    const nodeLength = node.textContent?.length || 0;
    
    if (!startNode && currentOffset + nodeLength > startChar) {
      startNode = node;
      startNodeOffset = startChar - currentOffset;
    }
    
    if (startNode && !endNode && currentOffset + nodeLength >= startChar + length) {
      endNode = node;
      endNodeOffset = (startChar + length) - currentOffset;
      break; 
    }
    
    currentOffset += nodeLength;
  }
  
  if (startNode && endNode) {
    const range = document.createRange();
    range.setStart(startNode, startNodeOffset);
    range.setEnd(endNode, endNodeOffset);
    
    const selection = window.getSelection();
    if (selection) {
      selection.removeAllRanges();
      selection.addRange(range);
    }
  }
};

import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { useShallow } from "zustand/react/shallow";

import { MIN_DOUBLE_PAGE_WIDTH, READER_CONTENT_MEASURE, READER_PAGE_GAP } from "@/constants";

export const ReaderWorkspace = () => {
  const { bookId } = useParams<{ bookId: string }>();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const fileId = searchParams.get("file_id") || undefined;
  
  const { t } = useTranslation();


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
    setMaxWidth,
    setReadingMode,
    setPageIndex,
    setPageFrameWidth,
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
    setMaxWidth: state.setMaxWidth,
    setReadingMode: state.setReadingMode,
    setPageIndex: state.setPageIndex,
    setPageFrameWidth: state.setPageFrameWidth,
    resetSettings: state.resetSettings,
    reset: state.reset,
  })));

  // New features hooks
  const { highlights, addHighlight, removeHighlight } = useHighlights(book?.id || '', currentChapter?.id);
  useReadingStats(book?.id, !settingsOpen); // Active when settings are closed

  useEffect(() => {
    return () => {
      reset();
    };
  }, [reset]);

    const { isSupported, isPlaying, isPaused, speak, pause, resume, stop } = useTTS({
    onEnd: () => clearHighlight(),
    onBoundary: (e) => {
      if (columnsRef.current && e.name === 'word') {
        highlightTextRange(columnsRef.current, ttsOffsetRef.current + e.charIndex, e.charLength);
      }
    }
  });

  // Clear highlight on unmount or when stopping TTS manually
  useEffect(() => {
    if (!isPlaying && !isPaused) {
      clearHighlight();
    }
  }, [isPlaying, isPaused]);


  const extractTextFromHtml = (html: string) => {
    if (typeof document === 'undefined') return '';
    const temp = document.createElement('div');
    temp.innerHTML = html;
    return temp.textContent || temp.innerText || '';
  };

  
  const getSelectionStartOffset = (container: HTMLElement) => {
    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0) return 0;
    
    const range = selection.getRangeAt(0);
    if (!container.contains(range.startContainer)) return 0;

    const preSelectionRange = range.cloneRange();
    preSelectionRange.selectNodeContents(container);
    preSelectionRange.setEnd(range.startContainer, range.startOffset);
    
    return preSelectionRange.toString().length;
  };

  
  // Floating toolbar state
  const [selectionRange, setSelectionRange] = useState<Range | null>(null);
  const [toolbarPos, setToolbarPos] = useState({ top: 0, left: 0 });

  useEffect(() => {
    const handleSelection = () => {
      const selection = window.getSelection();
      if (selection && selection.rangeCount > 0 && !selection.isCollapsed) {
        const range = selection.getRangeAt(0);
        // Only show toolbar if selecting inside reader content
        if (contentRef.current?.contains(range.commonAncestorContainer)) {
          setSelectionRange(range);
          const rect = range.getBoundingClientRect();
          setToolbarPos({ top: rect.top - 40, left: rect.left + rect.width / 2 });
          return;
        }
      }
      setSelectionRange(null);
    };
    document.addEventListener('selectionchange', handleSelection);
    return () => document.removeEventListener('selectionchange', handleSelection);
  }, []);

  useEffect(() => {
    if (!CSS.highlights) return;
    const highlightRanges = highlights.map((h: any) => {
      // Find the text nodes and create ranges for CSS Highlights.
      // This requires walking the DOM, for simplicity in this MVP we just clear them.
      // In a real app we'd construct accurate ranges based on startIndex and endIndex.
      return new Range(); 
    });
    // const highlightObj = new Highlight(...highlightRanges);
    // CSS.highlights.set('reader-highlights', highlightObj);
  }, [highlights]);

  const handleHighlight = async (color: string) => {
    if (selectionRange) {
      const text = selectionRange.toString();
      // Dummy start/end index for MVP
      await addHighlight(text, 0, text.length, color);
      window.getSelection()?.removeAllRanges();
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
        const res = await readerService.getBootstrap(bookId, fileId);
        if (res.status && res.data) {
          setBook(res.data.book);
          // Sort chapters by index
          const sorted = [...res.data.chapters].sort((a, b) => a.chapterIndex - b.chapterIndex);
          setChapters(sorted);
          if (sorted.length > 0) {
            loadChapter(sorted[0]);
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
      scrollToFragment(fragment);
    });
  }, [htmlContent]);

  useEffect(() => {
    if (!user || !bookId || !currentChapter || chapters.length === 0) return;
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
          autoScrollActive={autoScrollActive}
          onToggleAutoScroll={onToggleAutoScroll}

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
            {htmlContent ? (
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
    </div>
  );
};
