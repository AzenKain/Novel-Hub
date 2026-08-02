import { useEffect, useState, type RefObject } from "react";

import {
  getTextFromHereFromSaved,
  saveSelection,
  type SavedSelection,
  type TtsStartPoint,
} from "@/lib/readerHighlight";

type UseReaderSelectionArgs = {
  columnsRef: RefObject<HTMLDivElement | null>;
  contentRef: RefObject<HTMLDivElement | null>;
  savedSelectionRef: RefObject<SavedSelection | null>;
  ttsStartPointRef: RefObject<TtsStartPoint | null>;
  addHighlight: (text: string, start: number, end: number, color: string) => Promise<unknown>;
  speak: (text: string) => void;
  stop: () => void;
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
