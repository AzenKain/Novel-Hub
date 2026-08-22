import type { TFunction } from "i18next";
import { ArrowLeft, Bookmark, BookOpen, ChevronRight, FileText, ListTree, PanelLeftClose, Trash2 } from "lucide-react";
import React, { useState } from "react";

import type { Book, Chapter, Highlight } from "@/types";
import type { AudioBookmark } from "./AudioPlayer";
import type { ImageBookmark } from "./ReaderImageToolbar";
import { ReaderHighlightsPanel } from "./ReaderHighlightsPanel";
import { getMediaUrl } from "@/config/api";

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
  onSelectHighlight?: (highlight: Highlight) => void;
  onOpenQuoteCard?: (text?: string, imageUrl?: string) => void;
  nextInSeries?: any;
  nextInReadList?: Book | null;
  onGoToNextInSeries?: () => void;
  onGoToNextInReadList?: () => void;
  isAudio?: boolean;
  audioBookmarks?: AudioBookmark[];
  onSelectAudioBookmark?: (time_sec: number) => void;
  onDeleteAudioBookmark?: (id: string) => void;
  isVisualContent?: boolean;
  imageBookmarks?: ImageBookmark[];
  onSelectImageBookmark?: (bookmark: ImageBookmark) => void;
  onDeleteImageBookmark?: (id: string) => void;
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

