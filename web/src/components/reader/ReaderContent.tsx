import React, { useMemo } from "react";
import DOMPurify from "dompurify";

import type { ReadingMode } from "@/stores";

type ReaderContentProps = {
  htmlContent: string;
  proseClass: string;
  effectiveReadingMode: ReadingMode;
  pageWidth: number;
  columnsRef: React.RefObject<HTMLDivElement | null>;
  onContentClick: (event: React.MouseEvent<HTMLDivElement>) => void;
};

const sanitizeReaderHtml = (html: string) => {
  const stripped = html
    .replace(/<title\b[^>]*>[\s\S]*?<\/title>/gi, "")
    .replace(/<meta\b[^>]*>/gi, "")
    .replace(/<\/?(?:html|head|body)\b[^>]*>/gi, "");
    
  return DOMPurify.sanitize(stripped, {
    USE_PROFILES: { html: true, svg: true, mathMl: true },
    ADD_ATTR: ['target'],
  });
};

const isVisualChapter = (html: string) => {
  const mediaCount = (html.match(/<(?:img|svg|picture|canvas)\b/gi) || [])
    .length;
  if (mediaCount === 0) return false;

  const text = html
    .replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, "")
    .replace(/<style\b[^>]*>[\s\S]*?<\/style>/gi, "")
    .replace(/<[^>]+>/g, "")
    .replace(/&nbsp;|&#160;/gi, " ")
    .replace(/\s+/g, "")
    .trim();

  return text.length <= 24;
};

export const ReaderContent: React.FC<ReaderContentProps> = ({
  htmlContent,
  proseClass,
  effectiveReadingMode,
  pageWidth,
  columnsRef,
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

  if (!htmlContent) return null;

  return (
    <div
      ref={columnsRef}
      onClick={onContentClick}
      className={`reader-content ${visualChapter ? "reader-content-visual" : ""} ${proseClass} max-w-none w-full ${
        effectiveReadingMode === "scroll" ? "h-auto" : "flex-1 min-h-0"
      } [&>body]:m-0 [&>body]:p-0 [&>body]:display-block [&_img]:max-w-full [&_img]:h-auto ${
        effectiveReadingMode === "single"
          ? "reader-mode-single"
          : effectiveReadingMode === "double"
            ? "reader-mode-double"
            : ""
      } ${effectiveReadingMode !== "scroll" && pageWidth > 0 ? "reader-mode-measured" : ""}`}
      dangerouslySetInnerHTML={{ __html: sanitizedHTML }}
    />
  );
};
