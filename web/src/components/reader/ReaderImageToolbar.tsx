import type { TFunction } from "i18next";
import { BookmarkPlus, Check, Copy, Loader2, MessageSquarePlus, Sparkles, X } from "lucide-react";
import React, { useState, useLayoutEffect, useRef } from "react";
import { toast } from "react-toastify";
import { copyImageToClipboard } from "@/utils/clipboard";
import type { ActiveImageTarget, ImageBookmark } from "@/types";

interface ReaderImageToolbarProps {
  t: TFunction;
  target: ActiveImageTarget | null;
  onClose: () => void;
  onSaveBookmark: (bookmark: Omit<ImageBookmark, "id" | "created_at">) => void;
  onOpenQuoteCard?: (text?: string, imageUrl?: string) => void;
}

export const ReaderImageToolbar: React.FC<ReaderImageToolbarProps> = ({
  t,
  target,
  onClose,
  onSaveBookmark,
  onOpenQuoteCard,
}) => {
  const [note, setNote] = useState("");
  const [copied, setCopied] = useState(false);
  const [copying, setCopying] = useState(false);
  const toolbarRef = useRef<HTMLDivElement>(null);
  const [adjustedPos, setAdjustedPos] = useState<{ top: number; left: number } | null>(null);

  useLayoutEffect(() => {
    if (!target) return;
    const el = toolbarRef.current;
    const width = el?.offsetWidth || 280;
    const height = el?.offsetHeight || 100;
    const pad = 12;

    const targetX = target.x ?? (window.innerWidth / 2);
    const targetY = target.y ?? (window.innerHeight / 2);

    let left = targetX - width / 2;
    let top = targetY - height - 12;

    if (top < 64) {
      top = targetY + 24;
    }
    if (top + height > window.innerHeight - pad) {
      top = Math.max(64, window.innerHeight - height - pad);
    }
    if (left < pad) left = pad;
    if (left + width > window.innerWidth - pad) {
      left = Math.max(pad, window.innerWidth - width - pad);
    }

    setAdjustedPos({ top, left });
  }, [target]);

  if (!target) return null;

  const handleCopyImage = async (e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
    setCopying(true);
    const success = await copyImageToClipboard(target.image_url);
    setCopying(false);
    if (success) {
      setCopied(true);
      toast.success(t("reader.image_copied", "Image copied to clipboard!"));
      setTimeout(() => setCopied(false), 2000);
    } else {
      toast.error(t("reader.image_copy_failed", "Failed to copy image"));
    }
  };

  const handleSave = () => {
    onSaveBookmark({
      image_url: target.image_url,
      chapter_id: target.chapter_id,
      chapter_title: target.chapter_title,
      page_index: target.page_index,
      note: note.trim() || undefined,
    });
    setNote("");
    onClose();
  };

  const targetX = target.x ?? (window.innerWidth / 2);
  const targetY = target.y ?? (window.innerHeight / 2);
  const top = adjustedPos?.top ?? Math.max(70, targetY - 100);
  const left = adjustedPos?.left ?? Math.max(16, targetX - 140);

  return (
    <div
      ref={toolbarRef}
      data-reader-image-toolbar="true"
      data-reader-toolbar="true"
      onClick={(e) => e.stopPropagation()}
      onMouseDown={(e) => e.stopPropagation()}
      className="fixed z-50 flex flex-col gap-2 rounded-2xl border border-(--reader-ui-border) bg-(--reader-ui-surface-strong) p-2 shadow-2xl backdrop-blur-md animate-in fade-in zoom-in-95 duration-100 max-w-[calc(100vw-32px)] w-72 sm:w-80"
      style={{ top: `${top}px`, left: `${left}px` }}
    >
      {/* Top Row: Actions (Single Line, 3 Equal Buttons + Close Button) */}
      <div className="flex items-center gap-1.5 w-full">
        {/* Bookmark / Save Button */}
        <button
          type="button"
          onClick={handleSave}
          className="btn btn-ghost btn-xs h-7 flex-1 min-w-0 px-2 rounded-xl border border-(--reader-ui-border) bg-(--reader-ui-soft) text-(--reader-ui-text) hover:bg-(--reader-ui-hover) gap-1 text-[11px] font-medium transition-colors"
          title={t("reader.bookmark_image", "Bookmark this image")}
        >
          <BookmarkPlus className="h-3.5 w-3.5 text-primary shrink-0" />
          <span className="truncate">{t("reader.bookmark", "Bookmark")}</span>
        </button>

        {/* Copy Image Button */}
        <button
          type="button"
          onClick={handleCopyImage}
          disabled={copying}
          className="btn btn-ghost btn-xs h-7 flex-1 min-w-0 px-2 rounded-xl border border-(--reader-ui-border) bg-(--reader-ui-soft) text-(--reader-ui-text) hover:bg-(--reader-ui-hover) gap-1 text-[11px] font-medium transition-colors"
          title={t("reader.copy_image", "Copy Image")}
        >
          {copying ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin shrink-0" />
          ) : copied ? (
            <Check className="h-3.5 w-3.5 text-success shrink-0" />
          ) : (
            <Copy className="h-3.5 w-3.5 text-amber-500 shrink-0" />
          )}
          <span className="truncate">{copied ? t("common.copied", "Copied") : t("common.copy", "Copy")}</span>
        </button>

        {/* Quote Card Button */}
        {onOpenQuoteCard && (
          <button
            type="button"
            onClick={() => {
              onOpenQuoteCard(note.trim() || undefined, target.image_url);
              onClose();
            }}
            className="btn btn-ghost btn-xs h-7 flex-1 min-w-0 px-2 rounded-xl border border-(--reader-ui-border) bg-(--reader-ui-soft) text-(--reader-ui-text) hover:bg-(--reader-ui-hover) gap-1 text-[11px] font-medium transition-colors"
            title={t("reader.quote_card", "Create quote image")}
          >
            <Sparkles className="h-3.5 w-3.5 text-purple-400 shrink-0" />
            <span className="truncate">{t("reader.quote", "Quote")}</span>
          </button>
        )}

        {/* Close Button */}
        <button
          type="button"
          onClick={onClose}
          className="btn btn-ghost btn-circle btn-xs text-(--reader-ui-text) opacity-50 hover:opacity-100 shrink-0"
          title={t("common.close", "Close")}
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>

      {/* Bottom Row: Inline Note Input */}
      <div className="w-full pt-1 border-t border-(--reader-ui-border)/60">
        <div className="relative flex items-start">
          <MessageSquarePlus className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-(--reader-ui-muted) pointer-events-none" />
          <textarea
            rows={2}
            data-reader-toolbar="true"
            value={note}
            onChange={(e) => setNote(e.target.value)}
            onMouseDown={(e) => e.stopPropagation()}
            onClick={(e) => e.stopPropagation()}
            onKeyDown={(e) => {
              e.stopPropagation();
              if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
                e.preventDefault();
                handleSave();
              }
            }}
            placeholder={t("reader.image_note_placeholder", "Add a note for this image...")}
            className="reader-input w-full rounded-xl border border-(--reader-ui-border) bg-(--reader-ui-soft) pl-8 pr-12 py-1.5 text-xs text-(--reader-ui-text) placeholder:text-(--reader-ui-muted)/70 focus:border-(--reader-ui-accent) focus:outline-hidden transition-colors resize-none leading-relaxed"
          />
          {note.trim() && (
            <button
              type="button"
              onClick={handleSave}
              className="absolute right-2 top-2 btn btn-primary btn-xs h-6 rounded-lg px-2 text-[10px] font-bold"
            >
              {t("common.save", "Save")}
            </button>
          )}
        </div>
      </div>
    </div>
  );
};