function formatAudioTime(seconds: number): string {
  const mins = Math.floor(seconds / 60);
  const secs = Math.floor(seconds % 60);
  return `${mins}:${secs.toString().padStart(2, "0")}`;
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
  onSelectHighlight,
  onOpenQuoteCard,
  nextInSeries,
  nextInReadList,
  onGoToNextInSeries,
  onGoToNextInReadList,
  isAudio = false,
  audioBookmarks = [],
  onSelectAudioBookmark,
  onDeleteAudioBookmark,
  isVisualContent = false,
  imageBookmarks = [],
  onSelectImageBookmark,
  onDeleteImageBookmark,
}) => {
  const [activeTab, setActiveTab] = useState<"toc" | "highlights">("toc");
  const singleChapter = chapters.length === 1;
  const totalBookmarksCount =
    (highlights?.length || 0) + (imageBookmarks?.length || 0) + (audioBookmarks?.length || 0);

  const renderNextVolumeCard = () => {
    if (nextInReadList) {
      return (
        <div className="mt-6 p-4 rounded-2xl border border-base-content/10 bg-base-content/5">
          <p className="text-[10px] font-black uppercase tracking-wider opacity-60 mb-2.5">
            {t("reader.next_in_read_list", "Tập tiếp theo trong danh sách")}
          </p>
          <div className="flex items-center gap-3">
            <img
              src={
                (nextInReadList as any).cover_path
                  ? getMediaUrl((nextInReadList as any).cover_path)
                  : nextInReadList.cover_url || "/placeholder.png"
              }
              alt={nextInReadList.title}
              className="h-14 aspect-[3/4.2] rounded-lg object-cover bg-base-200 shrink-0 shadow-xs"
            />
            <div className="min-w-0 flex-1">
              <p className="text-xs font-bold truncate text-[var(--reader-ui-text)]">
                {nextInReadList.title}
              </p>
              {nextInReadList.description && (
                <p className="text-[10px] opacity-60 truncate mt-0.5">
                  {nextInReadList.description}
                </p>
              )}
            </div>
          </div>
          <button
            onClick={onGoToNextInReadList}
            className="btn btn-primary btn-xs w-full gap-1 rounded-lg mt-3 text-[10px] font-bold"
          >
            <BookOpen className="w-3 h-3" />
            {t("reader.read_next_in_list", "Đọc tập tiếp theo")}
            <ChevronRight className="w-3 h-3" />
          </button>
        </div>
      );
    }

    if (nextInSeries) {
      return (
        <div className="mt-6 p-4 rounded-2xl border border-base-content/10 bg-base-content/5">
          <p className="text-[10px] font-black uppercase tracking-wider opacity-60 mb-2.5">
            {t("reader.next_in_series", "Tập tiếp theo trong sê-ri")}
          </p>
          <div className="flex items-center gap-3">
            <img
              src={
                nextInSeries.cover_path
                  ? getMediaUrl(nextInSeries.cover_path)
                  : "/placeholder.png"
              }
              alt={nextInSeries.title}
              className="h-14 aspect-[3/4.2] rounded-lg object-cover bg-base-200 shrink-0 shadow-xs"
            />
            <div className="min-w-0 flex-1">
              <p className="text-xs font-bold truncate text-[var(--reader-ui-text)]">
                {nextInSeries.title}
              </p>
              {nextInSeries.series && (
                <p className="text-[10px] opacity-60 truncate mt-0.5">
                  {nextInSeries.series} #{nextInSeries.series_index || 1}
                </p>
              )}
            </div>
          </div>
          <button
            onClick={onGoToNextInSeries}
            className="btn btn-primary btn-xs w-full gap-1 rounded-lg mt-3 text-[10px] font-bold"
          >
            <BookOpen className="w-3 h-3" />
            {t("reader.read_next_volume", "Đọc tập tiếp theo")}
            <ChevronRight className="w-3 h-3" />
          </button>
        </div>
      );
    }

    return null;
  };

  return (
    <div className="drawer-side z-50">
      <label
        htmlFor="reader-drawer"
        aria-label={t("reader.close_toc", "Close table of contents")}
        className="drawer-overlay"
      />
      <aside
        ref={sidebarRef}
        className={`reader-sidebar ${singleChapter ? "reader-sidebar-single" : ""} flex flex-col h-full max-h-screen min-h-0 w-80 overflow-hidden border-r shadow-2xl transition-colors duration-300 ${sidebarBg}`}
      >
        {/* Sidebar Header */}
        <div className="flex items-center gap-3 border-b border-base-content/10 p-4 shrink-0">
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

        {/* Top Tab Bar */}
        <div className="flex border-b border-[var(--reader-ui-border,rgba(255,255,255,0.12))] bg-[var(--reader-ui-soft,rgba(0,0,0,0.04))] px-3 pt-2 shrink-0">
          <button
            type="button"
            onClick={() => setActiveTab("toc")}
            className={`flex items-center gap-1.5 pb-2.5 px-3 text-xs font-bold border-b-2 transition-all cursor-pointer ${
              activeTab === "toc"
                ? "border-[var(--reader-ui-accent,#38bdf8)] text-[var(--reader-ui-accent,#38bdf8)]"
                : "border-transparent text-[var(--reader-ui-text)] opacity-60 hover:opacity-100"
            }`}
          >
            <ListTree className="w-3.5 h-3.5" />
            <span>{t("reader.toc_tab", "Mục lục")}</span>
            {!singleChapter && chapters.length > 0 && (
              <span
                className={`badge badge-xs font-mono font-bold ml-0.5 ${
                  activeTab === "toc"
                    ? "bg-[var(--reader-ui-accent,#38bdf8)] text-[var(--reader-ui-accent-text,#08111d)]"
                    : "bg-[var(--reader-ui-soft)] text-[var(--reader-ui-text)] border border-[var(--reader-ui-border)]"
                }`}
              >
                {chapters.length}
              </span>
            )}
          </button>

          <button
            type="button"
            onClick={() => setActiveTab("highlights")}
            className={`flex items-center gap-1.5 pb-2.5 px-3 text-xs font-bold border-b-2 transition-all cursor-pointer ${
              activeTab === "highlights"
                ? "border-[var(--reader-ui-accent,#38bdf8)] text-[var(--reader-ui-accent,#38bdf8)]"
                : "border-transparent text-[var(--reader-ui-text)] opacity-60 hover:opacity-100"
            }`}
          >
            <Bookmark className="w-3.5 h-3.5" />
            <span>{isAudio ? t("reader.audio_bookmarks", "Dấu trang Audio") : t("reader.highlights_tab", "Đánh dấu")}</span>
            {totalBookmarksCount > 0 && (
              <span
                className={`badge badge-xs font-mono font-bold ml-0.5 ${
                  activeTab === "highlights"
                    ? "bg-[var(--reader-ui-accent,#38bdf8)] text-[var(--reader-ui-accent-text,#08111d)]"
                    : "bg-[var(--reader-ui-soft)] text-[var(--reader-ui-text)] border border-[var(--reader-ui-border)]"
                }`}
              >
                {totalBookmarksCount}
              </span>
            )}
          </button>
        </div>

        {/* Tab Content Body */}
        <div className="flex-1 min-h-0 overflow-hidden flex flex-col">
          {activeTab === "toc" ? (
            <div className="flex-1 min-h-0 overflow-y-auto reader-sidebar-body">
              {singleChapter ? (
                <div className="px-4 pb-4 flex-1">
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
                  {renderNextVolumeCard()}
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
                  {(nextInReadList || nextInSeries) && (
                    <li className="px-2 pb-4">
                      {renderNextVolumeCard()}
                    </li>
                  )}
                </ul>
              )}
            </div>
          ) : (
            <div className="flex-1 min-h-0 flex flex-col">
              {isAudio ? (
                <div className="flex-1 min-h-0 overflow-y-auto p-4">
                  {!audioBookmarks || audioBookmarks.length === 0 ? (
                    <div className="py-12 text-center space-y-1.5 opacity-60">
                      <Bookmark className="w-6 h-6 mx-auto opacity-40 text-primary" />
                      <p className="text-xs">
                        {t("reader.no_bookmarks", "Chưa có mốc thời gian nào được lưu.")}
                      </p>
                    </div>
                  ) : (
                    <ul className="flex flex-col gap-2">
                      {audioBookmarks.map((bm) => (
                        <li
                          key={bm.id}
                          className="flex items-center justify-between gap-2 rounded-xl border border-(--reader-ui-border) bg-(--reader-ui-soft) p-2.5 hover:border-primary/40 transition-colors"
                        >
                          <button
                            type="button"
                            onClick={() => {
                              onSelectAudioBookmark?.(bm.time_sec);
                              onClose();
                            }}
                            className="flex flex-1 items-start gap-2 text-left min-w-0 group"
                          >
                            <span className="px-2 py-0.5 rounded-md bg-primary/15 text-primary font-mono text-[11px] font-bold shrink-0">
                              {formatAudioTime(bm.time_sec)}
                            </span>
                            <div className="min-w-0 flex-1">
                              <p className="text-xs font-semibold text-[var(--reader-ui-text)] line-clamp-2 group-hover:text-primary transition-colors">
                                {bm.note || bm.chapter_title || t("reader.bookmark", "Dấu trang")}
                              </p>
                              {bm.note && bm.chapter_title && (
                                <p className="text-[10px] opacity-50 truncate mt-0.5">
                                  {bm.chapter_title}
                                </p>
                              )}
                            </div>
                          </button>
                          {onDeleteAudioBookmark && (
                            <button
                              type="button"
                              onClick={() => onDeleteAudioBookmark(bm.id)}
                              className="btn btn-ghost btn-xs text-error hover:bg-error/20"
                              title={t("common.delete", "Xóa")}
                              aria-label={t("common.delete", "Xóa")}
                            >
                              <Trash2 className="h-3.5 w-3.5" />
                            </button>
                          )}
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              ) : (
                <ReaderHighlightsPanel
                  t={t}
                  highlights={highlights || []}
                  imageBookmarks={imageBookmarks || []}
                  chapters={chapters}
                  onUpdate={onUpdateHighlight!}
                  onDelete={onDeleteHighlight!}
                  onSelect={(hl) => {
                    onSelectHighlight?.(hl);
                    onClose();
                  }}
                  onSelectImageBookmark={(bm) => {
                    onSelectImageBookmark?.(bm);
                    onClose();
                  }}
                  onDeleteImageBookmark={onDeleteImageBookmark}
                  onOpenQuoteCard={onOpenQuoteCard}
                />
              )}
            </div>
          )}
        </div>

        {/* Sidebar Footer */}
        <div className="reader-sidebar-footer shrink-0 border-t border-base-content/10">
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
