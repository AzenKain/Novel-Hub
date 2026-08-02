import { Link2, RefreshCw, Search } from "lucide-react";
import React, { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";

import {
  useMapBookTrackerMutation,
  useSearchTrackerMutation,
  useSyncTrackerProgressMutation,
  useTrackerReadingProgressQuery,
} from "@/hooks";
import { useAuthStore } from "@/stores";
import { usePublicSettings } from "@/hooks";
import { hasPermission } from "@/utils/permission";
import type { AniListSearchItem } from "@/types";

type TrackerMapCardProps = {
  book_id: string;
  title: string;
};

export const TrackerMapCard: React.FC<TrackerMapCardProps> = ({ book_id, title }) => {
  const { t } = useTranslation();
  const user = useAuthStore((state) => state.user);
  const publicSettings = usePublicSettings();
  const [seriesId, setSeriesId] = useState("");
  const [progress, setProgress] = useState("");
  const [results, setResults] = useState<AniListSearchItem[]>([]);

  const searchMutation = useSearchTrackerMutation();
  const mapMutation = useMapBookTrackerMutation();
  const syncMutation = useSyncTrackerProgressMutation();
  const { data: readingProgress } = useTrackerReadingProgressQuery(book_id);

  useEffect(() => {
    if (!progress && readingProgress?.chapter_index !== undefined) {
      setProgress(String(readingProgress.chapter_index + 1));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [readingProgress]);

  if (!hasPermission(user, "tracker.sync")) return null;
  if (publicSettings && publicSettings.enable_anilist_tracking === false) return null;

  const handleSearch = () => {
    const query = seriesId.trim() || title;
    setResults([]);
    searchMutation.mutate(query, {
      onSuccess: (data) => {
        if (!data.results.length) {
          toast.error(t("trackers.search_failed", "No AniList match found"));
          return;
        }
        setResults(data.results);
      },
      onError: (err) => {
        toast.error(err.message || t("trackers.search_failed", "No AniList match found"));
      },
    });
  };

  const handleSelectResult = (item: AniListSearchItem) => {
    setSeriesId(item.external_series_id);
  };

  const handleMap = (event: React.FormEvent) => {
    event.preventDefault();
    const trimmed = seriesId.trim();
    if (!trimmed) {
      toast.error(t("trackers.enter_series_id", "Enter or search for an AniList series ID"));
      return;
    }
    mapMutation.mutate(
      { book_id: book_id, provider: "anilist", external_series_id: trimmed },
      {
        onSuccess: () => toast.success(t("trackers.map_success", "Book linked to AniList")),
        onError: (err) =>
          toast.error(err.message || t("trackers.map_failed", "Failed to link book")),
      }
    );
  };

  const handleSync = () => {
    const value = Number(progress);
    if (!Number.isInteger(value) || value < 1) {
      toast.error(t("trackers.invalid_progress", "Progress must be a whole number of 1 or more"));
      return;
    }
    syncMutation.mutate(
      { book_id: book_id, title, progress: value },
      {
        onSuccess: () => toast.success(t("trackers.sync_success", "Progress synced to AniList")),
        onError: (err) =>
          toast.error(err.message || t("trackers.sync_failed", "Failed to sync progress")),
      }
    );
  };

  return (
    <div className="space-y-3">
      <h3 className="flex items-center gap-2 text-xl font-bold">
        <Link2 className="h-5 w-5" />
        {t("trackers.map_title", "AniList Tracking")}
      </h3>
      <p className="text-xs text-base-content/60">
        {t(
          "trackers.map_subtitle",
          "Link this book to an AniList series, then push your reading progress."
        )}
      </p>

      <form onSubmit={handleMap} className="flex flex-col gap-3 sm:flex-row sm:items-end">
        <div className="flex flex-1 min-w-0 flex-col gap-1.5">
          <label className="pl-1 text-xs font-medium" htmlFor="tracker-series-id">
            {t("trackers.series_id_label", "AniList series ID")}
          </label>
          <input
            id="tracker-series-id"
            type="text"
            className="input input-bordered input-sm w-full focus:input-primary"
            placeholder={t("trackers.series_id_placeholder", "e.g. 30002")}
            value={seriesId}
            onChange={(event) => setSeriesId(event.target.value)}
          />
        </div>
        <div className="flex shrink-0 gap-2">
          <button
            type="button"
            onClick={handleSearch}
            className="btn btn-outline btn-sm gap-2"
            disabled={searchMutation.isPending}
          >
            {searchMutation.isPending ? (
              <span className="loading loading-spinner loading-xs" />
            ) : (
              <Search className="h-4 w-4" />
            )}
            {t("trackers.search_btn", "Search")}
          </button>
          <button
            type="submit"
            className="btn btn-primary btn-sm gap-2"
            disabled={!seriesId.trim() || mapMutation.isPending}
          >
            {mapMutation.isPending ? (
              <span className="loading loading-spinner loading-xs" />
            ) : (
              <Link2 className="h-4 w-4" />
            )}
            {t("trackers.map_btn", "Link")}
          </button>
        </div>
      </form>

      {results.length > 0 && (
        <div className="rounded-lg border border-base-300 divide-y divide-base-200 max-h-64 overflow-y-auto">
          {results.map((item) => (
            <button
              type="button"
              key={`${item.media_type}-${item.external_series_id}`}
              onClick={() => handleSelectResult(item)}
              className={`w-full text-left p-2.5 text-sm hover:bg-base-200/60 transition-colors flex items-center justify-between gap-3 ${
                seriesId === item.external_series_id ? "bg-primary/10" : ""
              }`}
            >
              <span className="truncate">
                {item.title_english || item.title_romaji || item.external_series_id}
                {item.title_romaji && item.title_english && item.title_romaji !== item.title_english && (
                  <span className="text-base-content/50"> ({item.title_romaji})</span>
                )}
              </span>
              <span className="badge badge-ghost badge-sm shrink-0">{item.media_type}</span>
            </button>
          ))}
        </div>
      )}

      <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
        <div className="flex flex-1 min-w-0 flex-col gap-1.5">
          <label className="pl-1 text-xs font-medium" htmlFor="tracker-progress">
            {t("trackers.progress_label", "Chapters read")}
          </label>
          <input
            id="tracker-progress"
            type="number"
            min={1}
            className="input input-bordered input-sm w-full focus:input-primary"
            placeholder={t("trackers.progress_placeholder", "e.g. 12")}
            value={progress}
            onChange={(event) => setProgress(event.target.value)}
          />
        </div>
        <button
          type="button"
          onClick={handleSync}
          className="btn btn-outline btn-sm shrink-0 gap-2"
          disabled={!progress.trim() || syncMutation.isPending}
        >
          {syncMutation.isPending ? (
            <span className="loading loading-spinner loading-xs" />
          ) : (
            <RefreshCw className="h-4 w-4" />
          )}
          {t("trackers.sync_btn", "Sync progress")}
        </button>
      </div>
    </div>
  );
};
