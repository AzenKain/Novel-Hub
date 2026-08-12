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
import { ArrowLeft, Download, Plus, RefreshCw, Rss, Trash2 } from "lucide-react";
import React, { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { toast } from "react-toastify";

function formatDuration(secs?: number | null): string {
  if (!secs) return "";
  const h = Math.floor(secs / 3600);
  const m = Math.floor((secs % 3600) / 60);
  const s = secs % 60;
  if (h > 0) return `${h}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
  return `${m}:${String(s).padStart(2, "0")}`;
}

export const PodcastsPage: React.FC = () => {
  const { t } = useTranslation();
  const [feedURL, setFeedURL] = useState("");
  const [libraryID, setLibraryID] = useState("");
  const [activePodcastId, setActivePodcastId] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Podcast | null>(null);

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
    if (!activePodcastId && podcasts.length > 0) setActivePodcastId(podcasts[0].id);
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
        onError: (err) => toast.error(err instanceof Error ? err.message : t("podcasts.subscribe_failed", "Failed to subscribe")),
      },
    );
  };

  const handleToggleAutoDownload = (podcast: Podcast) => {
    updateMutation.mutate({ id: podcast.id, input: { auto_download: !podcast.auto_download } });
  };

  const handleRefresh = (podcast: Podcast) => {
    refreshMutation.mutate(podcast.id, {
      onSuccess: () => toast.success(t("podcasts.refresh_queued", "Refresh queued")),
      onError: (err) => toast.error(err instanceof Error ? err.message : t("podcasts.refresh_failed", "Failed to refresh")),
    });
  };

  const handleDownload = (episodeId: string) => {
    downloadMutation.mutate(episodeId, {
      onSuccess: () => toast.success(t("podcasts.download_queued", "Download queued")),
      onError: (err) => toast.error(err instanceof Error ? err.message : t("podcasts.download_failed", "Failed to enqueue download")),
    });
  };

  const activePodcast = podcasts.find((p) => p.id === activePodcastId) || null;

  return (
    <div className="flex min-h-screen flex-col bg-base-100">
      <TopNav />
      <div className="mx-auto w-full max-w-6xl flex-1 px-4 py-6">
        <div className="mb-6 flex items-center gap-3">
          <Link to="/" className="btn btn-ghost btn-circle btn-sm" aria-label={t("common.back", "Back")}>
            <ArrowLeft className="w-5 h-5" />
          </Link>
          <h1 className="text-2xl font-bold">{t("podcasts.title", "Podcasts")}</h1>
        </div>

        <div className="card bg-base-200/60 border border-base-200 p-4 mb-6">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
            <div className="flex-1">
              <label className="block text-xs font-medium text-base-content/70 mb-1">
                {t("podcasts.feed_url", "Feed URL")}
              </label>
              <input
                type="url"
                className="input input-bordered w-full rounded-xl"
                placeholder="https://example.com/feed.xml"
                value={feedURL}
                onChange={(e) => setFeedURL(e.target.value)}
              />
            </div>
            <div className="sm:w-56">
              <label className="block text-xs font-medium text-base-content/70 mb-1">
                {t("podcasts.library", "Library")}
              </label>
              <select
                className="select select-bordered w-full rounded-xl"
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
              className="btn btn-primary rounded-xl text-white"
              onClick={handleSubscribe}
              disabled={!feedURL.trim() || subscribeMutation.isPending}
            >
              {subscribeMutation.isPending ? <span className="loading loading-spinner loading-xs" /> : <Plus className="w-4 h-4" />}
              {t("podcasts.subscribe", "Subscribe")}
            </button>
          </div>
        </div>

        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {podcasts.map((podcast) => (
            <div
              key={podcast.id}
              className={`card cursor-pointer border transition-all ${
                activePodcastId === podcast.id
                  ? "border-primary bg-primary/5"
                  : "border-base-200 bg-base-200/40 hover:border-primary/40"
              }`}
              onClick={() => setActivePodcastId(podcast.id)}
            >
              <div className="card-body p-4">
                <div className="flex items-start gap-3">
                  {podcast.cover_url ? (
                    <img src={podcast.cover_url} alt={podcast.title} className="h-14 w-14 rounded-lg object-cover" />
                  ) : (
                    <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                      <Rss className="h-6 w-6" />
                    </div>
                  )}
                  <div className="min-w-0 flex-1">
                    <h3 className="truncate font-bold">{podcast.title}</h3>
                    {podcast.author && <p className="truncate text-xs text-base-content/60">{podcast.author}</p>}
                    <p className="text-xs text-base-content/50">
                      {t("podcasts.episode_count", "{{count}} episodes", { count: podcast.episode_count ?? 0 })}
                    </p>
                  </div>
                </div>
                <div className="mt-2 flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
                  <label className="flex cursor-pointer items-center gap-1.5 text-xs">
                    <input
                      type="checkbox"
                      className="toggle toggle-primary toggle-xs"
                      checked={podcast.auto_download}
                      onChange={() => handleToggleAutoDownload(podcast)}
                    />
                    {t("podcasts.auto_download", "Auto-download")}
                  </label>
                  <div className="ml-auto flex items-center gap-0.5">
                    <button
                      className="btn btn-ghost btn-xs btn-square"
                      title={t("podcasts.refresh", "Refresh")}
                      onClick={() => handleRefresh(podcast)}
                    >
                      <RefreshCw className={`h-4 w-4 ${refreshMutation.isPending ? "animate-spin" : ""}`} />
                    </button>
                    <button
                      className="btn btn-ghost btn-xs btn-square text-error"
                      title={t("common.delete", "Delete")}
                      onClick={() => setDeleteTarget(podcast)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>

        {activePodcast && (
          <div className="mt-8">
            <h2 className="mb-3 text-lg font-bold">{activePodcast.title}</h2>
            {episodesQuery.isLoading ? (
              <div className="flex justify-center py-8">
                <span className="loading loading-spinner loading-lg" />
              </div>
            ) : episodes.length === 0 ? (
              <div className="rounded-xl border border-dashed border-base-300 p-8 text-center text-sm text-base-content/50">
                {t("podcasts.no_episodes", "No episodes yet. Refresh to fetch the feed.")}
              </div>
            ) : (
              <div className="overflow-x-auto rounded-xl border border-base-200">
                <table className="table table-sm">
                  <thead>
                    <tr className="text-xs uppercase tracking-wider text-base-content/50">
                      <th>{t("common.title", "Title")}</th>
                      <th>{t("podcasts.duration", "Duration")}</th>
                      <th>{t("podcasts.published", "Published")}</th>
                      <th className="text-right">{t("common.actions", "Actions")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {episodes.map((ep) => (
                      <tr key={ep.id} className="hover:bg-base-200/40">
                        <td>
                          <div className="flex items-center gap-2">
                            <span className="truncate font-medium max-w-xs">{ep.title}</span>
                            {ep.downloaded && (
                              <span className="badge badge-success badge-sm">{t("podcasts.downloaded", "Downloaded")}</span>
                            )}
                          </div>
                        </td>
                        <td className="whitespace-nowrap text-xs">{formatDuration(ep.duration_sec)}</td>
                        <td className="whitespace-nowrap text-xs text-base-content/60">
                          {ep.published_at ? new Date(ep.published_at).toLocaleDateString() : "—"}
                        </td>
                        <td className="text-right">
                          <button
                            className="btn btn-ghost btn-xs btn-square"
                            title={t("podcasts.download", "Download")}
                            disabled={ep.downloaded || downloadMutation.isPending}
                            onClick={() => handleDownload(ep.id)}
                          >
                            <Download className="h-4 w-4" />
                          </button>
                        </td>
                      </tr>
                    ))}
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
          message={t("podcasts.delete_confirm", "Delete this podcast and its episodes? Download episodes already imported as books are kept.")}
          onClose={() => setDeleteTarget(null)}
          onConfirm={() => {
            deleteMutation.mutate(deleteTarget.id, {
              onSuccess: () => {
                toast.success(t("podcasts.deleted", "Podcast deleted"));
                if (activePodcastId === deleteTarget.id) setActivePodcastId(podcasts[0]?.id ?? null);
                setDeleteTarget(null);
              },
              onError: (err) => toast.error(err instanceof Error ? err.message : t("podcasts.delete_failed", "Failed to delete podcast")),
            });
          }}
          loading={deleteMutation.isPending}
        />
      )}
    </div>
  );
};