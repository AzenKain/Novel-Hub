import { useEffect, useRef, useState, type RefObject } from "react";

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

type ToolbarRect = Pick<DOMRect, "left" | "width" | "top"> & {
  height?: number;
  bottom?: number;
  lastRect?: DOMRect | null;
};

export function getVisibleSelectionRect(range: Range): ToolbarRect {
  const rects = range.getClientRects();
  const vw = window.innerWidth;
  const vh = window.innerHeight;

  let minTop = Infinity;
  let maxBottom = -Infinity;
  let minLeft = Infinity;
  let maxRight = -Infinity;
  let found = false;
  let lastVisibleRect: DOMRect | null = null;

  for (let i = 0; i < rects.length; i++) {
    const r = rects[i];
    if (r.right < 0 || r.left > vw || r.bottom < 0 || r.top > vh) continue;
    if (r.width === 0 && r.height === 0) continue;
    found = true;
    lastVisibleRect = r;
    minTop = Math.min(minTop, r.top);
    maxBottom = Math.max(maxBottom, r.bottom);
    minLeft = Math.min(minLeft, r.left);
    maxRight = Math.max(maxRight, r.right);
  }

  if (!found) {
    const br = range.getBoundingClientRect();
    return {
      left: br.left,
      width: br.width,
      top: br.top,
      height: br.height,
      bottom: br.bottom,
      lastRect: br,
    };
  }

  // Clamp to viewport edges so toolbar remains on-screen.
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
    lastRect:
      lastVisibleRect || (rects.length > 0 ? rects[rects.length - 1] : null),
  };
}

export function getToolbarPosition(
  rect: ToolbarRect,
  viewportWidth: number,
  viewportHeight = typeof window !== "undefined" ? window.innerHeight : 800,
  customLastRect?: DOMRect | null,
): { top: number; left: number; placement: "above" | "below" } {
  const margin = 8;
  const topBarHeight = 56;
  const toolbarEstimatedHeight = 175;
  const availableWidth = Math.min(380, Math.max(0, viewportWidth - margin * 2));
  const lastRect =
    customLastRect !== undefined ? customLastRect : rect.lastRect;
  const focalX = lastRect
    ? (lastRect.left + lastRect.right) / 2
    : rect.left + rect.width / 2;
  const maxLeft = Math.max(margin, viewportWidth - margin - availableWidth);
  const left = Math.min(Math.max(focalX - availableWidth / 2, margin), maxLeft);

  const spaceAbove = rect.top - topBarHeight;
  const spaceBelow =
    viewportHeight - (rect.bottom ?? rect.top + (rect.height ?? 24));

  if (spaceAbove >= toolbarEstimatedHeight + margin) {
    return {
      top: Math.max(
        topBarHeight + margin,
        rect.top - toolbarEstimatedHeight - 8,
      ),
      left,
      placement: "above",
    };
  }

  if (spaceBelow >= toolbarEstimatedHeight + margin) {
    const bottom = rect.bottom ?? rect.top + (rect.height ?? 24);
    return {
      top: bottom + 8,
      left,
      placement: "below",
    };
  }

  // For tall selections where neither above nor below fits completely:
  // Intelligently follow the active end of selection if space permits
  if (
    lastRect &&
    lastRect.bottom + toolbarEstimatedHeight + margin <= viewportHeight
  ) {
    return {
      top: lastRect.bottom + 8,
      left,
      placement: "below",
    };
  }

  if (
    lastRect &&
    lastRect.top - toolbarEstimatedHeight - margin >= topBarHeight + margin
  ) {
    return {
      top: lastRect.top - toolbarEstimatedHeight - 8,
      left,
      placement: "above",
    };
  }

  if (lastRect) {
    return {
      top: Math.max(
        topBarHeight + margin + 8,
        viewportHeight - toolbarEstimatedHeight - margin,
      ),
      left,
      placement: "below",
    };
  }

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
  addHighlight: (
    text: string,
    start: number,
    end: number,
    color: string,
    cfi_range?: string,
    note?: string,
  ) => Promise<unknown>;
  speak: (text: string) => void;
  stop: () => void;
  chapterIndex?: number;
  chapterId?: string;
};

