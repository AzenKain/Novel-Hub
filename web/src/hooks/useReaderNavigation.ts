import type React from "react";
import { type RefObject } from "react";

import type { Chapter } from "@/types";
import { getSideTapRatio } from "@/constants";

type UseReaderNavigationArgs = {
  columnsRef: RefObject<HTMLDivElement | null>;
  pendingFragmentRef: RefObject<string | null>;
  chapters: Chapter[];
  scrollLayout: boolean;
  loadChapter: (chapter: Chapter) => void | Promise<void>;
  getPagedScrollMetrics: () =>
    { container: Element; scrollStep: number; maxIndex: number } | undefined;
  scrollToPageIndex: (targetIndex: number, instant?: boolean) => void;
  onPagePrev?: () => void;
  onPageNext?: () => void;
  rtlPaging?: boolean;
  onCenterClick?: () => void;
  onImageClick?: (target: { image_url: string; x: number; y: number }) => void;
};

/** Handles chapter HTML links and navigation. */
export function useReaderNavigation({
  columnsRef,
  pendingFragmentRef,
  chapters,
  scrollLayout,
  loadChapter,
  getPagedScrollMetrics,
  scrollToPageIndex,
  onPagePrev,
  onPageNext,
  rtlPaging = false,
  onCenterClick,
  onImageClick,
}: UseReaderNavigationArgs) {
  const scrollToFragment = (fragment: string) => {
    const normalized = fragment.trim();
    if (!normalized) return;
    const root = columnsRef.current;
    if (!root) return;
    const escaped =
      typeof CSS !== "undefined" && typeof CSS.escape === "function"
        ? CSS.escape(normalized)
        : normalized.replace(/["\\.#:[\]>+~()]/g, "\\$&");
    const target = root.querySelector<HTMLElement>(`#${escaped}`);
    if (!target) return;
    if (scrollLayout) {
      target.scrollIntoView({
        behavior: "smooth",
        block: "start",
        inline: "start",
      });
      return;
    }
    // paged: jump to the page containing the target element
    const metrics = getPagedScrollMetrics();
    if (!metrics) return;
    let left = target.offsetLeft;
    let parent = target.offsetParent as HTMLElement | null;
    while (parent && parent !== metrics.container) {
      left += parent.offsetLeft;
      parent = parent.offsetParent as HTMLElement | null;
    }
    const pageIndex = Math.round(left / metrics.scrollStep);
    scrollToPageIndex(pageIndex);
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
          const found = chapters.find((ch) => ch.content_path === sectionPath);
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
            const found = chapters.find((ch) => ch.id === chId);
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
            const resolvedPath = decodeURIComponent(
              parts[1].split("#")[0].split("?")[0],
            );
            const targetPath = resolvedPath.toLowerCase().replace(/^\/+/, "");
            const found = chapters.find((ch) => {
              const chPath = ch.content_path?.toLowerCase().replace(/^\/+/, "");
              return (
                chPath === targetPath ||
                (chPath && targetPath.endsWith(chPath)) ||
                (chPath && chPath.endsWith(targetPath))
              );
            });
            if (found) {
              void loadChapter(found);
              return;
            }
          }
        }
      }
    }

    // In paged mode (single/double):
    // On large screens (>= 1024px): tap-to-turn is disabled (0%), clicking content triggers center click / selection
    // On smaller screens (< 1024px down to mobile): side tap zones scale gradually up to 30%
    if (!scrollLayout) {
      // Don't turn pages if user is selecting text
      const selection = window.getSelection()?.toString().trim();
      if (selection) return;

      // Don't turn pages if clicking interactive buttons, inputs, or toolbars
      if (
        target.closest("button, input, textarea, select, [data-reader-toolbar]")
      )
        return;

      const rect = e.currentTarget.getBoundingClientRect();
      const screenWidth =
        typeof window !== "undefined" ? window.innerWidth : rect.width;
      const sideRatio = getSideTapRatio(screenWidth);

      if (sideRatio > 0) {
        const clickRatio = (e.clientX - rect.left) / rect.width;
        if (clickRatio < sideRatio) {
          if (rtlPaging) {
            onPageNext?.();
          } else {
            onPagePrev?.();
          }
          return;
        }
        if (clickRatio > 1 - sideRatio) {
          if (rtlPaging) {
            onPagePrev?.();
          } else {
            onPageNext?.();
          }
          return;
        }
      }

      if (target.tagName.toLowerCase() === "img") {
        const img = target as HTMLImageElement;
        const src = img.currentSrc || img.src;
        if (src) {
          onImageClick?.({
            image_url: src,
            x: e.clientX,
            y: e.clientY,
          });
          return;
        }
      }

      onCenterClick?.();
    }
  };

  return { handleContentClick, scrollToFragment };
}
