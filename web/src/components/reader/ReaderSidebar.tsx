import type { TFunction } from "i18next";
import { ArrowLeft, BookOpen, FileText, PanelLeftClose } from "lucide-react";
import React from "react";

import type { Book, Chapter, Highlight } from "@/types";
import { ReaderHighlightsPanel } from "./ReaderHighlightsPanel";

type ReaderSidebarProps = {
  t: TFunction;
  book: Book;
  chapters: Chapter[];
  currentChapter: Chapter | null;
  sidebarBg: string;
  sidebarRef: React.RefObject<HTMLElement | null>;
  onClose: () => void;
  onBack: () => void;
  onSelectChapter: (chapter: Chapter) => void;
  highlights?: Highlight[];
  onUpdateHighlight?: (id: string, color: string, note?: string) => void;
  onDeleteHighlight?: (id: string) => void;
};

function getSidebarEntryKind(title: string) {
  const normalized = title.trim().toLowerCase();
  if (/^(quy[eê]n|volume|part)\b/.test(normalized)) return "section";
  if (/^(l[ờo]i d[ẫa]n|prologue|đoạn kết|epilogue|cover)\b/.test(normalized)) {
    return "special";
  }
  return "chapter";
}

function formatChapterTitle(chapter: Chapter, t: TFunction) {
  let displayTitle =
    chapter.title || `${t("reader.chapter", "Chapter")} ${chapter.chapter_index + 1}`;
  if (displayTitle.match(/\.(x)?html$/i)) {
    displayTitle = displayTitle
      .replace(/\.(x)?html$/i, "")
      .replace(/[-_]/g, " ");
    displayTitle = displayTitle.replace(/\b\w/g, (letter) =>
      letter.toUpperCase(),
    );
  }
  return displayTitle;
}

export const ReaderSidebar: React.FC<ReaderSidebarProps> = ({
  t,
  book,
  chapters,
  currentChapter,
  sidebarBg,
  sidebarRef,
  onClose,
  onBack,
  onSelectChapter,
  highlights,
  onUpdateHighlight,
  onDeleteHighlight,
}) => {
  const singleChapter = chapters.length <= 1;
  const showHighlights = Boolean(highlights && onUpdateHighlight && onDeleteHighlight);

  return (
    <div className="drawer-side z-50">
      <label
        htmlFor="reader-drawer"
        aria-label={t("reader.close_toc", "Close table of contents")}
        className="drawer-overlay"
      />
      <aside
        ref={sidebarRef}
        className={`reader-sidebar ${singleChapter ? "reader-sidebar-single" : ""} grid h-full max-h-screen min-h-0 w-80 grid-rows-[auto,minmax(0,1fr),auto] overflow-hidden border-r shadow-2xl transition-colors duration-300 ${sidebarBg}`}
      >
        <div className="flex items-center gap-3 border-b border-base-content/10 p-4">
          <button
            type="button"
            onClick={onClose}
            className="reader-sidebar-icon-btn"
            title={t("reader.close_toc", "Close table of contents")}
            aria-label={t("reader.close_toc", "Close table of contents")}
          >
            <PanelLeftClose className="h-5 w-5" />
          </button>
          <h2 className="line-clamp-2 flex-1 text-sm font-bold leading-tight">
            {book.title}
          </h2>
        </div>

        <div className="reader-sidebar-body">
          <div className="reader-sidebar-list-title">
            <span>{t("reader.toc", "Table of Contents")}</span>
            {!singleChapter && (
              <span className="reader-sidebar-count">{chapters.length}</span>
            )}
          </div>

          {singleChapter ? (
            <div className="px-4">
              <button
                onClick={() => chapters[0] && onSelectChapter(chapters[0])}
                className="group flex w-full min-w-0 items-center gap-3 rounded-2xl border border-[var(--reader-ui-accent)]/40 bg-[var(--reader-ui-accent-soft)] px-4 py-4 text-left text-[var(--reader-ui-accent)] shadow-sm"
              >
                <span className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-[var(--reader-ui-accent)]/15">
                  {getSidebarEntryKind(
                    chapters[0] ? formatChapterTitle(chapters[0], t) : "",
                  ) === "special" ? (
                    <BookOpen className="h-5 w-5" />
                  ) : (
                    <FileText className="h-5 w-5" />
                  )}
                </span>
                <span className="min-w-0">
                  <span className="block text-xs font-black uppercase tracking-wider opacity-70">
                    {t("reader.single_entry", "Current file")}
                  </span>
                  <span className="mt-0.5 block truncate font-bold text-[var(--reader-ui-text)]">
                    {chapters[0]
                      ? formatChapterTitle(chapters[0], t)
                      : book.title}
                  </span>
                </span>
              </button>
            </div>
          ) : (
            <ul className="reader-sidebar-list">
              {chapters.map((chapter) => {
                const displayTitle = formatChapterTitle(chapter, t);
                const entryKind = getSidebarEntryKind(displayTitle);
                const itemClassName =
                  entryKind === "section"
                    ? "reader-sidebar-item reader-sidebar-item-section"
                    : entryKind === "special"
                      ? "reader-sidebar-item reader-sidebar-item-special"
                      : "reader-sidebar-item reader-sidebar-item-chapter";
                const activeClassName =
                  currentChapter?.id === chapter.id
                    ? "reader-list-item-active"
                    : "reader-list-item opacity-80 hover:opacity-100";

                return (
                  <li
                    key={chapter.id}
                    className={`reader-sidebar-list-row reader-sidebar-list-row-${entryKind}`}
                  >
                    <button
                      onClick={() => onSelectChapter(chapter)}
                      className={`${itemClassName} ${activeClassName}`}
                    >
                      {entryKind === "chapter" && (
                        <span
                          className="reader-sidebar-index"
                          aria-hidden="true"
                        >
                          {String(chapter.chapter_index + 1).padStart(2, "0")}
                        </span>
                      )}
                      <span className="line-clamp-2 text-sm leading-tight">
                        {displayTitle}
                      </span>
                    </button>
                  </li>
                );
              })}
            </ul>
          )}

          {showHighlights && (
            <div className="mt-4 border-t border-base-content/10 pt-2">
              <ReaderHighlightsPanel
                t={t}
                highlights={highlights || []}
                onUpdate={onUpdateHighlight!}
                onDelete={onDeleteHighlight!}
              />
            </div>
          )}
        </div>

        <div className="reader-sidebar-footer">
          <button
            type="button"
            onClick={onBack}
            className="reader-sidebar-back-btn"
          >
            <ArrowLeft className="h-4 w-4" />
            <span>{t("reader.back_to_previous", "Back")}</span>
          </button>
        </div>
      </aside>
    </div>
  );
};
