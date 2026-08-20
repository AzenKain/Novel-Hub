import React, { useEffect, useMemo } from "react";

import type { PageFit, ReadingDirection, ReadingMode } from "@/stores";
import { isVisualChapter, sanitizeReaderHtml } from "@/utils/readerHtml";

type ReaderContentProps = {
  htmlContent: string;
  proseClass: string;
  effectiveReadingMode: ReadingMode;
  readingDirection: ReadingDirection;
  pageFit: PageFit;
  pageWidth: number;
  columnsRef: React.RefObject<HTMLDivElement | null>;
  isPendingLanding?: boolean;
  onContentClick: (event: React.MouseEvent<HTMLDivElement>) => void;
};

const isRawReaderContent = (html: string) =>
  html.includes('class="novelhub-raw-reader"');

export const ReaderContent: React.FC<ReaderContentProps> = React.memo(({
  htmlContent,
  proseClass,
  effectiveReadingMode,
  readingDirection,
  pageFit,
  pageWidth,
  columnsRef,
  isPendingLanding = false,
  onContentClick,
}) => {
  const sanitizedHTML = useMemo(
    () => sanitizeReaderHtml(htmlContent),
    [htmlContent],
  );
  const visualChapter = useMemo(
    () => isVisualChapter(sanitizedHTML),
    [sanitizedHTML],
  );
  const rawReader = useMemo(
    () => isRawReaderContent(sanitizedHTML),
    [sanitizedHTML],
  );

  // Clean up broken images and empty wrappers at runtime
  useEffect(() => {
    const el = columnsRef.current;
    if (!el) return;

    const images = el.querySelectorAll("img");
    images.forEach((img) => {
      const src = img.getAttribute("src");
      if (!src || src === "#" || src === "about:blank") {
        img.style.display = "none";
        const parent = img.parentElement;
        if (parent && parent.children.length === 1 && !parent.textContent?.trim()) {
          parent.style.display = "none";
        }
      }

      img.onerror = () => {
        img.style.display = "none";
        const parent = img.parentElement;
        if (parent && parent.children.length === 1 && !parent.textContent?.trim()) {
          parent.style.display = "none";
        }
      };
    });
  }, [sanitizedHTML, columnsRef]);

  if (!htmlContent) return null;

  return (
    <div
      ref={columnsRef}
      onClick={onContentClick}
      style={isPendingLanding ? { visibility: "hidden" } : undefined}
      className={`reader-content ${visualChapter ? "reader-content-visual" : ""} ${proseClass} max-w-none w-full ${
        rawReader
          ? "flex-1 min-h-0 h-full"
          : effectiveReadingMode === "scroll" ? "h-auto" : "flex-1 min-h-0"
      } [&>body]:m-0 [&>body]:p-0 [&>body]:display-block [&_img]:max-w-full [&_img]:h-auto ${
        effectiveReadingMode === "single"
          ? "reader-mode-single"
          : effectiveReadingMode === "double"
            ? "reader-mode-double"
            : ""
      } ${effectiveReadingMode !== "scroll" && pageWidth > 0 ? "reader-mode-measured" : ""} ${
        visualChapter && readingDirection === "rtl" ? "reader-dir-rtl" : ""
      } ${visualChapter ? `reader-fit-${pageFit}` : ""}`}
      dangerouslySetInnerHTML={{ __html: sanitizedHTML }}
    />
  );
});