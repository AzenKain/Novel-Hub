import { useEffect, useLayoutEffect, useRef, type RefObject } from "react";

import { READER_PAGE_GAP } from "@/constants";
import { createRangeFromCharOffset } from "@/lib/readerHighlight";
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
  const lastScrollFractionRef = useRef<number>(0);
  const lastKnownFractionRef = useRef<number>(0);
  const modeTransitionFractionRef = useRef<number | null>(null);
  const lastVisibleElementRef = useRef<HTMLElement | null>(null);
  const lastVisibleTextOffsetRef = useRef<number>(0);
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
    lastScrollFractionRef.current = 0;
    lastVisibleElementRef.current = null;
    lastVisibleTextOffsetRef.current = 0;
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

  const getScrollVisibleElement = () => {
    const el = contentRef.current;
    if (!el) return lastVisibleElementRef.current;
    const container = columnsRef.current || el;
    const containerRect = el.getBoundingClientRect();
    const candidates = container.querySelectorAll<HTMLElement>("p, h1, h2, h3, h4, h5, h6, li, figure, img, blockquote");
    for (let i = 0; i < candidates.length; i++) {
      const candidate = candidates[i];
      const rect = candidate.getBoundingClientRect();
      if (rect.height > 0 || rect.bottom > 0) {
        if (rect.bottom > containerRect.top + 15) {
          return candidate;
        }
      } else {
        let top = candidate.offsetTop;
        let parent = candidate.offsetParent as HTMLElement | null;
        while (parent && el.contains(parent) && parent !== el) {
          top += parent.offsetTop;
          parent = parent.offsetParent as HTMLElement | null;
        }
        if (top + (candidate.offsetHeight || 30) >= el.scrollTop) {
          return candidate;
        }
      }
    }
    return lastVisibleElementRef.current;
  };

  // In scroll mode, track scroll fraction with O(1) arithmetic and throttle DOM element lookups via rAF
  useEffect(() => {
    if (!scrollLayout) return;
    const el = contentRef.current;
    if (!el) return;

    const onScroll = () => {
      const max = el.scrollHeight - el.clientHeight;
      if (max > 0) {
        lastScrollFractionRef.current = Math.min(1, Math.max(0, el.scrollTop / max));
      } else {
        lastScrollFractionRef.current = 0;
      }
      lastKnownFractionRef.current = lastScrollFractionRef.current;
      lastVisibleElementRef.current = getScrollVisibleElement();
      rememberVisibleTextOffset("scroll");
    };

    el.addEventListener("scroll", onScroll, { passive: true });
    lastVisibleElementRef.current = getScrollVisibleElement();
    rememberVisibleTextOffset("scroll");

    return () => el.removeEventListener("scroll", onScroll);
  }, [scrollLayout, htmlContent]);

  const getPagedVisibleElement = (targetPageIndex: number) => {
    const container = getPagedScrollContainer();
    if (!container) return lastVisibleElementRef.current;

    const metrics = getPagedScrollMetrics();
    if (!metrics || metrics.scrollStep <= 0) return lastVisibleElementRef.current;

    const containerRect = container.getBoundingClientRect();
    const scrollStep = metrics.scrollStep;
    const currentScrollLeft = container.scrollLeft;

    const candidates = container.querySelectorAll<HTMLElement>("p, h1, h2, h3, h4, h5, h6, li, figure, img, blockquote");
    for (let i = 0; i < candidates.length; i++) {
      const el = candidates[i];
      const rect = el.getBoundingClientRect();
      if (rect.width === 0 && rect.height === 0) continue;
      const relativeLeft = (rect.left - containerRect.left) + currentScrollLeft;
      const elPage = Math.max(0, Math.floor(relativeLeft / scrollStep));
      if (elPage >= targetPageIndex) {
        return el;
      }
    }
    return lastVisibleElementRef.current;
  };

  const getVisibleTextOffset = (mode: "scroll" | "paged"): number | null => {
    const container = columnsRef.current || contentRef.current;
    const viewport = mode === "scroll" ? contentRef.current : getPagedScrollContainer();
    if (!container || !viewport) return null;

    const viewportRect = viewport.getBoundingClientRect();
    const walker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT, null);
    let currentOffset = 0;

    while (walker.nextNode()) {
      const node = walker.currentNode as Text;
      const nodeLength = node.textContent?.length || 0;
      if (nodeLength === 0 || !node.textContent?.trim()) {
        currentOffset += nodeLength;
        continue;
      }

      try {
        const range = document.createRange();
        range.selectNodeContents(node);
        const visible = Array.from(range.getClientRects()).some((rect) => {
          if (rect.width === 0 && rect.height === 0) return false;
          if (mode === "paged") {
            return rect.right > viewportRect.left + 2 && rect.left < viewportRect.right - 2;
          }
          return rect.bottom > viewportRect.top + 10 && rect.top < viewportRect.bottom - 10;
        });

        if (visible) {
          let low = 0;
          let high = nodeLength;
          let foundOffset = 0;

          while (low <= high) {
            const mid = Math.floor((low + high) / 2);
            const subRange = document.createRange();
            subRange.setStart(node, mid);
            subRange.setEnd(node, nodeLength);
            const firstSub = Array.from(subRange.getClientRects()).find((rect) => rect.width > 0 || rect.height > 0);
            if (!firstSub) {
              high = mid - 1;
              continue;
            }

            const beforeViewport = mode === "paged"
              ? firstSub.right <= viewportRect.left + 2
              : firstSub.bottom <= viewportRect.top + 10;
            if (beforeViewport) {
              low = mid + 1;
            } else {
              foundOffset = mid;
              high = mid - 1;
            }
          }

          return currentOffset + foundOffset;
        }
      } catch {}

      currentOffset += nodeLength;
    }

    return null;
  };

  const rememberVisibleTextOffset = (mode: "scroll" | "paged") => {
    const offset = getVisibleTextOffset(mode);
    if (offset != null) {
      lastVisibleTextOffsetRef.current = offset;
    }
  };

  // In paged mode, track the visible element on current page and update fraction
  useEffect(() => {
    if (scrollLayout || isResizingRef.current || prevModeRef.current !== effectiveReadingMode) return;
    const container = getPagedScrollContainer();
    if (!container) return;

    const metrics = getPagedScrollMetrics();
    if (metrics && metrics.scrollStep > 0) {
      if (metrics.maxIndex > 0) {
        lastScrollFractionRef.current = Math.min(1, Math.max(0, pageIndex / metrics.maxIndex));
        lastKnownFractionRef.current = lastScrollFractionRef.current;
      }
      lastVisibleElementRef.current = getPagedVisibleElement(pageIndex);
      rememberVisibleTextOffset("paged");
    }
  }, [scrollLayout, pageIndex, htmlContent, effectiveReadingMode]);

  useEffect(() => {
    if (scrollLayout) {
      setPageFrameWidth(0);
      return;
    }

    const frame = pageFrameRef.current;
    if (!frame) return;

    let rafId: number;
    let resizeTimer: ReturnType<typeof setTimeout>;

    const updatePageFrameWidth = () => {
      cancelAnimationFrame(rafId);
      clearTimeout(resizeTimer);

      isResizingRef.current = true;
      cachedNodesContentRef.current = "";
      let targetPage = lastPageIndexRef.current;

      setPageFrameWidth(frame.clientWidth);

      if (frame.clientHeight > 0) {
        frame.style.setProperty("--reader-page-height", `${frame.clientHeight}px`);
      }

      const container = getPagedScrollContainer();
      if (scrollLayout) {
        if (container) {
          container.scrollLeft = 0;
        }
        if (columnsRef.current) {
          columnsRef.current.scrollLeft = 0;
        }
        isResizingRef.current = false;
        return;
      }

      if (container) {
        if (frame.clientHeight > 0) {
          container.style.setProperty("--reader-page-height", `${frame.clientHeight}px`);
        }
        const newStep = frame.clientWidth + READER_PAGE_GAP;

        if (newStep > 0) {
          if (targetPage === 0) {
            const targetEl = lastVisibleElementRef.current;
            const fraction = lastScrollFractionRef.current;
            const metrics = getPagedScrollMetrics();

            if (targetEl && container.contains(targetEl)) {
              let offsetLeft = targetEl.offsetLeft;
              let parent = targetEl.offsetParent as HTMLElement | null;
              while (parent && container.contains(parent) && parent !== container) {
                offsetLeft += parent.offsetLeft;
                parent = parent.offsetParent as HTMLElement | null;
              }
              targetPage = Math.max(0, Math.floor(offsetLeft / newStep));
            }

            if ((targetPage === 0 || !targetEl) && fraction > 0.05 && metrics && metrics.maxIndex > 0) {
              targetPage = Math.round(fraction * metrics.maxIndex);
            }

            if (metrics) {
              targetPage = Math.min(targetPage, metrics.maxIndex);
            }
          }

          lastPageIndexRef.current = targetPage;
        }

        const targetLeft = targetPage * newStep * (rtlPaging ? -1 : 1);
        if (typeof container.scrollTo === "function") {
          container.scrollTo({ left: targetLeft, behavior: "auto" });
        } else {
          container.scrollLeft = targetLeft;
        }
      }

      rafId = requestAnimationFrame(() => {
        scrollToPageIndex(lastPageIndexRef.current, true);
        resizeTimer = setTimeout(() => {
          isResizingRef.current = false;
        }, 60);
      });
    };

    updatePageFrameWidth();

    const resizeObserver = new ResizeObserver(updatePageFrameWidth);
    resizeObserver.observe(frame);
    window.addEventListener("resize", updatePageFrameWidth);

    return () => {
      cancelAnimationFrame(rafId);
      clearTimeout(resizeTimer);
      resizeObserver.disconnect();
      window.removeEventListener("resize", updatePageFrameWidth);
    };
  }, [scrollLayout, maxWidth, effectiveReadingMode, rtlPaging]);

  useEffect(() => {
    if (!isResizingRef.current) {
      lastPageIndexRef.current = pageIndex;
    }
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
    if (scrollLayout) return;
    const metrics = getPagedScrollMetrics();
    if (!metrics) return;

    const { container, scrollStep, maxIndex } = metrics;
    const nextIndex = Math.min(Math.max(targetIndex, 0), maxIndex);
    lastPageIndexRef.current = nextIndex;
    lastKnownFractionRef.current = maxIndex > 0 ? Math.min(1, Math.max(0, nextIndex / maxIndex)) : 0;
    requestAnimationFrame(() => rememberVisibleTextOffset("paged"));

    const isSlide = !instant && pageAnimation === "slide";
    const behavior = isSlide ? "smooth" : "auto";

    const targetLeft = nextIndex * scrollStep * (rtlPaging ? -1 : 1);
    isResizingRef.current = true;
    if (typeof container.scrollTo === "function") {
      container.scrollTo({
        left: targetLeft,
        behavior,
      });
    } else {
      container.scrollLeft = targetLeft;
    }

    if (!instant && nextIndex !== pageIndex) {
      triggerPageAnimation();
    }

    setPageIndex(nextIndex);
    setTimeout(() => {
      isResizingRef.current = false;
    }, 100);
  };

  const restoreTextOffsetToScroll = (offset: number): boolean => {
    const scrollEl = contentRef.current;
    const container = columnsRef.current || scrollEl;
    if (!scrollEl || !container || offset < 0) return false;

    const range = createRangeFromCharOffset(container, offset, offset + 1);
    if (!range) return false;

    try {
      const rect = range.getBoundingClientRect();
      const scrollRect = scrollEl.getBoundingClientRect();
      if (rect.width === 0 && rect.height === 0 && rect.top === 0 && rect.bottom === 0) return false;
      const maxScroll = Math.max(0, scrollEl.scrollHeight - scrollEl.clientHeight);
      scrollEl.scrollTop = Math.max(0, Math.min(maxScroll, scrollEl.scrollTop + (rect.top - scrollRect.top) - 24));
      lastVisibleTextOffsetRef.current = offset;
      return true;
    } catch {
      return false;
    }
  };

  const restoreTextOffsetToPaged = (offset: number): boolean => {
    if (offset < 0) return false;
    const metrics = getPagedScrollMetrics();
    const container = columnsRef.current || metrics?.container;
    if (!metrics || !container) return false;

    const range = createRangeFromCharOffset(container as HTMLElement, offset, offset + 1);
    if (!range) return false;

    try {
      const rect = range.getBoundingClientRect();
      const containerRect = metrics.container.getBoundingClientRect();
      const relativeLeft = (rect.left - containerRect.left) + metrics.container.scrollLeft;
      const targetIndex = Math.min(metrics.maxIndex, Math.max(0, Math.floor(relativeLeft / metrics.scrollStep)));
      scrollToPageIndex(targetIndex, true);
      lastVisibleTextOffsetRef.current = offset;
      return true;
    } catch {
      return false;
    }
  };

  // Fractional location within the current chapter (0–1) for true progress.
  const getLocationFraction = (): number => {
    if (scrollLayout && contentRef.current) {
      const el = contentRef.current;
      const max = el.scrollHeight - el.clientHeight;
      return max > 0 ? Math.min(1, Math.max(0, el.scrollTop / max)) : lastScrollFractionRef.current;
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
    const isPrevScroll = prevMode === "scroll" || prevMode === "webtoon";
    const isNewScroll = effectiveReadingMode === "scroll" || effectiveReadingMode === "webtoon";

    prevModeRef.current = effectiveReadingMode;
    const savedOffset = lastVisibleTextOffsetRef.current;

    if (isPrevPaged && isNewPaged) {
      const prevIdx = lastPageIndexRef.current;
      const prevMetrics = getPagedScrollMetrics();
      const prevFraction = lastKnownFractionRef.current || (prevMetrics && prevMetrics.maxIndex > 0
        ? Math.min(1, Math.max(0, prevIdx / prevMetrics.maxIndex))
        : 0);
      let targetPage = prevIdx;

      const tryOffsetRestore = () => {
        if (savedOffset <= 0) return false;
        const restored = restoreTextOffsetToPaged(savedOffset);
        if (restored) {
          lastVisibleElementRef.current = getPagedVisibleElement(lastPageIndexRef.current);
        }
        return restored;
      };

      if (!tryOffsetRestore()) {
        if (prevMode === "double" && effectiveReadingMode === "single") {
          targetPage = prevIdx * 2;
        } else if (prevMode === "single" && effectiveReadingMode === "double") {
          targetPage = Math.floor(prevIdx / 2);
        }

        lastPageIndexRef.current = targetPage;
        lastVisibleElementRef.current = getPagedVisibleElement(targetPage);
      }

      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          if (!tryOffsetRestore()) scrollToPageIndex(targetPage, true);
          lastKnownFractionRef.current = prevFraction;
          lastScrollFractionRef.current = prevFraction;
          modeTransitionFractionRef.current = prevFraction;
        });
      });
      return;
    }

    if (isPrevPaged && isNewScroll) {
      const prevIdx = lastPageIndexRef.current;
      const targetEl = getPagedVisibleElement(prevIdx) || lastVisibleElementRef.current;
      const metrics = getPagedScrollMetrics();
      const prevFraction = modeTransitionFractionRef.current ?? (lastKnownFractionRef.current || ((metrics && metrics.maxIndex > 0)
        ? Math.min(1, Math.max(0, prevIdx / metrics.maxIndex))
        : lastScrollFractionRef.current));
      modeTransitionFractionRef.current = null;

      const container = getPagedScrollContainer();
      if (container) {
        container.scrollLeft = 0;
      }
      if (columnsRef.current) {
        columnsRef.current.scrollLeft = 0;
      }

      const restoreScrollPosition = () => {
        if (savedOffset > 0 && restoreTextOffsetToScroll(savedOffset)) {
          return;
        }

        const el = contentRef.current;
        if (!el) return;
        const c = getPagedScrollContainer();
        if (c) c.scrollLeft = 0;
        if (columnsRef.current) columnsRef.current.scrollLeft = 0;

        const maxScroll = el.scrollHeight - el.clientHeight;

        if (targetEl && el.contains(targetEl)) {
          let top = targetEl.offsetTop;
          let parent = targetEl.offsetParent as HTMLElement | null;
          while (parent && el.contains(parent) && parent !== el) {
            top += parent.offsetTop;
            parent = parent.offsetParent as HTMLElement | null;
          }
          if (top > 0 || prevFraction <= 0.02) {
            el.scrollTop = Math.min(maxScroll, Math.max(0, top - 20));
          } else if (prevFraction > 0 && maxScroll > 0) {
            el.scrollTop = Math.round(prevFraction * maxScroll);
          }
        } else if (prevFraction > 0 && maxScroll > 0) {
          el.scrollTop = Math.round(prevFraction * maxScroll);
        }

        const newMax = el.scrollHeight - el.clientHeight;
        lastScrollFractionRef.current = newMax > 0 ? el.scrollTop / newMax : prevFraction;
        lastKnownFractionRef.current = lastScrollFractionRef.current;
      };

      restoreScrollPosition();
      requestAnimationFrame(() => {
        restoreScrollPosition();
        setTimeout(restoreScrollPosition, 50);
        setTimeout(restoreScrollPosition, 150);
      });
      return;
    }

    if (isPrevScroll && isNewPaged) {
      const targetEl = lastVisibleElementRef.current;
      const fraction = lastKnownFractionRef.current || lastScrollFractionRef.current;
      isResizingRef.current = true;

      const restorePagedPosition = () => {
        if (savedOffset > 0 && restoreTextOffsetToPaged(savedOffset)) {
          return;
        }

        const container = getPagedScrollContainer();
        if (!container) return;

        const scrollStep = container.clientWidth + READER_PAGE_GAP;
        const metrics = getPagedScrollMetrics();
        let targetIndex = lastPageIndexRef.current;

        if (targetIndex === 0) {
          if (targetEl && container.contains(targetEl) && scrollStep > 0) {
            const containerRect = container.getBoundingClientRect();
            const rect = targetEl.getBoundingClientRect();
            if (rect.width > 0 || rect.height > 0) {
              const relativeLeft = (rect.left - containerRect.left) + container.scrollLeft;
              targetIndex = Math.max(0, Math.floor(relativeLeft / scrollStep));
            } else {
              let offsetLeft = targetEl.offsetLeft;
              let parent = targetEl.offsetParent as HTMLElement | null;
              while (parent && container.contains(parent) && parent !== container) {
                offsetLeft += parent.offsetLeft;
                parent = parent.offsetParent as HTMLElement | null;
              }
              targetIndex = Math.max(0, Math.floor(offsetLeft / scrollStep));
            }
          }

          if ((targetIndex === 0 || !targetEl) && fraction > 0.05 && metrics && metrics.maxIndex > 0) {
            targetIndex = Math.round(fraction * metrics.maxIndex);
          }

          if (metrics) {
            targetIndex = Math.min(targetIndex, metrics.maxIndex);
          }
        }

        lastPageIndexRef.current = targetIndex;
        scrollToPageIndex(targetIndex, true);
        setTimeout(() => {
          isResizingRef.current = false;
        }, 120);
      };

      restorePagedPosition();
      requestAnimationFrame(() => {
        restorePagedPosition();
      });
      return;
    }
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
