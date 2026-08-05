import { TopNav } from "@/components/common/TopNav";
import { offlineStore, type OfflineBook } from "@/lib/offlineStore";
import { ArrowLeft, BookOpen, CloudOff, Trash2 } from "lucide-react";
import React, { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

const formatBytes = (bytes: number) => {
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(0)} KB`;
  if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(1)} MB`;
  return `${(bytes / 1024 ** 3).toFixed(2)} GB`;
};

export const OfflineBooksPage: React.FC = () => {
  const { t } = useTranslation();
  const [books, setBooks] = useState<OfflineBook[]>([]);
  const [usage, setUsage] = useState<{ usage: number; quota: number } | null>(null);

  const refresh = () => {
    void offlineStore.listBooks().then(setBooks).catch(() => setBooks([]));
    void offlineStore.usage().then(setUsage).catch(() => setUsage(null));
  };

  useEffect(refresh, []);

  return (
    <div className="bg-base-200 min-h-screen flex flex-col">
      <TopNav showSidebarToggle={false} />

      <div className="flex-1 container mx-auto p-4 sm:p-6 max-w-3xl flex flex-col gap-6">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl sm:text-3xl font-black">{t("offline.title")}</h1>
            {usage && (
              <p className="text-xs sm:text-sm text-base-content/60">
                {t("offline.usage", {
                  used: formatBytes(usage.usage),
                  quota: formatBytes(usage.quota),
                })}
              </p>
            )}
          </div>
          <Link to="/" className="btn btn-ghost btn-sm gap-1 text-primary">
            <ArrowLeft className="h-4 w-4" />
            {t("library.back_to_library", "Back to Library")}
          </Link>
        </div>

        {books.length === 0 ? (
          <div className="flex flex-col items-center gap-2 py-16 opacity-60 rounded-2xl border border-base-300 bg-base-100">
            <CloudOff className="w-10 h-10" />
            <span className="text-sm">{t("offline.empty")}</span>
          </div>
        ) : (
          <ul className="flex flex-col gap-2">
            {books.map((entry) => (
              <li
                key={entry.book.id}
                className="flex items-center justify-between gap-3 p-4 rounded-2xl border border-base-300 bg-base-100"
              >
                <Link to={`/reader/${entry.book.id}`} className="min-w-0 flex items-center gap-3 flex-1">
                  <BookOpen className="w-4 h-4 shrink-0 opacity-60" />
                  <div className="min-w-0">
                    <p className="text-sm font-medium truncate">{entry.book.title}</p>
                    <p className="text-xs opacity-60">
                      {t("offline.saved_at", { date: new Date(entry.savedAt).toLocaleDateString() })}
                      {" · "}
                      {t("offline.chapter_count", { count: entry.chapters.length })}
                    </p>
                  </div>
                </Link>
                <button
                  className="btn btn-sm btn-ghost text-error"
                  onClick={() => void offlineStore.deleteBook(entry.book.id).then(refresh)}
                  aria-label={t("offline.remove")}
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
};
