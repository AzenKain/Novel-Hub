import { useEffect, useState, type RefObject } from "react";

import {
  getCharacterOffsetOfRange,
  getTextFromHereFromSaved,
  saveSelection,
  createRangeFromCharOffset,
  setActiveSelectionHighlight,
  clearActiveSelectionHighlight,
  type SavedSelection,
  type TtsStartPoint,
} from "@/lib/readerHighlight";
import { generateCfiRange } from "@/lib/epubCfi";
import { copyText } from "@/utils/clipboard";

type ToolbarRect = Pick<DOMRect, "left" | "width" | "top"> & { height?: number; bottom?: number };

/**
 * In CSS multi-column layouts (reader-mode-single / reader-mode-double), a
 * full-page text selection spans *all* columns—including those hidden by
 * `overflow: hidden`.  `range.getBoundingClientRect()` returns the union of
 * every column rect, which can be thousands of pixels wide and whose center
 * is far off-screen.
 *
 * This helper calls `range.getClientRects()` and keeps only the rects that
 * actually intersect the current viewport, then returns a tight bounding box
 * around them.  If no client rect is visible (edge-case), it falls back to
 * `getBoundingClientRect()`.
 */
export function getVisibleSelectionRect(range: Range): ToolbarRect {
  const rects = range.getClientRects();
  const vw = window.innerWidth;
  const vh = window.innerHeight;

  let minTop = Infinity;
  let maxBottom = -Infinity;
  let minLeft = Infinity;
  let maxRight = -Infinity;
  let found = false;

  for (let i = 0; i < rects.length; i++) {
    const r = rects[i];
    // Skip rects entirely outside the viewport
    if (r.right < 0 || r.left > vw || r.bottom < 0 || r.top > vh) continue;
    // Skip zero-size rects (collapsed whitespace, etc.)
    if (r.width === 0 && r.height === 0) continue;
    found = true;
    minTop = Math.min(minTop, r.top);
    maxBottom = Math.max(maxBottom, r.bottom);
    minLeft = Math.min(minLeft, r.left);
    maxRight = Math.max(maxRight, r.right);
  }

  if (!found) {
    // Fallback: no client rect intersects the viewport
    const br = range.getBoundingClientRect();
    return { left: br.left, width: br.width, top: br.top, height: br.height, bottom: br.bottom };
  }

  // Clamp to viewport edges so the toolbar never anchors off-screen
  const clampedTop = Math.max(0, minTop);
  const clampedBottom = Math.min(vh, maxBottom);
  const clampedLeft = Math.max(0, minLeft);
  const clampedRight = Math.min(vw, maxRight);

  return {
    top: clampedTop,
    bottom: clampedBottom,
    left: clampedLeft,
    width: clampedRight - clampedLeft,
    height: clampedBottom - clampedTop,
  };
}

export function getToolbarPosition(
  rect: ToolbarRect,
  viewportWidth: number,
  viewportHeight = typeof window !== "undefined" ? window.innerHeight : 800
): { top: number; left: number; placement: "above" | "below" } {
  const margin = 8;
  const topBarHeight = 56;
  const toolbarEstimatedHeight = 175;
  const availableWidth = Math.min(380, Math.max(0, viewportWidth - margin * 2));
  const center = rect.left + rect.width / 2;
  const maxLeft = Math.max(margin, viewportWidth - margin - availableWidth);
  const left = Math.min(
    Math.max(center - availableWidth / 2, margin),
    maxLeft
  );

  const spaceAbove = rect.top - topBarHeight;
  const spaceBelow = viewportHeight - (rect.bottom ?? (rect.top + (rect.height ?? 24)));

  // 1. If there's enough space above selection, position toolbar directly above selection
  if (spaceAbove >= toolbarEstimatedHeight + margin) {
    return {
      top: Math.max(topBarHeight + margin, rect.top - toolbarEstimatedHeight - 8),
      left,
      placement: "above",
    };
  }

  // 2. If there's enough space below selection, position toolbar below selection
  if (spaceBelow >= toolbarEstimatedHeight + margin) {
    const bottom = rect.bottom ?? (rect.top + (rect.height ?? 24));
    return {
      top: bottom + 8,
      left,
      placement: "below",
    };
  }

  // 3. For large selections covering whole page (select-all), position right under top header bar
  return {
    top: topBarHeight + margin + 8,
    left,
    placement: "below",
  };
}

type UseReaderSelectionArgs = {
  columnsRef: RefObject<HTMLDivElement | null>;
  contentRef: RefObject<HTMLDivElement | null>;
  savedSelectionRef: RefObject<SavedSelection | null>;
  ttsStartPointRef: RefObject<TtsStartPoint | null>;
  addHighlight: (text: string, start: number, end: number, color: string, cfi_range?: string, note?: string) => Promise<unknown>;
  speak: (text: string) => void;
  stop: () => void;
  chapterIndex?: number;
  chapterId?: string;
};

/**
 * Tracks the user's text selection inside the reader: where the toolbar should
 * float, and the highlight / copy / speak actions that act on it.
 */
