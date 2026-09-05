import { TopNav } from "@/components/common/TopNav";
import { DeleteConfirmModal } from "@/components/admin/books/DeleteConfirmModal";
import {
  useDeletePodcastMutation,
  useDownloadEpisodeMutation,
  useLibrariesQuery,
  usePodcastEpisodesQuery,
  usePodcastsQuery,
  useRefreshPodcastMutation,
  useSubscribePodcastMutation,
  useUpdatePodcastMutation,
} from "@/hooks";
import type { Podcast } from "@/types";
import {
  ArrowLeft,
  Check,
  Download,
  Plus,
  Podcast as PodcastIcon,
  RefreshCw,
  Rss,
  Trash2,
} from "lucide-react";
import React, { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { toast } from "react-toastify";
import { usePodcastDownloadStore } from "@/stores";

function formatDuration(secs?: number | null): string {
  if (!secs) return "";
  const h = Math.floor(secs / 3600);
  const m = Math.floor((secs % 3600) / 60);
  const s = secs % 60;
  if (h > 0)
    return `${h}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
  return `${m}:${String(s).padStart(2, "0")}`;
}

export const PodcastsPage: React.FC = () => {
  const { t } = useTranslation();
  const [feedURL, setFeedURL] = useState("");
  const [libraryID, setLibraryID] = useState("");
  const [activePodcastId, setActivePodcastId] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Podcast | null>(null);

  const activeDownloads = usePodcastDownloadStore(
    (state) => state.activeDownloads,
  );

  const podcastsQuery = usePodcastsQuery();
  const podcasts = podcastsQuery.data || [];
  const librariesQuery = useLibrariesQuery();
  const libraries = librariesQuery.data || [];

  const episodesQuery = usePodcastEpisodesQuery(activePodcastId || "");
  const episodes = episodesQuery.data || [];

  const subscribeMutation = useSubscribePodcastMutation();
  const updateMutation = useUpdatePodcastMutation();
  const deleteMutation = useDeletePodcastMutation();
  const refreshMutation = useRefreshPodcastMutation();
  const downloadMutation = useDownloadEpisodeMutation(activePodcastId || "");

  useEffect(() => {
    if (!activePodcastId && podcasts.length > 0)
      setActivePodcastId(podcasts[0].id);
  }, [activePodcastId, podcasts]);

  useEffect(() => {
    if (!libraryID && libraries.length > 0) setLibraryID(libraries[0].id);
  }, [libraryID, libraries]);

  const handleSubscribe = () => {
    if (!feedURL.trim() || !libraryID) return;
    subscribeMutation.mutate(
      { feed_url: feedURL.trim(), library_id: libraryID },
      {
        onSuccess: (podcast) => {
          toast.success(t("podcasts.subscribed", "Subscribed to podcast"));
          setFeedURL("");
          if (podcast) setActivePodcastId(podcast.id);
        },
        onError: (err) =>
          toast.error(
            err instanceof Error
              ? err.message
              : t("podcasts.subscribe_failed", "Failed to subscribe"),
          ),
      },
    );
  };

  const handleToggleAutoDownload = (podcast: Podcast) => {
    updateMutation.mutate({
      id: podcast.id,
      input: { auto_download: !podcast.auto_download },
    });
  };

  const handleRefresh = (podcast: Podcast) => {
    refreshMutation.mutate(podcast.id, {
      onSuccess: () =>
        toast.success(t("podcasts.refresh_queued", "Refresh queued")),
      onError: (err) =>
        toast.error(
          err instanceof Error
            ? err.message
            : t("podcasts.refresh_failed", "Failed to refresh"),
        ),
    });
  };

  const handleDownload = (episodeId: string, episodeTitle?: string) => {
    downloadMutation.mutate(
      { episodeId, episodeTitle },
      {
        onSuccess: () =>
          toast.info(t("podcasts.download_queued", "Download queued")),
        onError: (err) =>
          toast.error(
            err instanceof Error
              ? err.message
              : t("podcasts.download_failed", "Failed to enqueue download"),
          ),
      },
    );
  };

  const activePodcast = podcasts.find((p) => p.id === activePodcastId) || null;

  return (
    <div className="min-h-screen bg-base-200/40 flex flex-col font-sans">
      <TopNav showSidebarToggle={false} />
      <div className="flex-1 container mx-auto p-4 sm:p-6 lg:p-8 max-w-[1700px] w-full flex flex-col gap-6">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-2 sm:gap-3">
            <Link
              to="/"
              className="btn btn-ghost btn-sm btn-square sm:w-auto sm:px-3 gap-1.5 text-primary -ml-2 sm:-ml-2.5 shrink-0"
              aria-label={t("library.back_to_library", "Back to Library")}
              title={t("library.back_to_library", "Back to Library")}
            >
              <ArrowLeft className="h-4 w-4" />
              <span className="hidden sm:inline whitespace-nowrap">
                {t("library.back_to_library", "Back to Library")}
              </span>
            </Link>
            <div className="h-5 sm:h-6 w-px bg-base-300 shrink-0" />
            <h1 className="flex items-center gap-2 text-lg sm:text-xl font-black text-base-content">
              <PodcastIcon className="h-5 w-5 text-primary shrink-0" />
              <span>{t("podcasts.title", "Podcasts")}</span>
            </h1>
          </div>
        </div>

        <div className="flex flex-col gap-3 sm:flex-row sm:items-end w-full">
          <div className="flex-1">
            <label className="block text-xs font-bold text-base-content/80 mb-1.5 pl-1">
              {t("podcasts.feed_url", "Feed URL")}
            </label>
            <input
              type="url"
              className="input input-bordered input-md w-full rounded-xl bg-base-100 shadow-2xs text-sm font-medium"
              placeholder="https://example.com/feed.xml"
              value={feedURL}
              onChange={(e) => setFeedURL(e.target.value)}
            />
          </div>
          <div className="sm:w-56">
            <label className="block text-xs font-bold text-base-content/80 mb-1.5 pl-1">
              {t("podcasts.library", "Library")}
            </label>
            <select
              className="select select-bordered select-md w-full rounded-xl bg-base-100 shadow-2xs text-sm font-medium"
              value={libraryID}
              onChange={(e) => setLibraryID(e.target.value)}
            >
              {libraries.map((lib) => (
                <option key={lib.id} value={lib.id}>
                  {lib.name}
                </option>
              ))}
            </select>
          </div>
          <button
            className="btn btn-primary btn-md rounded-xl font-bold gap-1.5 px-6 shadow-2xs shrink-0"
            onClick={handleSubscribe}
            disabled={!feedURL.trim() || subscribeMutation.isPending}
          >
            {subscribeMutation.isPending ? (
              <span className="loading loading-spinner loading-xs" />
            ) : (
              <Plus className="w-4 h-4" />
            )}
            {t("podcasts.subscribe", "Subscribe")}
          </button>
        </div>

        {podcasts.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-3 py-16 rounded-2xl border border-dashed border-base-300 bg-base-100 shadow-2xs text-center">
            <div className="grid h-16 w-16 place-items-center rounded-2xl bg-base-200 text-base-content/40">
              <PodcastIcon className="w-8 h-8" />
            </div>
            <div>
              <p className="font-bold text-base text-base-content/80">
                {t("podcasts.empty", "No podcast feeds subscribed yet")}
              </p>
              <p className="text-xs text-base-content/50 mt-1">
                {t(
                  "podcasts.empty_hint",
                  "Paste an RSS feed URL above to subscribe to your favorite podcasts.",
                )}
              </p>
            </div>
          </div>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {podcasts.map((podcast) => {
              const isActive = activePodcastId === podcast.id;
              return (
                <div
                  key={podcast.id}
                  className={`rounded-2xl border transition-all shadow-2xs cursor-pointer ${
                    isActive
                      ? "border-primary bg-primary/5 ring-1 ring-primary/30"
                      : "border-base-200 bg-base-100 hover:border-primary/40 hover:shadow-md"
                  }`}
                  onClick={() => setActivePodcastId(podcast.id)}
                >
                  <div className="p-4 sm:p-5">
                    <div className="flex items-start gap-3.5">
                      {podcast.cover_url &&
                      (podcast.cover_url.startsWith("http://") ||
                        podcast.cover_url.startsWith("https://")) ? (
                        <div className="relative h-14 w-14 shrink-0 overflow-hidden rounded-xl border border-base-200 shadow-2xs">
                          <img
                            src={podcast.cover_url}
                            alt={podcast.title}
                            className="h-14 w-14 object-cover"
                            onError={(e) => {
                              e.currentTarget.style.display = "none";
                              e.currentTarget.nextElementSibling?.classList.remove(
                                "hidden",
                              );
                            }}
                          />
                          <div className="flex h-14 w-14 items-center justify-center bg-primary/10 text-primary">
                            <Rss className="h-6 w-6" />
                          </div>
                        </div>
                      ) : (
                        <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary border border-primary/20 shadow-2xs">
                          <Rss className="h-6 w-6" />
                        </div>
                      )}
                      <div className="min-w-0 flex-1">
                        <h3 className="truncate font-bold text-base-content text-sm sm:text-base">
                          {podcast.title}
                        </h3>
                        {podcast.author && (
                          <p className="truncate text-xs text-base-content/60 mt-0.5">
                            {podcast.author}
                          </p>
                        )}
                        <p className="text-xs text-base-content/50 mt-1">
                          {t("podcasts.episode_count", "{{count}} episodes", {
                            count: podcast.episode_count ?? 0,
                          })}
                        </p>
                      </div>
                    </div>
                    <div
                      className="mt-4 flex items-center justify-between pt-3 border-t border-base-200"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <label className="flex cursor-pointer items-center gap-1.5 text-xs font-medium text-base-content/70">
                        <input
                          type="checkbox"
                          className="toggle toggle-primary toggle-xs"
                          checked={podcast.auto_download}
                          onChange={() => handleToggleAutoDownload(podcast)}
                        />
                        {t("podcasts.auto_download", "Auto-download")}
                      </label>
                      <div className="ml-auto flex items-center gap-1">
                        <button
                          className="btn btn-ghost btn-xs btn-square rounded-lg"
                          title={t("podcasts.refresh", "Refresh")}
                          onClick={() => handleRefresh(podcast)}
                        >
                          <RefreshCw
                            className={`h-4 w-4 ${refreshMutation.isPending ? "animate-spin" : ""}`}
                          />
                        </button>
                        <button
                          className="btn btn-ghost btn-xs btn-square text-error rounded-lg hover:bg-error/10"
                          title={t("common.delete", "Delete")}
                          onClick={() => setDeleteTarget(podcast)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {activePodcast && (
          <div className="rounded-2xl border border-base-200 bg-base-100 shadow-2xs overflow-hidden flex flex-col">
            <div className="p-4 sm:p-5 border-b border-base-200 flex flex-wrap items-center justify-between gap-3 bg-base-100">
              <div className="flex items-center gap-3">
                <div className="grid h-10 w-10 place-items-center rounded-xl bg-primary/10 text-primary shrink-0">
                  <PodcastIcon className="h-5 w-5" />
                </div>
                <div>
                  <h2 className="text-base sm:text-lg font-bold text-base-content">
                    {activePodcast.title}
                  </h2>
                  {activePodcast.author && (
                    <p className="text-xs text-base-content/60 truncate">
                      {activePodcast.author}
                    </p>
                  )}
                </div>
              </div>
              <span className="text-xs font-semibold text-base-content/60 px-2.5 py-1 bg-base-200/70 rounded-lg">
                {t("podcasts.episode_count", "{{count}} episodes", {
                  count: episodes.length,
                })}
              </span>
            </div>

            {episodesQuery.isLoading ? (
              <div className="flex justify-center py-16">
                <span className="loading loading-spinner loading-lg text-primary" />
              </div>
            ) : episodes.length === 0 ? (
              <div className="p-12 text-center text-sm text-base-content/50">
                {t(
                  "podcasts.no_episodes",
                  "No episodes yet. Refresh to fetch the feed.",
                )}
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="table table-sm sm:table-md">
                  <thead>
                    <tr className="border-b border-base-200 bg-base-200/40 text-xs font-bold uppercase tracking-wider text-base-content/60">
                      <th className="py-3 pl-4 sm:pl-5">
                        {t("common.title", "Title")}
                      </th>
                      <th className="py-3">
                        {t("podcasts.duration", "Duration")}
                      </th>
                      <th className="py-3">
                        {t("podcasts.published", "Published")}
                      </th>
                      <th className="py-3 pr-4 sm:pr-5 text-right">
                        {t("podcasts.actions", "Actions")}
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-base-200">
                    {episodes.map((ep) => {
                      const isDownloading = !!activeDownloads[ep.id];
                      return (
                        <tr
                          key={ep.id}
                          className="hover:bg-base-200/30 transition-colors"
                        >
                          <td className="py-3 pl-4 sm:pl-5">
                            <div className="flex items-center gap-2 min-w-0">
                              <span className="truncate font-medium max-w-xs sm:max-w-md">
                                {ep.title}
                              </span>
                              {ep.downloaded ? (
                                <span className="badge badge-success badge-sm shrink-0 whitespace-nowrap">
                                  {t("podcasts.downloaded", "Downloaded")}
                                </span>
                              ) : isDownloading ? (
                                <span className="badge badge-warning badge-sm gap-1 animate-pulse shrink-0 whitespace-nowrap">
                                  <span className="loading loading-spinner loading-xs" />
                                  {t(
                                    "podcasts.downloading_short",
                                    "Downloading...",
                                  )}
                                </span>
                              ) : null}
                            </div>
                          </td>
                          <td className="whitespace-nowrap text-xs text-base-content/70">
                            {formatDuration(ep.duration_sec)}
                          </td>
                          <td className="whitespace-nowrap text-xs text-base-content/60">
                            {ep.published_at
                              ? new Date(ep.published_at).toLocaleDateString()
                              : "—"}
                          </td>
                          <td className="text-right py-3 pr-4 sm:pr-5">
                            {ep.downloaded ? (
                              <span className="text-xs text-success font-medium inline-flex items-center gap-1 px-1">
                                <Check className="h-4 w-4" />
                                <span className="hidden sm:inline">
                                  {t("podcasts.downloaded", "Downloaded")}
                                </span>
                              </span>
                            ) : isDownloading ? (
                              <div
                                className="inline-flex items-center gap-1 text-warning text-xs font-medium px-2 py-1 bg-warning/10 rounded-lg animate-pulse"
                                title={t(
                                  "podcasts.downloading",
                                  "Downloading episode...",
                                )}
                              >
                                <span className="loading loading-spinner loading-xs" />
                                <span className="hidden sm:inline">
                                  {t(
                                    "podcasts.downloading_short",
                                    "Downloading...",
                                  )}
                                </span>
                              </div>
                            ) : (
                              <button
                                className="btn btn-ghost btn-xs btn-square rounded-lg"
                                title={t("podcasts.download", "Download")}
                                disabled={downloadMutation.isPending}
                                onClick={() => handleDownload(ep.id, ep.title)}
                              >
                                <Download className="h-4 w-4" />
                              </button>
                            )}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}
      </div>

      {deleteTarget && (
        <DeleteConfirmModal
          open={!!deleteTarget}
          title={t("podcasts.delete_title", "Delete Podcast")}
          message={t(
            "podcasts.delete_confirm",
            "Delete this podcast and its episodes? Download episodes already imported as books are kept.",
          )}
          onClose={() => setDeleteTarget(null)}
          onConfirm={() => {
            deleteMutation.mutate(deleteTarget.id, {
              onSuccess: () => {
                toast.success(t("podcasts.deleted", "Podcast deleted"));
                if (activePodcastId === deleteTarget.id)
                  setActivePodcastId(podcasts[0]?.id ?? null);
                setDeleteTarget(null);
              },
              onError: (err) =>
                toast.error(
                  err instanceof Error
                    ? err.message
                    : t("podcasts.delete_failed", "Failed to delete podcast"),
                ),
            });
          }}
          loading={deleteMutation.isPending}
        />
      )}
    </div>
  );
};
