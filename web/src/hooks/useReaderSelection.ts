import { useEffect, useRef, useState, type RefObject } from "react";

import {
  getCharacterOffsetOfRange,
  getTextFromHereFromSaved,
  saveSelection,
  createRangeFromCharOffset,
  type SavedSelection,
  type TtsStartPoint,
} from "@/lib/readerHighlight";
import { generateCfiRange } from "@/lib/epubCfi";
import { copyText } from "@/utils/clipboard";

type ToolbarRect = Pick<DOMRect, "left" | "width" | "top">;

export function getToolbarPosition(rect: ToolbarRect, viewportWidth: number) {
  const margin = 8;
  const availableWidth = Math.min(440, Math.max(0, viewportWidth - margin * 2));
  const center = rect.left + rect.width / 2;
  const maxLeft = Math.max(margin, viewportWidth - margin - availableWidth);
  const left = Math.min(
    Math.max(center - availableWidth / 2, margin),
    maxLeft,
  );
  return {
    top: Math.max(10, rect.top - 40),
    left,
  };
}
type UseReaderSelectionArgs = {
  columnsRef: RefObject<HTMLDivElement | null>;
  contentRef: RefObject<HTMLDivElement | null>;
  savedSelectionRef: RefObject<SavedSelection | null>;
  ttsStartPointRef: RefObject<TtsStartPoint | null>;
  addHighlight: (text: string, start: number, end: number, color: string, cfi_range?: string) => Promise<unknown>;
  speak: (text: string) => void;
  stop: () => void;
  chapterIndex?: number;
  chapterId?: string;
};

/**
 * Tracks the user's text selection inside the reader: where the toolbar should
 * float, and the highlight / copy / speak actions that act on it.
 *
 * Extracted verbatim from ReaderWorkspace — behaviour is unchanged.
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
              // saveSelection captures document-relative char offsets (startIndex/
              // endIndex) while the range is still live, so the highlight can be
              // created later even if the reader DOM is rebuilt (which would
              // invalidate this cloned Range).
              savedSelectionRef.current = saved;
              setSelectionRange(range.cloneRange());
              const rect = range.getBoundingClientRect();
              setToolbarPos(getToolbarPosition(rect, window.innerWidth));
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

  const handleHighlight = async (color: string) => {
    const saved = savedSelectionRef.current;
    const container = columnsRef.current || contentRef.current;

    // Prefer the selection captured at selection time (text + document-relative
    // char offsets). The cloned Range in `selectionRange` state can be
    // invalidated if the reader DOM is rebuilt between selecting text and
    // clicking a color — its boundaries then collapse and toString() returns
    // "". The saved snapshot is taken while the range is still live, so it is
    // the reliable source of truth for the highlight.
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
        // Surface why a highlight was dropped so a "nothing happens, no
        // request, no error" failure stays diagnosable instead of silent.
        if (import.meta.env.DEV) {
          // eslint-disable-next-line no-console
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

    await addHighlight(selectedText, start, end, color, cfiRange);
    window.getSelection()?.removeAllRanges();
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
      void copyText(textToCopy);
      window.getSelection()?.removeAllRanges();
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
