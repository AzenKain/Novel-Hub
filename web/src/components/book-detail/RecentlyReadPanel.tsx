import { ArrowRight, BookOpen, Clock3, History } from "lucide-react";
import React from "react";

import { getMediaUrl } from "@/config/api";
import type { ReadingHistory } from "@/types";

type RecentlyReadPanelProps = {
  items: ReadingHistory[];
  onOpen: (item: ReadingHistory) => void;
  t: (key: string, fallback: string) => string;
  className?: string;
};

export const RecentlyReadPanel: React.FC<RecentlyReadPanelProps> = ({
  items,
  onOpen,
  t,
  className = "mt-4",
}) => {
  const recent = items.slice(0, 4);
  return (
    <section
      className={`${className} overflow-hidden rounded-xl border border-base-300 bg-base-100 shadow-sm`}
    >
      <div className="flex items-center gap-2.5 border-b border-base-200 px-4 py-3.5">
        <span className="grid h-9 w-9 place-items-center rounded-xl bg-primary/10 text-primary">
          <History className="h-4 w-4" />
        </span>
        <div>
          <h3 className="font-black leading-tight">
            {t("library.recently_read", "Recently read")}
          </h3>
          <p className="text-xs text-base-content/50">
            {t("library.jump_back", "Jump back into your last books")}
          </p>
        </div>
      </div>

      <div className="flex flex-col gap-1.5 p-3">
        {recent.length > 0 ? (
          recent.map((item) => {
            const cover_url = item.book_cover_url
              ? getMediaUrl(item.book_cover_url)
              : "";
            const progress = Math.max(
              0,
              Math.min(100, Math.round(item.progress_percent || 0)),
            );
            return (
              <button
                key={`${item.book_id}-${item.chapter_id}`}
                className="group grid grid-cols-[44px_minmax(0,1fr)_auto] items-center gap-2.5 rounded-xl border border-transparent p-2 text-left transition-colors duration-150 hover:border-base-300 hover:bg-base-200/60"
                onClick={() => onOpen(item)}
              >
                <span className="relative block aspect-[3/4.12] overflow-hidden rounded-lg bg-base-300 shadow-sm">
                  {cover_url ? (
                    <img
                      src={cover_url}
                      alt={t("common.alt_cover", "Cover")}
                      loading="lazy"
                      className="absolute inset-0 h-full w-full object-cover"
                    />
                  ) : (
                    <span className="absolute inset-0 grid place-items-center text-[9px] font-black text-base-content/30">
                      NH
                    </span>
                  )}
                </span>
                <span className="min-w-0">
                  <span className="block truncate text-sm font-bold">
                    {item.book_title}
                  </span>
                  <span className="mt-0.5 flex items-center gap-1 text-xs text-base-content/55">
                    <Clock3 className="h-3.5 w-3.5 shrink-0" />
                    <span className="truncate">
                      {item.chapter_title || `Chapter ${item.chapter_index}`}
                    </span>
                  </span>
                  <span className="mt-2 block h-1.5 overflow-hidden rounded-full bg-base-300">
                    <span
                      className="block h-full rounded-full bg-primary"
                      style={{ width: `${progress}%` }}
                    />
                  </span>
                </span>
                <span className="flex flex-col items-end gap-1 text-xs font-bold text-base-content/45">
                  {progress}%
                  <ArrowRight className="h-4 w-4 transition-colors duration-150 group-hover:text-primary" />
                </span>
              </button>
            );
          })
        ) : (
          <div className="grid min-h-44 place-items-center rounded-xl border border-dashed border-base-300 bg-base-200/30 p-5 text-center">
            <div>
              <BookOpen className="mx-auto mb-3 h-8 w-8 text-base-content/25" />
              <p className="font-bold text-base-content/70">
                {t("library.no_recent_books", "No recently read books")}
              </p>
              <p className="mt-1 text-sm text-base-content/45">
                {t(
                  "library.no_recent_books_hint",
                  "Books you open in the reader will appear here.",
                )}
              </p>
            </div>
          </div>
        )}
      </div>
    </section>
  );
};