export function useReaderSelection({
  columnsRef,
  contentRef,
  savedSelectionRef,
  ttsStartPointRef,
  addHighlight,
  speak,
  stop,
  chapterIndex,
  chapterId,
}: UseReaderSelectionArgs) {
  const [selectionRange, setSelectionRange] = useState<Range | null>(null);
  const [toolbarPos, setToolbarPos] = useState<{ top: number; left: number; placement: "above" | "below" }>({
    top: 0,
    left: 0,
    placement: "above",
  });

  useEffect(() => {
    const container = columnsRef.current || contentRef.current;
    if (selectionRange) {
      setActiveSelectionHighlight(selectionRange, container, savedSelectionRef.current);
    } else {
      clearActiveSelectionHighlight();
    }
    return () => {
      clearActiveSelectionHighlight();
    };
  }, [selectionRange]);

  useEffect(() => {
    const handleSelection = (e: Event) => {
      const targetNode = e.target as Node | null;
      const targetElem = targetNode?.nodeType === Node.ELEMENT_NODE
        ? (targetNode as HTMLElement)
        : targetNode?.parentElement;
      const isToolbarOrModal = !!targetElem?.closest?.(
        '[data-reader-toolbar="true"], [data-reader-modal="true"], .modal, [role="dialog"]'
      );

      if (isToolbarOrModal) {
        return;
      }

      setTimeout(() => {
        const selection = window.getSelection();

        if (selection && selection.rangeCount > 0 && !selection.isCollapsed) {
          const range = selection.getRangeAt(0);
          const container = columnsRef.current || contentRef.current;
          if (!container) return;

          const commonNode = range.commonAncestorContainer.nodeType === Node.TEXT_NODE
            ? range.commonAncestorContainer.parentNode
            : range.commonAncestorContainer;

          const isValidSelection =
            commonNode &&
            (container.contains(commonNode) ||
              commonNode.contains(container) ||
              commonNode === container);
          if (isValidSelection && range.toString().trim().length > 0) {
            const saved = saveSelection(container, range);
            if (saved) {
              savedSelectionRef.current = saved;
            }
            const cloned = range.cloneRange();
            setSelectionRange(cloned);
            const rect = getVisibleSelectionRect(range);
            setToolbarPos(getToolbarPosition(rect, window.innerWidth, window.innerHeight));
            return;
          }
        }
        savedSelectionRef.current = null;
        setSelectionRange(null);
      }, 20);
    };

    document.addEventListener("mouseup", handleSelection);
    document.addEventListener("touchend", handleSelection);
    document.addEventListener("keyup", handleSelection);
    return () => {
      document.removeEventListener("mouseup", handleSelection);
      document.removeEventListener("touchend", handleSelection);
      document.removeEventListener("keyup", handleSelection);
    };
  }, [columnsRef, contentRef, savedSelectionRef]);

  const handleHighlight = async (color: string, note?: string) => {
    const saved = savedSelectionRef.current;
    const container = columnsRef.current || contentRef.current;

    let selectedText: string;
    let start: number;
    let end: number;

    if (saved && saved.endIndex > saved.startIndex && saved.selectedText.trim()) {
      selectedText = saved.selectedText;
      start = saved.startIndex;
      end = saved.endIndex;
    } else {
      if (!selectionRange) {
        return;
      }
      const rangeText = selectionRange.toString();
      const offset = container ? getCharacterOffsetOfRange(container, selectionRange) : null;
      if (!rangeText.trim() || !offset || offset.end <= offset.start) {
        if (import.meta.env.DEV) {
          console.warn("[reader] highlight dropped", {
            hasText: Boolean(rangeText.trim()),
            hasOffset: Boolean(offset),
          });
        }
        return;
      }
      selectedText = rangeText;
      start = offset.start;
      end = offset.end;
    }

    let cfiRange: string | undefined = undefined;
    if (container && chapterId) {
      let range: Range | null = null;
      if (saved && saved.endIndex > saved.startIndex && saved.selectedText.trim()) {
        range = createRangeFromCharOffset(container, saved.startIndex, saved.endIndex);
      } else if (selectionRange) {
        range = selectionRange;
      }
      if (range) {
        cfiRange = generateCfiRange(container, range, chapterIndex || 0, chapterId);
      }
    }

    const trimmedNote = note?.trim() || undefined;
    await addHighlight(selectedText, start, end, color, cfiRange, trimmedNote);
    window.getSelection()?.removeAllRanges();
    clearActiveSelectionHighlight();
    savedSelectionRef.current = null;
    setSelectionRange(null);
  };

  const handleReadSelection = () => {
    const container = columnsRef.current || contentRef.current;
    const saved = savedSelectionRef.current;
    if (container && saved && saved.selectedText) {
      ttsStartPointRef.current = { textNodeIndex: saved.textNodeIndex, offset: saved.offset };
      stop();
      speak(saved.selectedText);
      clearActiveSelectionHighlight();
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
      clearActiveSelectionHighlight();
      savedSelectionRef.current = null;
      setSelectionRange(null);
    }
  };

  const handleCopyText = () => {
    const saved = savedSelectionRef.current;
    const textToCopy = saved?.selectedText || selectionRange?.toString();
    if (textToCopy) {
      void copyText(textToCopy);
      window.getSelection()?.removeAllRanges();
      clearActiveSelectionHighlight();
      savedSelectionRef.current = null;
      setSelectionRange(null);
    }
  };

  return {
    selectionRange,
    setSelectionRange,
    toolbarPos,
    handleHighlight,
    handleReadSelection,
    handleReadFromHere,
    handleCopyText,
  };
}
