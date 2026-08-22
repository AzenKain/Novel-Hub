import React, { useEffect, useMemo } from "react";

import type { ReadingDirection, ReadingMode } from "@/stores";
import { sanitizeReaderHtml } from "@/utils/readerHtml";

type ReaderContentProps = {
  htmlContent: string;
  proseClass: string;
  effectiveReadingMode: ReadingMode;
  readingDirection: ReadingDirection;
  pageWidth: number;
  columnsRef: React.RefObject<HTMLDivElement | null>;
  onContentClick: (event: React.MouseEvent<HTMLDivElement>) => void;
};

const isRawReaderContent = (html: string) =>
  html.includes('class="novelhub-raw-reader"');

export const ReaderContent: React.FC<ReaderContentProps> = React.memo(({
  htmlContent,
  proseClass,
  effectiveReadingMode,
  readingDirection,
  pageWidth,
  columnsRef,
  onContentClick,
}) => {
  const sanitizedHTML = useMemo(
    () => sanitizeReaderHtml(htmlContent),
    [htmlContent],
  );
  const rawReader = useMemo(
    () => isRawReaderContent(sanitizedHTML),
    [sanitizedHTML],
  );

  // Clean up broken images, tag wrappers and inject floating bookmark button in paged modes
  useEffect(() => {
    const el = columnsRef.current;
    if (!el) return;

    const isPaged = effectiveReadingMode === "single" || effectiveReadingMode === "double";
    const images = el.querySelectorAll("img");
    images.forEach((img) => {
      const src = img.getAttribute("src");
      if (!src || src === "#" || src === "about:blank") {
        img.style.display = "none";
        const parent = img.parentElement;
        if (parent && parent.children.length === 1 && !parent.textContent?.trim()) {
          parent.style.display = "none";
        }
        return;
      }

      // Tag immediate block wrapper as reader-image-page
      let wrapper = img.parentElement;
      if (wrapper && wrapper.tagName.toLowerCase() === "a") {
        wrapper = wrapper.parentElement;
      }
      if (wrapper && !wrapper.classList.contains("reader-image-page") && !wrapper.classList.contains("reader-content")) {
        wrapper.classList.add("reader-image-page");
        wrapper.style.position = "relative";
      }

      img.style.cursor = "pointer";

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
      className={`reader-content ${proseClass} max-w-none w-full ${
        rawReader
          ? "flex-1 min-h-0 h-full"
          : effectiveReadingMode === "scroll" ? "h-auto" : "flex-1 min-h-0 h-full"
      } [&>body]:m-0 [&>body]:p-0 [&>body]:display-block ${
        effectiveReadingMode === "scroll" ? "[&_img]:max-w-full [&_img]:h-auto" : ""
      } ${
        effectiveReadingMode === "single"
          ? "reader-mode-single"
          : effectiveReadingMode === "double"
            ? "reader-mode-double"
            : ""
      } ${effectiveReadingMode !== "scroll" && pageWidth > 0 ? "reader-mode-measured" : ""} ${
        readingDirection === "rtl" ? "reader-dir-rtl" : ""
      }`}
      dangerouslySetInnerHTML={{ __html: sanitizedHTML }}
    />
  );
});