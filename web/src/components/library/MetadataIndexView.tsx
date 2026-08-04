import { BookOpen } from "lucide-react";
import React from "react";

import { getMediaUrl } from "@/config/api";
import type { MetadataCount } from "@/types";

export type MetadataFacetSection = {
  nav: string;
  type: string;
  label: string;
  icon: React.ReactNode;
  items: MetadataCount[];
};

type MetadataIndexViewProps = {
  section: MetadataFacetSection;
  filteredItems: MetadataCount[];
  controls: React.ReactNode;
  t: (key: string, fallback: string) => string;
  onFacetClick: (type: string, item: MetadataCount, nav: string) => void;
  hasMore?: boolean;
  loadingMore?: boolean;
  onLoadMore?: () => void;
};

export const MetadataIndexView: React.FC<MetadataIndexViewProps> = ({
  section,
  filteredItems,
  controls,
  t,
  onFacetClick,
  hasMore,
  loadingMore,
  onLoadMore,
}) => {
  const isSeries = section.type === "series";

  return (
    <section className="min-h-[calc(100vh-7rem)] rounded-2xl border border-base-200 bg-base-100 p-4 shadow-sm sm:p-6">
      <div className="mb-5 flex flex-col gap-4">
        <div>
          <h2 className="text-3xl font-black tracking-tight sm:text-4xl">
            {section.label}
          </h2>
          <p className="mt-1 text-sm text-base-content/50">
            {/* Filtering now happens server-side, so loaded == matching. Showing "x / x"
                would read as if something were filtered out. */}
            {filteredItems.length}
            {hasMore ? "+" : ""} {t("library.entries", "entries")}
          </p>
        </div>
        {controls}
      </div>

      {filteredItems.length === 0 ? (
        <div className="rounded-xl border border-dashed border-base-300 bg-base-200/30 p-12 text-center">
          <BookOpen className="mx-auto mb-3 h-10 w-10 text-base-content/25" />
          <p className="font-bold text-base-content/70">
            {t("library.no_items", "No items")}
          </p>
        </div>
      ) : isSeries ? (
        <div className="grid grid-cols-2 gap-x-5 gap-y-8 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6 items-start">
          {filteredItems.map((item) => {
            const cover_url = item.cover_url ? getMediaUrl(item.cover_url) : "";
            return (
              <button
                key={item.id}
                className="group text-left focus-visible:outline-none"
                onClick={() => onFacetClick(section.type, item, section.nav)}
              >
                <span className="relative block aspect-[3/4.18] overflow-hidden rounded-lg bg-base-300 shadow-sm ring-1 ring-base-300 transition-[box-shadow,outline-color] duration-200 ease-out group-hover:shadow-md group-hover:ring-primary/35 group-focus-visible:ring-2 group-focus-visible:ring-primary/45">
                  {cover_url ? (
                    <>
                      <img
                        src={cover_url}
                        alt={item.name}
                        loading="lazy"
                        className="absolute inset-0 h-full w-full object-cover transition-[filter] duration-150 ease-out motion-reduce:transition-none group-hover:brightness-105"
                      />
                      <span className="absolute inset-0 bg-primary/0 transition-colors duration-200 ease-out group-hover:bg-primary/[0.03]" />
                    </>
                  ) : (
                    <span className="absolute inset-0 grid place-items-center text-xl font-black text-base-content/25">
                      NH
                    </span>
                  )}
                  <span className="absolute left-2 top-2 rounded-full bg-base-100/95 px-2 py-0.5 text-xs font-black text-base-content shadow">
                    {item.book_count}
                  </span>
                </span>
                <span className="mt-3 block line-clamp-3 text-sm font-black leading-snug transition group-hover:text-primary sm:text-base">
                  {item.name}
                </span>
              </button>
            );
          })}
        </div>
      ) : (
        <div className="grid gap-x-10 gap-y-1 sm:grid-cols-2 xl:grid-cols-3">
          {filteredItems.map((item) => (
            <button
              key={item.id}
              className="group grid grid-cols-[auto_minmax(0,1fr)] items-center gap-3 rounded-lg px-2 py-1.5 text-left hover:bg-base-200/70"
              onClick={() => onFacetClick(section.type, item, section.nav)}
            >
              <span className="badge badge-neutral badge-sm min-w-8">
                {item.book_count}
              </span>
              <span className="truncate text-sm font-semibold text-primary group-hover:underline sm:text-base">
                {item.name}
              </span>
            </button>
          ))}
        </div>
      )}

      {hasMore && (
        <div className="mt-6 flex justify-center">
          <button
            className="btn btn-outline btn-wide"
            disabled={loadingMore}
            onClick={onLoadMore}
          >
            {loadingMore ? (
              <span className="loading loading-spinner loading-sm" />
            ) : (
              t("common.load_more", "Load more")
            )}
          </button>
        </div>
      )}
    </section>
  );
};
