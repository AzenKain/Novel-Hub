import type { TFunction } from "i18next";
import {
  Check,
  FileText,
  Filter,
  Image as ImageIcon,
  MessageSquare,
  Pencil,
  Search,
  Sparkles,
  Trash2,
  X,
} from "lucide-react";
import React, { useMemo, useState } from "react";

import type { Chapter, Highlight } from "@/types";
import type { ImageBookmark } from "./ReaderImageToolbar";

export const HIGHLIGHT_COLORS = [
  "#fef08a",
  "#bbf7d0",
  "#bfdbfe",
  "#fbcfe8",
  "#fed7aa",
  "#e9d5ff",
  "#fecaca",
  "#99f6e4",
];

type FilterType = "all" | "text" | "image";

type ReaderHighlightsPanelProps = {
  t: TFunction;
  highlights: Highlight[];
  imageBookmarks?: ImageBookmark[];
  chapters?: Chapter[];
  onSelect?: (highlight: Highlight) => void;
  onUpdate: (id: string, color: string, note?: string) => void;
  onDelete: (id: string) => void;
  onSelectImageBookmark?: (bm: ImageBookmark) => void;
  onDeleteImageBookmark?: (id: string) => void;
  onOpenQuoteCard?: (text?: string, imageUrl?: string) => void;
};

interface UnifiedBookmarkItem {
  id: string;
  type: "text" | "image";
  chapter_id?: string;
  chapter_title?: string;
  text_content?: string;
  image_url?: string;
  note?: string;
  color?: string;
  page_index?: number;
  rawHighlight?: Highlight;
  rawImageBookmark?: ImageBookmark;
}

