import type React from "react";
import { type RefObject } from "react";

import type { Chapter } from "@/types";

type UseReaderNavigationArgs = {
  columnsRef: RefObject<HTMLDivElement | null>;
  pendingFragmentRef: RefObject<string | null>;
  chapters: Chapter[];
  scrollLayout: boolean;
  loadChapter: (chapter: Chapter) => void | Promise<void>;
  getPagedScrollMetrics: () => { container: Element; scrollStep: number; maxIndex: number } | undefined;
  scrollToPageIndex: (targetIndex: number, instant?: boolean) => void;
};

/**
 * Handles links inside chapter HTML: in-page `#fragment` anchors, `section:`
 * links, and the API `/chapter/` and `/asset/` URLs the backend rewrites hrefs
 * into. Also resolves a fragment to a scroll position or page.
 *
 * Extracted verbatim from ReaderWorkspace — behaviour is unchanged.
 */
export function useReaderNavigation({
  columnsRef,
  pendingFragmentRef,
  chapters,
  scrollLayout,
  loadChapter,
  getPagedScrollMetrics,
  scrollToPageIndex,
}: UseReaderNavigationArgs) {
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
    if (scrollLayout) {
      target.scrollIntoView({ behavior: "smooth", block: "start", inline: "start" });
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

  return { handleContentClick, scrollToFragment };
}
