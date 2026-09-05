import React, { useEffect, useMemo } from "react";

import type { ReadingDirection, ReadingMode, TextAlignment } from "@/stores";
import { sanitizeReaderHtml } from "@/utils/readerHtml";

type ReaderContentProps = {
  htmlContent: string;
  proseClass: string;
  textAlign?: TextAlignment;
  effectiveReadingMode: ReadingMode;
  readingDirection: ReadingDirection;
  pageWidth: number;
  columnsRef: React.RefObject<HTMLDivElement | null>;
  onContentClick: (event: React.MouseEvent<HTMLDivElement>) => void;
};

const isRawReaderContent = (html: string) =>
  html.includes('class="novelhub-raw-reader"');

export const ReaderContent: React.FC<ReaderContentProps> = React.memo(
  ({
    htmlContent,
    proseClass,
    textAlign = "original",
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

    useEffect(() => {
      const el = columnsRef.current;
      if (!el) return;

      const isPaged =
        effectiveReadingMode === "single" || effectiveReadingMode === "double";
      const images = el.querySelectorAll("img");
      images.forEach((img) => {
        const src = img.getAttribute("src");
        if (!src || src === "#" || src === "about:blank") {
          img.style.display = "none";
          const parent = img.parentElement;
          if (
            parent &&
            parent.children.length === 1 &&
            !parent.textContent?.trim()
          ) {
            parent.style.display = "none";
          }
          return;
        }

        let highestOnlyChild: HTMLElement | null = null;
        let current = img.parentElement;
        while (
          current &&
          current !== el &&
          !current.classList.contains("reader-content")
        ) {
          const text = (current.textContent || "").trim();
          const imgCount = current.querySelectorAll("img").length;
          if (imgCount === 1 && text === "") {
            highestOnlyChild = current;
          }
          current = current.parentElement;
        }

        if (highestOnlyChild) {
          let node: HTMLElement | null = img.parentElement;
          while (node && node !== highestOnlyChild.parentElement) {
            node.classList.add("reader-image-page");
            Array.from(node.childNodes).forEach((child) => {
              if (
                child.nodeType === Node.TEXT_NODE &&
                (child.textContent || "").trim() === ""
              ) {
                child.remove();
              } else if (
                child.nodeType === Node.ELEMENT_NODE &&
                (child as HTMLElement).tagName.toLowerCase() === "br"
              ) {
                child.remove();
              }
            });
            node = node.parentElement;
          }
        }

        img.style.cursor = "pointer";

        img.onerror = () => {
          img.style.display = "none";
          const parent = img.parentElement;
          if (
            parent &&
            parent.children.length === 1 &&
            !parent.textContent?.trim()
          ) {
            parent.style.display = "none";
          }
        };
      });
    }, [sanitizedHTML, columnsRef]);

    if (!htmlContent) return null;

    return (
      <div
        ref={columnsRef}
        data-align={textAlign}
        onClick={onContentClick}
        className={`reader-content ${proseClass} max-w-none w-full ${
          rawReader
            ? "flex-1 min-h-0 h-full"
            : effectiveReadingMode === "scroll"
              ? "h-auto"
              : "flex-1 min-h-0 h-full"
        } [&>body]:m-0 [&>body]:p-0 [&>body]:display-block ${
          effectiveReadingMode === "scroll"
            ? "[&_img]:max-w-full [&_img]:h-auto"
            : ""
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
  },
);