export const ReaderHighlightsPanel: React.FC<ReaderHighlightsPanelProps> = ({
  t,
  highlights,
  imageBookmarks = [],
  chapters = [],
  onSelect,
  onUpdate,
  onDelete,
  onSelectImageBookmark,
  onDeleteImageBookmark,
  onOpenQuoteCard,
}) => {
  const [searchQuery, setSearchQuery] = useState("");
  const [filterType, setFilterType] = useState<FilterType>("all");
  const [selectedColor, setSelectedColor] = useState<string | null>(null);

  const [editingId, setEditingId] = useState<string | null>(null);
  const [draftNote, setDraftNote] = useState("");
  const [draftColor, setDraftColor] = useState(HIGHLIGHT_COLORS[0]);

  // Create chapters map for quick title lookup
  const chaptersMap = useMemo(() => {
    const map = new Map<string, string>();
    for (const ch of chapters) {
      map.set(ch.id, ch.title);
    }
    return map;
  }, [chapters]);

  // Merge text highlights & image bookmarks into a unified list
  const allItems: UnifiedBookmarkItem[] = useMemo(() => {
    const items: UnifiedBookmarkItem[] = [];

    for (const bm of imageBookmarks) {
      const title =
        bm.chapter_title ||
        (bm.chapter_id ? chaptersMap.get(bm.chapter_id) : undefined) ||
        (bm.page_index !== undefined
          ? `${t("reader.page", "Trang")} ${bm.page_index + 1}`
          : t("reader.illustration", "Minh họa"));
      items.push({
        id: `img-${bm.id}`,
        type: "image",
        chapter_id: bm.chapter_id,
        chapter_title: title,
        image_url: bm.image_url,
        note: bm.note,
        page_index: bm.page_index,
        rawImageBookmark: bm,
      });
    }

    for (const hl of highlights) {
      const title =
        (hl.chapter_id ? chaptersMap.get(hl.chapter_id) : undefined) ||
        t("reader.chapter", "Chương");
      items.push({
        id: `text-${hl.id}`,
        type: "text",
        chapter_id: hl.chapter_id,
        chapter_title: title,
        text_content: hl.text_content,
        note: hl.note,
        color: hl.color,
        rawHighlight: hl,
      });
    }

    return items;
  }, [imageBookmarks, highlights, chaptersMap, t]);

  // Apply local Search & Filter chips
  const filteredItems = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();

    return allItems.filter((item) => {
      // Type Filter
      if (filterType === "text" && item.type !== "text") return false;
      if (filterType === "image" && item.type !== "image") return false;

      // Color Filter
      if (selectedColor && item.type === "text") {
        const itemColor = (item.color || "").toLowerCase();
        const targetColor = selectedColor.toLowerCase();
        if (itemColor !== targetColor) return false;
      }

      // Search Query Filter (Searches in text, personal notes, or chapter title)
      if (q) {
        const textMatch = item.text_content?.toLowerCase().includes(q) ?? false;
        const noteMatch = item.note?.toLowerCase().includes(q) ?? false;
        const titleMatch = item.chapter_title?.toLowerCase().includes(q) ?? false;
        if (!textMatch && !noteMatch && !titleMatch) return false;
      }

      return true;
    });
  }, [allItems, filterType, selectedColor, searchQuery]);

  // Group filtered items by Chapter Title
  const groupedItems = useMemo(() => {
    const groups: { title: string; items: UnifiedBookmarkItem[] }[] = [];
    const map = new Map<string, UnifiedBookmarkItem[]>();

    for (const item of filteredItems) {
      const groupKey = item.chapter_title || t("reader.general", "Chung");
      if (!map.has(groupKey)) {
        map.set(groupKey, []);
      }
      map.get(groupKey)!.push(item);
    }

    for (const [title, items] of map.entries()) {
      groups.push({ title, items });
    }

    return groups;
  }, [filteredItems, t]);

  const startEditing = (highlight: Highlight) => {
    setEditingId(highlight.id);
    setDraftNote(highlight.note || "");
    setDraftColor(highlight.color || HIGHLIGHT_COLORS[0]);
  };

  const stopEditing = () => {
    setEditingId(null);
    setDraftNote("");
  };

  const saveEditing = (id: string) => {
    onUpdate(id, draftColor, draftNote.trim() || undefined);
    stopEditing();
  };

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* Search Bar & Compact Filter Toolbar */}
      <div className="p-2.5 border-b border-[var(--reader-ui-border,rgba(255,255,255,0.12))] flex flex-col gap-2 bg-[var(--reader-ui-surface-strong,rgba(30,31,41,0.5))]">
        {/* Search Input */}
        <div className="relative flex items-center">
          <Search className="w-3.5 h-3.5 absolute left-2.5 opacity-40 pointer-events-none text-[var(--reader-ui-text)]" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder={t("reader.search_highlights_placeholder", "Tìm kiếm đánh dấu, ghi chú...")}
            className="input input-xs h-7 pl-7 pr-6 w-full rounded-lg bg-[var(--reader-ui-soft,rgba(255,255,255,0.06))] border-[var(--reader-ui-border,rgba(255,255,255,0.12))] text-xs text-[var(--reader-ui-text)] focus:border-[var(--reader-ui-accent)] focus:outline-hidden"
          />
          {searchQuery && (
            <button
              type="button"
              onClick={() => setSearchQuery("")}
              className="absolute right-2 opacity-50 hover:opacity-100 text-[var(--reader-ui-text)] cursor-pointer"
            >
              <X className="w-3 h-3" />
            </button>
          )}
        </div>

        {/* 3-Column Compact Segmented Filter (Tất cả | Chữ | Ảnh) */}
        <div className="grid grid-cols-3 gap-1 p-0.5 bg-[var(--reader-ui-soft,rgba(255,255,255,0.05))] border border-[var(--reader-ui-border,rgba(255,255,255,0.1))] rounded-lg">
          <button
            type="button"
            onClick={() => setFilterType("all")}
            className={`h-6 rounded-md text-[11px] font-semibold transition-all flex items-center justify-center gap-1 cursor-pointer ${
              filterType === "all"
                ? "bg-[var(--reader-ui-accent,#38bdf8)] text-[var(--reader-ui-accent-text,#08111d)] shadow-xs"
                : "text-[var(--reader-ui-text)] opacity-70 hover:opacity-100 hover:bg-[var(--reader-ui-hover)]"
            }`}
          >
            <span>{t("reader.filter_all", "Tất cả")}</span>
          </button>

          <button
            type="button"
            onClick={() => setFilterType("text")}
            className={`h-6 rounded-md text-[11px] font-semibold transition-all flex items-center justify-center gap-1 cursor-pointer ${
              filterType === "text"
                ? "bg-[var(--reader-ui-accent,#38bdf8)] text-[var(--reader-ui-accent-text,#08111d)] shadow-xs"
                : "text-[var(--reader-ui-text)] opacity-70 hover:opacity-100 hover:bg-[var(--reader-ui-hover)]"
            }`}
          >
            <FileText className="w-3 h-3 shrink-0" />
            <span>{t("reader.filter_text", "Chữ")}</span>
          </button>

          <button
            type="button"
            onClick={() => setFilterType("image")}
            className={`h-6 rounded-md text-[11px] font-semibold transition-all flex items-center justify-center gap-1 cursor-pointer ${
              filterType === "image"
                ? "bg-[var(--reader-ui-accent,#38bdf8)] text-[var(--reader-ui-accent-text,#08111d)] shadow-xs"
                : "text-[var(--reader-ui-text)] opacity-70 hover:opacity-100 hover:bg-[var(--reader-ui-hover)]"
            }`}
          >
            <ImageIcon className="w-3 h-3 shrink-0" />
            <span>{t("reader.filter_images", "Ảnh")}</span>
          </button>
        </div>

        {/* Color Filter Dots Row (Compact) */}
        {filterType !== "image" && (
          <div className="flex items-center justify-between px-0.5 pt-0.5">
            <span className="text-[10px] uppercase font-bold opacity-50 text-[var(--reader-ui-text)]">
              {t("reader.color", "Màu")}:
            </span>
            <div className="flex items-center gap-1.5">
              {HIGHLIGHT_COLORS.map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => setSelectedColor(selectedColor === c ? null : c)}
                  style={{ backgroundColor: c }}
                  className={`w-3.5 h-3.5 rounded-full border transition-all cursor-pointer ${
                    selectedColor === c
                      ? "ring-2 ring-[var(--reader-ui-accent,#38bdf8)] ring-offset-1 ring-offset-[var(--reader-ui-surface,#13141b)] scale-125 border-current shadow-xs"
                      : "border-black/25 opacity-75 hover:opacity-100 hover:scale-110"
                  }`}
                  title={c}
                  aria-label={c}
                />
              ))}
              {selectedColor && (
                <button
                  type="button"
                  onClick={() => setSelectedColor(null)}
                  className="btn btn-ghost btn-circle btn-xs h-3.5 w-3.5 min-h-0 text-[10px] opacity-60 hover:opacity-100"
                  title={t("reader.clear_color_filter", "Xóa lọc màu")}
                >
                  ✕
                </button>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Main List Body */}
      <div className="flex-1 min-h-0 overflow-y-auto p-3 space-y-4">
        {allItems.length === 0 ? (
          <div className="py-12 text-center space-y-2 opacity-60">
            <Filter className="w-8 h-8 mx-auto opacity-30 text-[var(--reader-ui-accent,#38bdf8)]" />
            <p className="text-xs font-medium">
              {t("reader.no_highlights", "Chưa có đoạn đánh dấu nào.")}
            </p>
          </div>
        ) : filteredItems.length === 0 ? (
          <div className="py-12 text-center space-y-2 opacity-60">
            <Search className="w-8 h-8 mx-auto opacity-30 text-[var(--reader-ui-text)]" />
            <p className="text-xs font-medium">
              {t("reader.no_matching_highlights", "Không tìm thấy đánh dấu phù hợp")}
            </p>
          </div>
        ) : (
          groupedItems.map((group) => (
            <div key={group.title} className="space-y-2">
              {/* Group Header */}
              <div className="flex items-center justify-between px-1 text-[11px] font-bold uppercase tracking-wider text-[var(--reader-ui-text)] opacity-60">
                <span className="truncate max-w-[210px]">{group.title}</span>
                <span className="badge badge-xs font-mono font-bold bg-[var(--reader-ui-soft,rgba(255,255,255,0.1))] text-[var(--reader-ui-text)]">
                  {group.items.length}
                </span>
              </div>

              {/* Group Items */}
              <ul className="flex flex-col gap-2">
                {group.items.map((item) => {
                  if (item.type === "image" && item.rawImageBookmark) {
                    const bm = item.rawImageBookmark;
                    return (
                      <li
                        key={item.id}
                        className="flex items-center justify-between gap-2 rounded-xl border border-[var(--reader-ui-border,rgba(255,255,255,0.12))] bg-[var(--reader-ui-soft,rgba(255,255,255,0.06))] p-2.5 hover:border-[var(--reader-ui-accent,#38bdf8)]/50 transition-colors shadow-2xs"
                      >
                        <button
                          type="button"
                          onClick={() => onSelectImageBookmark?.(bm)}
                          className="flex flex-1 items-start gap-2.5 text-left min-w-0 group cursor-pointer"
                        >
                          <img
                            src={bm.image_url}
                            alt="Thumbnail"
                            className="h-12 w-12 rounded-lg object-cover bg-base-300 shrink-0 shadow-xs border border-[var(--reader-ui-border,rgba(255,255,255,0.1))]"
                          />
                          <div className="min-w-0 flex-1">
                            <p className="text-xs font-bold text-[var(--reader-ui-text)] truncate group-hover:text-[var(--reader-ui-accent,#38bdf8)] transition-colors">
                              {item.chapter_title}
                            </p>
                            {bm.note && (
                              <p className="text-xs italic opacity-75 line-clamp-2 mt-0.5 text-[var(--reader-ui-text)]">
                                {bm.note}
                              </p>
                            )}
                            {bm.page_index !== undefined && (
                              <p className="text-[10px] opacity-50 mt-0.5 font-mono text-[var(--reader-ui-text)]">
                                {t("reader.page", "Trang")} {bm.page_index + 1}
                              </p>
                            )}
                          </div>
                        </button>
                        <div className="flex items-center gap-1 shrink-0">
                          {onOpenQuoteCard && (
                            <div className="tooltip tooltip-top" data-tip={t("reader.quote_card", "Tạo ảnh trích dẫn")}>
                              <button
                                type="button"
                                onClick={() =>
                                  onOpenQuoteCard(
                                    bm.note?.trim() || undefined,
                                    bm.image_url
                                  )
                                }
                                className="btn btn-ghost btn-xs text-amber-500 hover:text-amber-600"
                                aria-label={t("reader.quote_card", "Tạo ảnh trích dẫn")}
                              >
                                <Sparkles className="h-3.5 w-3.5" />
                              </button>
                            </div>
                          )}
                          {onDeleteImageBookmark && (
                            <div className="tooltip tooltip-top" data-tip={t("common.delete", "Xóa")}>
                              <button
                                type="button"
                                onClick={() => onDeleteImageBookmark(bm.id)}
                                className="btn btn-ghost btn-xs text-error hover:bg-error/20"
                                aria-label={t("common.delete", "Xóa")}
                              >
                                <Trash2 className="h-3.5 w-3.5" />
                              </button>
                            </div>
                          )}
                        </div>
                      </li>
                    );
                  }

                  if (item.type === "text" && item.rawHighlight) {
                    const highlight = item.rawHighlight;
                    const isEditing = editingId === highlight.id;

                    return (
                      <li
                        key={item.id}
                        className="rounded-xl border border-[var(--reader-ui-border,rgba(255,255,255,0.12))] bg-[var(--reader-ui-soft,rgba(255,255,255,0.06))] p-3 shadow-2xs hover:border-[var(--reader-ui-accent,#38bdf8)]/50 transition-colors"
                      >
                        <div className="flex items-start gap-2">
                          <span
                            className="mt-1 h-3 w-3 shrink-0 rounded-full border border-current/20 shadow-2xs"
                            style={{ backgroundColor: highlight.color }}
                            aria-hidden="true"
                          />
                          <button
                            type="button"
                            onClick={() => onSelect?.(highlight)}
                            className="line-clamp-3 flex-1 text-left text-xs leading-snug hover:underline text-[var(--reader-ui-text)] cursor-pointer"
                          >
                            {highlight.text_content}
                          </button>
                        </div>

                        {isEditing ? (
                          <div className="mt-2.5 flex flex-col gap-2 border-t border-[var(--reader-ui-border,rgba(255,255,255,0.1))] pt-2">
                            <div className="flex items-center gap-1.5">
                              {HIGHLIGHT_COLORS.map((color) => (
                                <button
                                  key={color}
                                  type="button"
                                  onClick={() => setDraftColor(color)}
                                  style={{ backgroundColor: color }}
                                  className={`h-4 w-4 rounded-full border transition-transform hover:scale-125 cursor-pointer ${
                                    draftColor === color
                                      ? "border-current ring-2 ring-[var(--reader-ui-accent,#38bdf8)] ring-offset-1"
                                      : "border-current/20"
                                  }`}
                                  aria-label={t("reader.highlight_color", "Highlight color")}
                                />
                              ))}
                            </div>
                            <textarea
                              rows={2}
                              value={draftNote}
                              onChange={(e) => setDraftNote(e.target.value)}
                              onKeyDown={(e) => {
                                if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
                                  e.preventDefault();
                                  saveEditing(highlight.id);
                                }
                              }}
                              placeholder={t("reader.add_note_placeholder", "Add a note...")}
                              className="textarea textarea-bordered textarea-xs w-full rounded-lg text-xs bg-[var(--reader-ui-soft)] border-[var(--reader-ui-border)] text-[var(--reader-ui-text)]"
                            />
                            <div className="flex justify-end gap-1">
                              <button
                                type="button"
                                onClick={stopEditing}
                                className="btn btn-ghost btn-xs"
                                aria-label={t("common.cancel", "Cancel")}
                              >
                                <X className="h-3.5 w-3.5" />
                              </button>
                              <button
                                type="button"
                                onClick={() => saveEditing(highlight.id)}
                                className="btn btn-xs bg-[var(--reader-ui-accent,#38bdf8)] text-[var(--reader-ui-accent-text,#08111d)] border-0"
                                aria-label={t("common.save", "Save")}
                              >
                                <Check className="h-3.5 w-3.5" />
                              </button>
                            </div>
                          </div>
                        ) : (
                          <>
                            {highlight.note && (
                              <p
                                onClick={() => onSelect?.(highlight)}
                                className="mt-1.5 text-xs italic opacity-80 hover:opacity-100 cursor-pointer hover:underline border-l-2 border-[var(--reader-ui-accent,#38bdf8)]/50 pl-2 text-[var(--reader-ui-text)]"
                                title={t("reader.jump_to_highlight", "Jump to highlight")}
                              >
                                {highlight.note}
                              </p>
                            )}
                            <div className="mt-2 flex justify-end gap-1 border-t border-[var(--reader-ui-border,rgba(255,255,255,0.06))] pt-1">
                              {onOpenQuoteCard && (
                                <div className="tooltip tooltip-top" data-tip={t("reader.quote_card", "Tạo ảnh trích dẫn")}>
                                  <button
                                    type="button"
                                    onClick={() => onOpenQuoteCard(highlight.text_content, undefined)}
                                    className="btn btn-ghost btn-xs text-amber-500 hover:text-amber-600"
                                    aria-label={t("reader.quote_card", "Tạo ảnh trích dẫn")}
                                  >
                                    <Sparkles className="h-3.5 w-3.5" />
                                  </button>
                                </div>
                              )}
                              <div className="tooltip tooltip-top" data-tip={t("reader.edit_highlight", "Sửa ghi chú")}>
                                <button
                                  type="button"
                                  onClick={() => startEditing(highlight)}
                                  className="btn btn-ghost btn-xs text-[var(--reader-ui-text)] opacity-70 hover:opacity-100"
                                  aria-label={t("reader.edit_highlight", "Edit note")}
                                >
                                  <Pencil className="h-3.5 w-3.5" />
                                </button>
                              </div>
                              <div className="tooltip tooltip-top" data-tip={t("common.delete", "Xóa")}>
                                <button
                                  type="button"
                                  onClick={() => onDelete(highlight.id)}
                                  className="btn btn-ghost btn-xs text-error hover:bg-error/20"
                                  aria-label={t("common.delete", "Delete")}
                                >
                                  <Trash2 className="h-3.5 w-3.5" />
                                </button>
                              </div>
                            </div>
                          </>
                        )}
                      </li>
                    );
                  }

                  return null;
                })}
              </ul>
            </div>
          ))
        )}
      </div>
    </div>
  );
};
