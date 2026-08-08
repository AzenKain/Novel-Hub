import { TopNav } from "@/components/common/TopNav";
import { getMediaUrl } from "@/config/api";
import { offlineStore, type OfflineBook } from "@/lib/offlineStore";
import { useLibraryStore } from "@/stores";
import { ArrowLeft, BookOpen, CloudOff, Search, Trash2 } from "lucide-react";
import React, { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { useShallow } from "zustand/react/shallow";

const formatBytes = (bytes: number) => {
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(0)} KB`;
  if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(1)} MB`;
  return `${(bytes / 1024 ** 3).toFixed(2)} GB`;
};

export const OfflineBooksPage: React.FC = () => {
  const { t } = useTranslation();
  const [books, setBooks] = useState<OfflineBook[]>([]);
  const [coverMap, setCoverMap] = useState<Record<string, string>>({});
  const [usage, setUsage] = useState<{ usage: number; quota: number } | null>(null);

  const { search, setSearch } = useLibraryStore(
    useShallow((state) => ({ search: state.search, setSearch: state.setSearch }))
  );

  const refresh = () => {
    void offlineStore.listBooks().then(async (list) => {
      setBooks(list);
      const newMap: Record<string, string> = {};
      for (const entry of list) {
        try {
          const blob = await offlineStore.getBlob(entry.book.id, "cover");
          if (blob) {
            newMap[entry.book.id] = URL.createObjectURL(blob);
          }
        } catch (e) {
          // ignore
        }
      }
      setCoverMap(newMap);
    }).catch(() => setBooks([]));
    void offlineStore.usage().then(setUsage).catch(() => setUsage(null));
  };

  useEffect(refresh, []);

  const filteredBooks = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return books;
    return books.filter((entry) => {
      const titleMatch = entry.book.title?.toLowerCase().includes(q);
      const authorMatch = entry.book.author_name?.toLowerCase().includes(q);
      const descMatch = entry.book.description?.toLowerCase().includes(q);

      let metaMatch = false;
      if (entry.book.metadata_json) {
        try {
          const meta = JSON.parse(entry.book.metadata_json);
          metaMatch =
            Boolean(meta.series?.toLowerCase().includes(q)) ||
            Boolean(meta.publisher?.toLowerCase().includes(q)) ||
            Boolean(meta.publishers?.some((p: string) => p.toLowerCase().includes(q))) ||
            Boolean(meta.creators?.some((c: string) => c.toLowerCase().includes(q))) ||
            Boolean(meta.subject?.some((s: string) => s.toLowerCase().includes(q)));
        } catch {
          // ignore invalid json
        }
      }

      return titleMatch || authorMatch || descMatch || metaMatch;
    });
  }, [books, search]);

  return (
    <div className="bg-base-100 min-h-screen flex flex-col font-sans">
      <TopNav showSidebarToggle={false} />

      <div className="flex-1 container mx-auto p-4 sm:p-6 lg:p-8 max-w-[1700px] w-full flex flex-col gap-6">
        <div className="flex items-center justify-between gap-4 border-b border-base-200 pb-4">
          <div>
            <h1 className="text-2xl sm:text-3xl font-black text-base-content">{t("offline.title")}</h1>
            {usage && (
              <p className="text-xs sm:text-sm text-base-content/60 mt-1">
                {t("offline.usage", {
                  used: formatBytes(usage.usage),
                  quota: formatBytes(usage.quota),
                })}
              </p>
            )}
          </div>
          <Link to="/" className="btn btn-ghost btn-sm gap-1.5 text-primary">
            <ArrowLeft className="h-4 w-4" />
            {t("library.back_to_library", "Back to Library")}
          </Link>
        </div>

        {books.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-3 py-20 rounded-2xl border border-dashed border-base-300 bg-base-100 shadow-2xs text-center">
            <div className="grid h-16 w-16 place-items-center rounded-2xl bg-base-200 text-base-content/40">
              <CloudOff className="w-8 h-8" />
            </div>
            <div>
              <p className="font-bold text-base text-base-content/80">{t("offline.empty", "No offline books saved yet")}</p>
              <p className="text-xs text-base-content/50 mt-1">{t("offline.empty_hint", "Books you save for offline reading will appear here.")}</p>
            </div>
          </div>
        ) : filteredBooks.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-3 py-20 rounded-2xl border border-dashed border-base-300 bg-base-100 shadow-2xs text-center">
            <div className="grid h-16 w-16 place-items-center rounded-2xl bg-base-200 text-base-content/40">
              <Search className="w-8 h-8" />
            </div>
            <div>
              <p className="font-bold text-base text-base-content/80">{t("offline.no_match", "No matching offline books found")}</p>
              <p className="text-xs text-base-content/50 mt-1">{t("offline.no_match_hint", "Try searching for a different title, author, or tag.")}</p>
              <button
                type="button"
                onClick={() => setSearch("")}
                className="btn btn-ghost btn-xs text-primary mt-3"
              >
                {t("common.clear_search", "Clear search")}
              </button>
            </div>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-5 3xl:grid-cols-6 gap-4">
            {filteredBooks.map((entry) => {
              const coverUrl = coverMap[entry.book.id] || (entry.book.cover_url ? getMediaUrl(entry.book.cover_url) : null);
              return (
                <div
                  key={entry.book.id}
                  className="group relative flex items-center justify-between gap-3.5 p-3 rounded-2xl border border-base-200 bg-base-100 shadow-2xs hover:shadow-md hover:border-primary/40 transition-all"
                >
                  <Link to={`/offline/reader/${entry.book.id}`} className="min-w-0 flex items-center gap-3.5 flex-1">
                    <div className="relative aspect-[3/4.2] w-14 shrink-0 rounded-xl overflow-hidden bg-base-200 border border-base-200 shadow-2xs">
                      {coverUrl ? (
                        <img
                          src={coverUrl}
                          alt={entry.book.title}
                          loading="lazy"
                          className="w-full h-full object-cover transition-transform duration-200 group-hover:scale-105"
                        />
                      ) : (
                        <div className="grid w-full h-full place-items-center bg-primary/10 text-primary">
                          <BookOpen className="w-6 h-6" />
                        </div>
                      )}
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-bold text-base-content group-hover:text-primary transition-colors truncate">
                        {entry.book.title}
                      </p>
                      {entry.book.author_name && (
                        <p className="text-xs text-base-content/60 truncate mt-0.5">
                          {entry.book.author_name}
                        </p>
                      )}
                      <p className="text-[11px] font-medium text-base-content/45 mt-1">
                        {t("offline.saved_at", { date: new Date(entry.savedAt).toLocaleDateString() })}
                        {" · "}
                        {t("offline.chapter_count", { count: entry.chapters.length })}
                      </p>
                    </div>
                  </Link>
                  <button
                    className="btn btn-sm btn-ghost btn-square text-error hover:bg-error/10 rounded-xl"
                    onClick={() => void offlineStore.deleteBook(entry.book.id).then(refresh)}
                    aria-label={t("offline.remove")}
                    title={t("offline.remove", "Remove")}
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};
