import React from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Layers, ChevronRight, BookOpen } from "lucide-react";
import { useSeriesBooksQuery } from "@/hooks/useBooksQuery";
import { getMediaUrl } from "@/config/api";
import { Book } from "@/types";

interface SeriesBooksSectionProps {
  currentBookId: string;
  seriesId: string;
  seriesName: string;
  seriesIndex?: string;
}

export const SeriesBooksSection: React.FC<SeriesBooksSectionProps> = ({
  currentBookId,
  seriesId,
  seriesName,
}) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { data: books = [], isLoading } = useSeriesBooksQuery(seriesId, !!seriesId);

  if (!seriesId || (books.length <= 1 && !isLoading)) {
    return null;
  }

  return (
    <div className="pt-4 border-t border-base-200 mt-4">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <Layers className="w-4 h-4 text-primary" />
          <h3 className="text-base font-bold text-base-content">
            {t("book.in_same_series", "Books in this Series")}
          </h3>
          <span className="badge badge-sm badge-ghost font-medium">
            {seriesName}
          </span>
        </div>
        <button
          onClick={() =>
            navigate(
              `/?nav=series&facet=series&facet_id=${encodeURIComponent(seriesId)}&name=${encodeURIComponent(seriesName)}`
            )
          }
          className="btn btn-ghost btn-xs gap-1 text-primary hover:bg-primary/10"
        >
          <span>{t("common.view_all", "View All")}</span>
          <ChevronRight className="w-3.5 h-3.5" />
        </button>
      </div>

      {isLoading ? (
        <div className="flex gap-3 overflow-x-auto pb-2">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="w-28 shrink-0 flex flex-col gap-2 animate-pulse">
              <div className="w-28 h-40 bg-base-300 rounded-lg" />
              <div className="h-3 bg-base-300 rounded w-3/4" />
            </div>
          ))}
        </div>
      ) : (
        <div className="flex gap-3 overflow-x-auto pb-2 scrollbar-thin">
          {books.map((b: Book) => {
            const isCurrent = b.id === currentBookId;
            return (
              <div
                key={b.id}
                onClick={() => navigate(`/books/${encodeURIComponent(b.id)}`)}
                className={`group w-28 shrink-0 cursor-pointer flex flex-col gap-1.5 transition-transform hover:-translate-y-1 ${
                  isCurrent ? "opacity-100" : "opacity-85 hover:opacity-100"
                }`}
              >
                <div className="relative w-28 h-40 rounded-lg overflow-hidden border border-base-300 shadow-2xs bg-base-200 group-hover:border-primary transition-colors">
                  {b.cover_url ? (
                    <img
                      src={getMediaUrl(b.cover_url)}
                      alt={b.title}
                      className="w-full h-full object-cover"
                      loading="lazy"
                    />
                  ) : (
                    <div className="w-full h-full flex items-center justify-center text-base-content/40">
                      <BookOpen className="w-8 h-8" />
                    </div>
                  )}

                  {isCurrent && (
                    <span className="absolute top-1.5 right-1.5 badge badge-primary badge-xs font-bold text-[9px] shadow-xs">
                      {t("common.current", "Current")}
                    </span>
                  )}
                </div>

                <div className="min-w-0">
                  <p
                    className="text-xs font-semibold text-base-content line-clamp-2 leading-snug group-hover:text-primary transition-colors"
                    title={b.title}
                  >
                    {b.title}
                  </p>
                  {b.author_name && (
                    <p className="text-[11px] text-base-content/60 truncate mt-0.5">
                      {b.author_name}
                    </p>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};
