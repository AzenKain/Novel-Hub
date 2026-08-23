import React, { useMemo, useState, useEffect, useRef, useCallback } from "react";
import { useTranslation } from "react-i18next";
import {
  BookmarkPlus,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";
import { getSideTapRatio } from "@/constants";
import { sanitizeReaderHtml } from "@/utils/readerHtml";
import { useReaderStore } from "@/stores/readerStore";

interface ComicReaderProps {
  htmlContent: string;
  onContentClick?: (event: React.MouseEvent<HTMLDivElement>) => void;
  onImageClick?: (target: { image_url: string; page_index: number; x: number; y: number }) => void;
  onPrevChapter?: () => void;
  onNextChapter?: () => void;
  canGoPrevChapter?: boolean;
  canGoNextChapter?: boolean;
  initialLanding?: string | null;
  currentPage?: number;
  onPageChange?: (page: number) => void;
  onTotalPagesChange?: (total: number) => void;
}

export const ComicReader: React.FC<ComicReaderProps> = React.memo(({
  htmlContent,
  onContentClick,
  onImageClick,
  onPrevChapter,
  onNextChapter,
  canGoPrevChapter = false,
  canGoNextChapter = false,
  initialLanding = null,
  currentPage: propPage,
  onPageChange,
  onTotalPagesChange,
}) => {
  const { t } = useTranslation();
  const {
    readingMode,
    comicInvertColors,
    pageFit,
    readingDirection,
    maxWidth,
  } = useReaderStore();

  const isWebtoon = readingMode === "webtoon" || readingMode === "scroll";
  const isDouble = readingMode === "double";
  const isRtl = readingDirection === "rtl";
  const [internalPage, setInternalPage] = useState(0);
  const currentPage = propPage !== undefined ? propPage : internalPage;
  const leftPageIdx = isDouble && isRtl ? currentPage + 1 : currentPage;
  const rightPageIdx = isDouble && isRtl ? currentPage : currentPage + 1;

  const [windowWidth, setWindowWidth] = useState<number>(() =>
    typeof window !== "undefined" ? window.innerWidth : 1024
  );

  useEffect(() => {
    const handleResize = () => {
      setWindowWidth(window.innerWidth);
    };
    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  const sideTapRatio = getSideTapRatio(windowWidth);
  const sideWidthPercent = Math.round(sideTapRatio * 100);

  const [failedImages, setFailedImages] = useState<Set<string>>(new Set());

  const handleImageError = useCallback((src: string) => {
    if (!src) return;
    setFailedImages((prev) => {
      if (prev.has(src)) return prev;
      const next = new Set(prev);
      next.add(src);
      return next;
    });
  }, []);

  // Extract image URLs from the HTML content
  const allImageUrls = useMemo(() => {
    if (!htmlContent) return [];
    const div = document.createElement("div");
    div.innerHTML = htmlContent;
    const imgs = Array.from(div.querySelectorAll("img"));
    return imgs.map((img) => img.getAttribute("src") || "").filter(Boolean);
  }, [htmlContent]);

  const imageUrls = useMemo(() => {
    return allImageUrls.filter((url) => !failedImages.has(url));
  }, [allImageUrls, failedImages]);

  useEffect(() => {
    onTotalPagesChange?.(imageUrls.length);
  }, [imageUrls.length, onTotalPagesChange]);

  // Set page when content changes or initialLanding specifies landing at end
  useEffect(() => {
    if (initialLanding === "end" && !isWebtoon && imageUrls.length > 0) {
      const step = isDouble ? 2 : 1;
      const lastIndex = Math.max(0, imageUrls.length - step);
      setInternalPage(lastIndex);
      onPageChange?.(lastIndex);
    } else {
      setInternalPage(0);
      onPageChange?.(0);
    }
  }, [htmlContent, initialLanding, isWebtoon, isDouble]); // eslint-disable-line react-hooks/exhaustive-deps

  const step = isDouble ? 2 : 1;
  const isLastPage = currentPage >= imageUrls.length - step;

  const handleGoNext = useCallback(() => {
    if (currentPage >= imageUrls.length - step) {
      if (canGoNextChapter && onNextChapter) onNextChapter();
    } else {
      const next = Math.min(imageUrls.length - 1, currentPage + step);
      setInternalPage(next);
      onPageChange?.(next);
    }
  }, [currentPage, imageUrls.length, step, canGoNextChapter, onNextChapter, onPageChange]);

  const handleGoPrev = useCallback(() => {
    if (currentPage <= 0) {
      if (canGoPrevChapter && onPrevChapter) onPrevChapter();
    } else {
      const prev = Math.max(0, currentPage - step);
      setInternalPage(prev);
      onPageChange?.(prev);
    }
  }, [currentPage, step, canGoPrevChapter, onPrevChapter, onPageChange]);

  // Keyboard navigation for paged modes
  useEffect(() => {
    if (isWebtoon || imageUrls.length === 0) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "ArrowRight" || e.key === "PageDown" || e.key === " ") {
        e.preventDefault();
        if (isRtl) {
          handleGoPrev();
        } else {
          handleGoNext();
        }
      } else if (e.key === "ArrowLeft" || e.key === "PageUp") {
        e.preventDefault();
        if (isRtl) {
          handleGoNext();
        } else {
          handleGoPrev();
        }
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isWebtoon, imageUrls.length, isRtl, handleGoNext, handleGoPrev]);

  const sanitizedHTML = useMemo(() => sanitizeReaderHtml(htmlContent), [htmlContent]);
  const webtoonContainerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isWebtoon || !webtoonContainerRef.current) return;
    const imgs = webtoonContainerRef.current.querySelectorAll("img");
    imgs.forEach((img) => { img.loading = "lazy"; });
  }, [isWebtoon, sanitizedHTML]);
  const isJumpingRef = useRef(false);
  const lastJumpedPageRef = useRef<number | null>(null);

  // Smooth scroll to target page when jumping in webtoon mode
  useEffect(() => {
    if (!isWebtoon || !webtoonContainerRef.current) return;
    if (lastJumpedPageRef.current === currentPage) return;
    const imgs = Array.from(webtoonContainerRef.current.querySelectorAll("img"));
    if (imgs && imgs[currentPage]) {
      isJumpingRef.current = true;
      lastJumpedPageRef.current = currentPage;
      imgs[currentPage].scrollIntoView({ behavior: "smooth", block: "start" });
      setTimeout(() => {
        isJumpingRef.current = false;
      }, 500);
    }
  }, [currentPage, isWebtoon]);

  // Track scroll position in webtoon mode to update current page
  useEffect(() => {
    if (!isWebtoon || !webtoonContainerRef.current) return;
    const container = webtoonContainerRef.current;

    let ticking = false;
    const handleScroll = () => {
      if (isJumpingRef.current || ticking) return;
      ticking = true;
      requestAnimationFrame(() => {
        const imgs = Array.from(container.querySelectorAll("img"));
        if (imgs.length === 0) {
          ticking = false;
          return;
        }

        const containerTop = container.getBoundingClientRect().top;
        let closestIndex = 0;
        let minDiff = Infinity;

        imgs.forEach((img, idx) => {
          const rect = img.getBoundingClientRect();
          const diff = Math.abs(rect.top - containerTop);
          if (diff < minDiff) {
            minDiff = diff;
            closestIndex = idx;
          }
        });

        if (closestIndex !== currentPage) {
          lastJumpedPageRef.current = closestIndex;
          setInternalPage(closestIndex);
          onPageChange?.(closestIndex);
        }
        ticking = false;
      });
    };

    container.addEventListener("scroll", handleScroll, { passive: true });
    return () => container.removeEventListener("scroll", handleScroll);
  }, [isWebtoon, currentPage, onPageChange]);

  // Touch Long-Press and Double-Click handlers for mobile / tablet bookmarking
  const longPressTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const touchStartPosRef = useRef<{ x: number; y: number } | null>(null);

  const handleTouchStart = (e: React.TouchEvent) => {
    if (e.touches.length !== 1) return;
    const touch = e.touches[0];
    touchStartPosRef.current = { x: touch.clientX, y: touch.clientY };
    if (longPressTimerRef.current) clearTimeout(longPressTimerRef.current);
    longPressTimerRef.current = setTimeout(() => {
      if (imageUrls[currentPage]) {
        onImageClick?.({
          image_url: imageUrls[currentPage],
          page_index: currentPage,
          x: touch.clientX,
          y: touch.clientY,
        });
      }
    }, 450);
  };

  const handleTouchMove = (e: React.TouchEvent) => {
    if (!touchStartPosRef.current || e.touches.length !== 1) return;
    const touch = e.touches[0];
    const dx = Math.abs(touch.clientX - touchStartPosRef.current.x);
    const dy = Math.abs(touch.clientY - touchStartPosRef.current.y);
    if (dx > 12 || dy > 12) {
      if (longPressTimerRef.current) clearTimeout(longPressTimerRef.current);
    }
  };

  const handleTouchEnd = () => {
    if (longPressTimerRef.current) clearTimeout(longPressTimerRef.current);
  };

  const handleDoubleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (imageUrls[currentPage]) {
      onImageClick?.({
        image_url: imageUrls[currentPage],
        page_index: currentPage,
        x: e.clientX,
        y: e.clientY,
      });
    }
  };

  const filterStyle = comicInvertColors
    ? { filter: "invert(1) hue-rotate(180deg)" }
    : undefined;

  return (
    <div className="relative w-full h-full flex flex-col items-center justify-center overflow-hidden">
      {/* Mode 1: Webtoon Continuous Vertical Scroll */}
      {isWebtoon && (
        <div
          ref={webtoonContainerRef}
          className="w-full h-full overflow-y-auto overflow-x-hidden flex flex-col items-center p-0 select-none comic-webtoon-container"
          style={filterStyle}
          onClick={(e) => {
            const target = e.target as HTMLElement;
            if (target && target.tagName.toLowerCase() === "img") {
              const img = target as HTMLImageElement;
              const src = img.currentSrc || img.src;
              if (src) {
                const idx = imageUrls.indexOf(src);
                onImageClick?.({
                  image_url: src,
                  page_index: idx >= 0 ? idx : currentPage,
                  x: e.clientX,
                  y: e.clientY,
                });
                return;
              }
            }
            onContentClick?.(e);
          }}
          onErrorCapture={(e) => {
            const target = e.target as HTMLElement;
            if (target && target.tagName.toLowerCase() === "img") {
              target.style.display = "none";
            }
          }}
        >
          <div
            className="w-full flex flex-col items-center mx-auto"
            style={{
              maxWidth: maxWidth >= 1600 ? "100%" : `${maxWidth}px`,
            }}
            dangerouslySetInnerHTML={{ __html: sanitizedHTML }}
          />
        </div>
      )}

      {/* Mode 2 & 3: Single Page or Double Page Spread */}
      {!isWebtoon && (
        <div
          className={`relative w-full flex-1 min-h-0 flex flex-col items-center justify-center p-0 select-none ${
            pageFit === "height" ? "overflow-hidden h-full" : "overflow-y-auto"
          }`}
          onClick={onContentClick}
          onDoubleClick={handleDoubleClick}
          onTouchStart={handleTouchStart}
          onTouchMove={handleTouchMove}
          onTouchEnd={handleTouchEnd}
          style={filterStyle}
        >
          {/* Navigation Click Zones (Responsive sideWidthPercent: 0% on desktop >= 1024px, scaling up to 30% on mobile) */}
          {sideWidthPercent > 0 && (
            <div
              onClick={(e) => {
                e.stopPropagation();
                const targetPageIdx = isDouble ? leftPageIdx : currentPage;
                if (imageUrls[targetPageIdx]) {
                  onImageClick?.({
                    image_url: imageUrls[targetPageIdx],
                    page_index: targetPageIdx,
                    x: e.clientX,
                    y: e.clientY,
                  });
                }
                if (isRtl) {
                  handleGoNext();
                } else {
                  handleGoPrev();
                }
              }}
              style={{ width: `${sideWidthPercent}%` }}
              className="absolute left-0 inset-y-0 z-30 cursor-w-resize flex items-center justify-start pl-6 group pointer-events-auto"
              title={
                isRtl
                  ? isLastPage && canGoNextChapter ? t("reader.next_chapter", "Next Chapter") : t("reader.next_page", "Next Page")
                  : currentPage <= 0 && canGoPrevChapter ? t("reader.prev_chapter", "Previous Chapter") : t("reader.prev_page", "Previous Page")
              }
            >
              <div className="p-3.5 rounded-full bg-black/60 text-white opacity-0 group-hover:opacity-100 transition-opacity backdrop-blur-xs shadow-lg">
                {isRtl ? <ChevronRight className="w-6 h-6" /> : <ChevronLeft className="w-6 h-6" />}
              </div>
            </div>
          )}

          {/* Center Click Zone: Toggles UI / Header / Bars */}
          <div
            onClick={(e) => {
              e.stopPropagation();
              const isRightSide = e.clientX >= window.innerWidth / 2;
              const targetPageIdx = isDouble && isRightSide && imageUrls[rightPageIdx] ? rightPageIdx : leftPageIdx;
              if (imageUrls[targetPageIdx]) {
                onImageClick?.({
                  image_url: imageUrls[targetPageIdx],
                  page_index: targetPageIdx,
                  x: e.clientX,
                  y: e.clientY,
                });
              }
              onContentClick?.(e);
            }}
            onContextMenu={(e) => {
              e.preventDefault();
              e.stopPropagation();
              const isRightSide = e.clientX >= window.innerWidth / 2;
              const targetPageIdx = isDouble && isRightSide && imageUrls[rightPageIdx] ? rightPageIdx : leftPageIdx;
              if (imageUrls[targetPageIdx]) {
                onImageClick?.({
                  image_url: imageUrls[targetPageIdx],
                  page_index: targetPageIdx,
                  x: e.clientX,
                  y: e.clientY,
                });
              }
            }}
            style={{
              left: `${sideWidthPercent}%`,
              width: `${100 - 2 * sideWidthPercent}%`,
            }}
            className="absolute inset-y-0 z-30 cursor-pointer pointer-events-auto"
          />

          {sideWidthPercent > 0 && (
            <div
              onClick={(e) => {
                e.stopPropagation();
                const targetPageIdx = isDouble && imageUrls[rightPageIdx] ? rightPageIdx : currentPage;
                if (imageUrls[targetPageIdx]) {
                  onImageClick?.({
                    image_url: imageUrls[targetPageIdx],
                    page_index: targetPageIdx,
                    x: e.clientX,
                    y: e.clientY,
                  });
                }
                if (isRtl) {
                  handleGoPrev();
                } else {
                  handleGoNext();
                }
              }}
              style={{ width: `${sideWidthPercent}%` }}
              className="absolute right-0 inset-y-0 z-30 cursor-e-resize flex items-center justify-end pr-6 group pointer-events-auto"
              title={
                isRtl
                  ? currentPage <= 0 && canGoPrevChapter ? t("reader.prev_chapter", "Previous Chapter") : t("reader.next_page", "Next Page")
                  : isLastPage && canGoNextChapter ? t("reader.next_chapter", "Next Chapter") : t("reader.next_page", "Next Page")
              }
            >
              <div className="p-3.5 rounded-full bg-black/60 text-white opacity-0 group-hover:opacity-100 transition-opacity backdrop-blur-xs shadow-lg">
                {isRtl ? <ChevronLeft className="w-6 h-6" /> : <ChevronRight className="w-6 h-6" />}
              </div>
            </div>
          )}

          {/* Rendered Images */}
          <div
            className={`flex items-center justify-center gap-2 w-full mx-auto relative z-10 ${pageFit === "height" ? "h-full min-h-0" : ""}`}
            style={{
              maxWidth: maxWidth >= 1600 ? "100%" : `${maxWidth}px`,
            }}
          >
            {/* Left page (in double mode) or single page */}
            {imageUrls[leftPageIdx] && (
              <img
                src={imageUrls[leftPageIdx]}
                alt=""
                onClick={(e) => {
                  e.stopPropagation();
                  onImageClick?.({
                    image_url: imageUrls[leftPageIdx],
                    page_index: leftPageIdx,
                    x: e.clientX,
                    y: e.clientY,
                  });
                }}
                onError={() => handleImageError(imageUrls[leftPageIdx])}
                className={`rounded-lg object-contain shadow-md transition-all duration-150 ${
                  pageFit === "height"
                    ? "max-h-full h-full w-full max-w-full"
                    : pageFit === "width"
                    ? "w-full h-auto"
                    : "max-w-none"
                }`}
                style={{
                  maxWidth: isDouble ? "50%" : "100%",
                }}
              />
            )}

            {/* Right page in double mode */}
            {isDouble && imageUrls[rightPageIdx] && (
              <img
                src={imageUrls[rightPageIdx]}
                alt=""
                onClick={(e) => {
                  e.stopPropagation();
                  onImageClick?.({
                    image_url: imageUrls[rightPageIdx],
                    page_index: rightPageIdx,
                    x: e.clientX,
                    y: e.clientY,
                  });
                }}
                onError={() => handleImageError(imageUrls[rightPageIdx])}
                className={`rounded-lg object-contain shadow-md hidden sm:block transition-all duration-150 ${
                  pageFit === "height"
                    ? "max-h-full h-full w-full max-w-full"
                    : pageFit === "width"
                    ? "w-full h-auto"
                    : "max-w-none"
                }`}
                style={{
                  maxWidth: "50%",
                }}
              />
            )}
          </div>
        </div>
      )}
    </div>
  );
});

ComicReader.displayName = "ComicReader";