/** Tracks text selection and toolbar positioning. */
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
  const selectionRangeRef = useRef<Range | null>(null);
  selectionRangeRef.current = selectionRange;

  const [toolbarPos, setToolbarPos] = useState<{
    top: number;
    left: number;
    placement: "above" | "below";
  }>({
    top: 0,
    left: 0,
    placement: "above",
  });

  useEffect(() => {
    const container = columnsRef.current || contentRef.current;
    if (selectionRange) {
      setActiveSelectionHighlight(
        selectionRange,
        container,
        savedSelectionRef.current,
      );
    } else {
      clearActiveSelectionHighlight();
    }
    return () => {
      clearActiveSelectionHighlight();
    };
  }, [selectionRange]);

  useEffect(() => {
    let timeoutId: ReturnType<typeof setTimeout> | null = null;
    let isMouseDown = false;
    let isTouchActive = false;

    const runSelectionCheck = () => {
      const selection = window.getSelection();

      if (selection && selection.rangeCount > 0 && !selection.isCollapsed) {
        const range = selection.getRangeAt(0);
        const container = columnsRef.current || contentRef.current;
        if (!container) return;

        const commonNode =
          range.commonAncestorContainer.nodeType === Node.TEXT_NODE
            ? range.commonAncestorContainer.parentNode
            : range.commonAncestorContainer;

        const isValidSelection =
          commonNode &&
          (container.contains(commonNode) ||
            commonNode.contains(container) ||
            commonNode === container);
        if (isValidSelection && range.toString().trim().length > 0) {
          const cloned = range.cloneRange();
          selectionRangeRef.current = cloned;
          setSelectionRange(cloned);
          const rect = getVisibleSelectionRect(range);
          setToolbarPos(
            getToolbarPosition(rect, window.innerWidth, window.innerHeight),
          );

          // Calculate saved selection asynchronously in background without blocking UI render
          if (typeof window !== "undefined" && "requestIdleCallback" in window) {
            (window as any).requestIdleCallback(() => {
              savedSelectionRef.current = saveSelection(container, cloned);
            });
          } else {
            setTimeout(() => {
              savedSelectionRef.current = saveSelection(container, cloned);
            }, 0);
          }
          return;
        }
      }

      savedSelectionRef.current = null;
      selectionRangeRef.current = null;
      setSelectionRange((prev) => (prev !== null ? null : prev));
    };

    const handleSelection = (e?: Event) => {
      if (e?.target) {
        const targetNode = e.target as Node | null;
        const targetElem =
          targetNode?.nodeType === Node.ELEMENT_NODE
            ? (targetNode as HTMLElement)
            : targetNode?.parentElement;
        const isToolbarOrModal = !!targetElem?.closest?.(
          '[data-reader-toolbar="true"], [data-reader-modal="true"], .modal, [role="dialog"]',
        );

        if (isToolbarOrModal) {
          return;
        }
      }

      if (timeoutId) clearTimeout(timeoutId);
      timeoutId = setTimeout(runSelectionCheck, 10);
    };

    const onMouseDown = (e: MouseEvent) => {
      const targetNode = e.target as Node | null;
      const targetElem =
        targetNode?.nodeType === Node.ELEMENT_NODE
          ? (targetNode as HTMLElement)
          : targetNode?.parentElement;
      if (
        targetElem?.closest?.(
          '[data-reader-toolbar="true"], [data-reader-modal="true"], .modal, [role="dialog"]',
        )
      ) {
        return;
      }
      isMouseDown = true;
    };

    const onMouseUp = (e: MouseEvent) => {
      isMouseDown = false;
      handleSelection(e);
    };

    const onTouchStart = (e: TouchEvent) => {
      const targetNode = e.target as Node | null;
      const targetElem =
        targetNode?.nodeType === Node.ELEMENT_NODE
          ? (targetNode as HTMLElement)
          : targetNode?.parentElement;
      if (
        targetElem?.closest?.(
          '[data-reader-toolbar="true"], [data-reader-modal="true"], .modal, [role="dialog"]',
        )
      ) {
        return;
      }
      isTouchActive = true;
    };

    const onTouchEnd = (e: TouchEvent) => {
      isTouchActive = false;
      handleSelection(e);
    };

    const onTouchCancel = () => {
      isTouchActive = false;
    };

    const onSelectionChange = (e: Event) => {
      // If mouse or touch is currently dragging, wait until mouseup/touchend
      if (isMouseDown || isTouchActive) {
        return;
      }
      handleSelection(e);
    };

    const onReposition = () => {
      const activeRange = selectionRangeRef.current;
      if (activeRange) {
        const rect = getVisibleSelectionRect(activeRange);
        setToolbarPos(
          getToolbarPosition(rect, window.innerWidth, window.innerHeight),
        );
      }
    };

    document.addEventListener("mousedown", onMouseDown);
    document.addEventListener("mouseup", onMouseUp);
    document.addEventListener("touchstart", onTouchStart, { passive: true });
    document.addEventListener("touchend", onTouchEnd, { passive: true });
    document.addEventListener("touchcancel", onTouchCancel, { passive: true });
    document.addEventListener("keyup", handleSelection);
    document.addEventListener("selectionchange", onSelectionChange);
    window.addEventListener("resize", onReposition, { passive: true });
    window.addEventListener("scroll", onReposition, { passive: true });

    return () => {
      if (timeoutId) clearTimeout(timeoutId);
      document.removeEventListener("mousedown", onMouseDown);
      document.removeEventListener("mouseup", onMouseUp);
      document.removeEventListener("touchstart", onTouchStart);
      document.removeEventListener("touchend", onTouchEnd);
      document.removeEventListener("touchcancel", onTouchCancel);
      document.removeEventListener("keyup", handleSelection);
      document.removeEventListener("selectionchange", onSelectionChange);
      window.removeEventListener("resize", onReposition);
      window.removeEventListener("scroll", onReposition);
    };
  }, [columnsRef, contentRef, savedSelectionRef]);

  const handleHighlight = async (color: string, note?: string) => {
    const saved = savedSelectionRef.current;
    const container = columnsRef.current || contentRef.current;

    let selectedText: string;
    let start: number;
    let end: number;

    if (
      saved &&
      saved.endIndex > saved.startIndex &&
      saved.selectedText.trim()
    ) {
      selectedText = saved.selectedText;
      start = saved.startIndex;
      end = saved.endIndex;
    } else {
      if (!selectionRange) {
        return;
      }
      const rangeText = selectionRange.toString();
      const offset = container
        ? getCharacterOffsetOfRange(container, selectionRange)
        : null;
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
      if (
        saved &&
        saved.endIndex > saved.startIndex &&
        saved.selectedText.trim()
      ) {
        range = createRangeFromCharOffset(
          container,
          saved.startIndex,
          saved.endIndex,
        );
      } else if (selectionRange) {
        range = selectionRange;
      }
      if (range) {
        cfiRange = generateCfiRange(
          container,
          range,
          chapterIndex || 0,
          chapterId,
        );
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
    if (!savedSelectionRef.current && container && selectionRange) {
      savedSelectionRef.current = saveSelection(container, selectionRange);
    }
    const saved = savedSelectionRef.current;
    const textToRead = saved?.selectedText || selectionRange?.toString();
    if (container && textToRead && textToRead.trim()) {
      if (saved) {
        ttsStartPointRef.current = {
          textNodeIndex: saved.textNodeIndex,
          offset: saved.offset,
        };
      }
      stop();
      speak(textToRead.trim());
      clearActiveSelectionHighlight();
      savedSelectionRef.current = null;
      setSelectionRange(null);
    }
  };

  const handleReadFromHere = () => {
    const container = columnsRef.current || contentRef.current;
    if (!savedSelectionRef.current && container && selectionRange) {
      savedSelectionRef.current = saveSelection(container, selectionRange);
    }
    const saved = savedSelectionRef.current;
    if (container && saved) {
      const textFromHere = getTextFromHereFromSaved(container, saved);
      if (textFromHere) {
        ttsStartPointRef.current = {
          textNodeIndex: saved.textNodeIndex,
          offset: saved.offset,
        };
        stop();
        speak(textFromHere);
      }
      clearActiveSelectionHighlight();
      savedSelectionRef.current = null;
      setSelectionRange(null);
    }
  };

  const handleCopyText = () => {
    const container = columnsRef.current || contentRef.current;
    if (!savedSelectionRef.current && container && selectionRange) {
      savedSelectionRef.current = saveSelection(container, selectionRange);
    }
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
