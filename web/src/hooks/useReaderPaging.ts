import { useEffect, useLayoutEffect, useRef, type RefObject } from "react";

import { READER_PAGE_GAP } from "@/constants";
import type { PageAnimation } from "@/stores";

type UseReaderPagingArgs = {
  contentRef: RefObject<HTMLDivElement | null>;
  columnsRef: RefObject<HTMLDivElement | null>;
  pageFrameRef: RefObject<HTMLDivElement | null>;
  pendingLandingRef?: RefObject<string | null>;
  htmlContent: string;
  maxWidth: number;
  scrollLayout: boolean;
  effectiveReadingMode: string;
  rtlPaging: boolean;
  pageAnimation?: PageAnimation;
  pageIndex: number;
  setPageIndex: (index: number) => void;
  setPageFrameWidth: (width: number) => void;
  onChapterNext: () => void;
  onChapterPrev: () => void;
};

/**
 * Owns everything about *where in the chapter the reader is*: the page frame
 * width, the current page index, scrolling to a page, and keeping the index in
 * sync when the user scrolls or switches between scroll/single/double modes.
 *
 * Extracted verbatim from ReaderWorkspace — behaviour is unchanged.
 */
export function useReaderPaging({
  contentRef,
  columnsRef,
  pageFrameRef,
  pendingLandingRef,
  htmlContent,
  maxWidth,
  scrollLayout,
  effectiveReadingMode,
  rtlPaging,
  pageAnimation = "eink",
  pageIndex,
  setPageIndex,
  setPageFrameWidth,
  onChapterNext,
  onChapterPrev,
}: UseReaderPagingArgs) {
  const prevModeRef = useRef<string>(effectiveReadingMode);
  const lastPageIndexRef = useRef<number>(pageIndex);
  const isResizingRef = useRef<boolean>(false);
  const cachedNodesRef = useRef<HTMLElement[] | null>(null);
  const cachedNodesContentRef = useRef<string>("");

  useLayoutEffect(() => {
    if (!htmlContent || htmlContent.trim() === "") return;

    const container = columnsRef.current || contentRef.current;

    const isPagedMode = effectiveReadingMode === "single" || effectiveReadingMode === "double";
    if (pendingLandingRef?.current === "end" && isPagedMode) {
      pendingLandingRef.current = null;

      const landAtEnd = () => {
        const metrics = getPagedScrollMetrics();
        if (metrics) {
          lastPageIndexRef.current = metrics.maxIndex;
          scrollToPageIndex(metrics.maxIndex, true);
        }
        if (container) {
          container.style.visibility = "visible";
        }
        if (columnsRef.current) {
          columnsRef.current.style.visibility = "visible";
        }
        if (contentRef.current) {
          contentRef.current.style.visibility = "visible";
        }
      };

      // Perform landing immediately
      landAtEnd();

      // Listen for image loads to stay anchored to the last page as image assets load
      const images = (columnsRef.current || contentRef.current)?.querySelectorAll("img");
      if (images && images.length > 0) {
        images.forEach((img) => {
          if (!img.complete) {
            img.addEventListener("load", landAtEnd, { once: true });
            img.addEventListener("error", landAtEnd, { once: true });
          }
        });
      }

      requestAnimationFrame(() => {
        landAtEnd();
      });
      return;
    }

    if (pendingLandingRef) {
      pendingLandingRef.current = null;
    }

    setPageIndex(0);
    lastPageIndexRef.current = 0;
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
    if (container) {
      container.style.visibility = "visible";
    }
  }, [htmlContent]);

  useEffect(() => {
    if (scrollLayout) {
      setPageFrameWidth(0);
      return;
    }

    const frame = pageFrameRef.current;
    if (!frame) return;

    const updatePageFrameWidth = () => {
      isResizingRef.current = true;
      cachedNodesContentRef.current = "";
      const targetPage = lastPageIndexRef.current;
      setPageFrameWidth(frame.clientWidth);

      if (frame.clientHeight > 0) {
        frame.style.setProperty("--reader-page-height", `${frame.clientHeight}px`);
      }

      const container = getPagedScrollContainer();
      if (container) {
        if (frame.clientHeight > 0) {
          container.style.setProperty("--reader-page-height", `${frame.clientHeight}px`);
        }
        const newStep = frame.clientWidth + READER_PAGE_GAP;
        container.scrollLeft = targetPage * newStep * (rtlPaging ? -1 : 1);
      }

      requestAnimationFrame(() => {
        scrollToPageIndex(targetPage, true);
        setTimeout(() => {
          isResizingRef.current = false;
        }, 60);
      });
    };

    updatePageFrameWidth();

    const resizeObserver = new ResizeObserver(updatePageFrameWidth);
    resizeObserver.observe(frame);
    window.addEventListener("resize", updatePageFrameWidth);

    return () => {
      resizeObserver.disconnect();
      window.removeEventListener("resize", updatePageFrameWidth);
    };
  }, [scrollLayout, maxWidth, effectiveReadingMode, rtlPaging]);

  useEffect(() => {
    lastPageIndexRef.current = pageIndex;
  }, [pageIndex]);

  const getPagedScrollContainer = () => {
    const readerContent = columnsRef.current;
    if (!readerContent) return null;
    return readerContent.querySelector("body") || readerContent;
  };

  const getPagedScrollMetrics = () => {
    const container = getPagedScrollContainer();
    if (!container) return;

    const scrollStep = container.clientWidth + READER_PAGE_GAP;
    let scrollWidth = container.scrollWidth;

    if (cachedNodesContentRef.current !== htmlContent) {
      cachedNodesRef.current = Array.from(container.querySelectorAll<HTMLElement>("p, div, figure, h1, h2, h3, h4, h5, h6, img"));
      cachedNodesContentRef.current = htmlContent;
    }
    const allChildren = cachedNodesRef.current || [];
    if (allChildren.length > 0) {
      let maxChildRight = 0;
      for (let i = allChildren.length - 1; i >= Math.max(0, allChildren.length - 30); i--) {
        const el = allChildren[i];
        const right = el.offsetLeft + el.offsetWidth;
        if (right > maxChildRight) {
          maxChildRight = right;
        }
      }
      if (maxChildRight > scrollWidth) {
        scrollWidth = maxChildRight;
      }
    }

    const maxScroll = Math.max(0, scrollWidth - container.clientWidth);
    const maxIndex = Math.max(0, Math.round(maxScroll / scrollStep));
    return { container, scrollStep, maxIndex };
  };

  const triggerPageAnimation = () => {
    const el = columnsRef.current;
    if (!el) return;
    if (pageAnimation === "eink") {
      el.classList.remove("reader-anim-eink", "reader-anim-fade");
      void el.offsetWidth;
      el.classList.add("reader-anim-eink");
    } else if (pageAnimation === "fade") {
      el.classList.remove("reader-anim-eink", "reader-anim-fade");
      void el.offsetWidth;
      el.classList.add("reader-anim-fade");
    }
  };

  const scrollToPageIndex = (targetIndex: number, instant = false) => {
    const metrics = getPagedScrollMetrics();
    if (!metrics) return;

    const { container, scrollStep, maxIndex } = metrics;
    const nextIndex = Math.min(Math.max(targetIndex, 0), maxIndex);

    const isSlide = !instant && pageAnimation === "slide";
    const behavior = isSlide ? "smooth" : "auto";

    container.scrollTo({
      left: nextIndex * scrollStep * (rtlPaging ? -1 : 1),
      behavior,
    });

    if (!instant && nextIndex !== pageIndex) {
      triggerPageAnimation();
    }

    setPageIndex(nextIndex);
  };

  // Fractional location within the current chapter (0–1) for true progress.
  const getLocationFraction = (): number => {
    if (scrollLayout && contentRef.current) {
      const el = contentRef.current;
      const max = el.scrollHeight - el.clientHeight;
      return max > 0 ? Math.min(1, Math.max(0, el.scrollTop / max)) : 0;
    }
    const metrics = getPagedScrollMetrics();
    if (!metrics || metrics.maxIndex <= 0) return 0;
    return Math.min(1, Math.max(0, pageIndex / metrics.maxIndex));
  };

  useEffect(() => {
    const prevMode = prevModeRef.current;
    if (prevMode === effectiveReadingMode) return;

    const isPrevPaged = prevMode === "single" || prevMode === "double";
    const isNewPaged = effectiveReadingMode === "single" || effectiveReadingMode === "double";

    if (isPrevPaged && isNewPaged) {
      const prevIdx = lastPageIndexRef.current;
      let targetPage = prevIdx;

      if (prevMode === "double" && effectiveReadingMode === "single") {
        targetPage = prevIdx * 2;
      } else if (prevMode === "single" && effectiveReadingMode === "double") {
        targetPage = Math.floor(prevIdx / 2);
      }

      prevModeRef.current = effectiveReadingMode;

      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          scrollToPageIndex(targetPage, true);
        });
      });
      return;
    }

    if (isPrevPaged && !isNewPaged) {
      const fraction = getLocationFraction();
      prevModeRef.current = effectiveReadingMode;
      requestAnimationFrame(() => {
        if (contentRef.current) {
          const el = contentRef.current;
          const maxScroll = el.scrollHeight - el.clientHeight;
          el.scrollTop = Math.round(fraction * maxScroll);
        }
      });
      return;
    }

    if (!isPrevPaged && isNewPaged) {
      const fraction = getLocationFraction();
      prevModeRef.current = effectiveReadingMode;
      requestAnimationFrame(() => {
        const metrics = getPagedScrollMetrics();
        if (metrics) {
          const targetIndex = Math.round(fraction * metrics.maxIndex);
          scrollToPageIndex(targetIndex, true);
        }
      });
      return;
    }

    prevModeRef.current = effectiveReadingMode;
  }, [effectiveReadingMode]);

  // Sync pageIndex from manual horizontal scroll in paged modes.
  useEffect(() => {
    if (scrollLayout) return;
    const container = getPagedScrollContainer();
    if (!container) return;
    const onScroll = () => {
      if (isResizingRef.current) return;
      const metrics = getPagedScrollMetrics();
      if (!metrics) return;
      const idx = Math.round(Math.abs(container.scrollLeft) / metrics.scrollStep);
      if (idx !== pageIndex) setPageIndex(idx);
    };
    container.addEventListener("scroll", onScroll, { passive: true });
    return () => container.removeEventListener("scroll", onScroll);
  }, [scrollLayout, htmlContent, pageIndex]);

  const handlePageNext = () => {
    const metrics = getPagedScrollMetrics();
    if (metrics && pageIndex >= metrics.maxIndex) {
      onChapterNext();
      return;
    }
    scrollToPageIndex(pageIndex + 1);
  };

  const handlePagePrev = () => {
    if (pageIndex <= 0) {
      onChapterPrev();
      return;
    }
    scrollToPageIndex(pageIndex - 1);
  };

  return {
    getPagedScrollMetrics,
    scrollToPageIndex,
    getLocationFraction,
    handlePageNext,
    handlePagePrev,
  };
}
