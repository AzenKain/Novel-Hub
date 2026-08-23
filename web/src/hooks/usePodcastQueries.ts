import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { podcastService } from "@/services";
import type { Podcast, PodcastEpisode, SubscribePodcastInput, UpdatePodcastInput, DownloadEpisodeParams } from "@/types";
import { usePodcastDownloadStore } from "@/stores";
import i18n from "@/i18n";
import { toast } from "react-toastify";

export function usePodcastsQuery() {
  return useQuery<Podcast[]>({
    queryKey: ["podcasts"],
    queryFn: async () => {
      const res = await podcastService.listPodcasts();
      if (!res.status) throw new Error(res.message || i18n.t("podcasts.load_failed", "Failed to load podcasts"));
      return res.data || [];
    },
  });
}

export function usePodcastEpisodesQuery(podcastId: string) {
  const finishDownload = usePodcastDownloadStore((state) => state.finishDownload);
  const queryClient = useQueryClient();

  return useQuery<PodcastEpisode[]>({
    queryKey: ["podcastEpisodes", podcastId],
    queryFn: async () => {
      const res = await podcastService.listEpisodes(podcastId);
      if (!res.status) throw new Error(res.message || i18n.t("podcasts.load_failed", "Failed to load episodes"));
      const episodes = res.data || [];

      // Check if any active download finished
      const activeDownloads = usePodcastDownloadStore.getState().activeDownloads;
      episodes.forEach((ep) => {
        if (ep.downloaded && activeDownloads[ep.id]) {
          const title = activeDownloads[ep.id].episodeTitle || ep.title;
          finishDownload(ep.id);
          toast.success(
            i18n.t("podcasts.download_completed", 'Episode "{{title}}" downloaded successfully', { title }),
            { toastId: `podcast-download-${ep.id}` }
          );
          void queryClient.invalidateQueries({ queryKey: ["podcasts"] });
          void queryClient.invalidateQueries({ queryKey: ["books"] });
        }
      });

      return episodes;
    },
    enabled: !!podcastId,
    refetchInterval: () => (usePodcastDownloadStore.getState().hasActiveDownloads(podcastId) ? 2000 : false),
    refetchOnMount: "always",
  });
}

export function useSubscribePodcastMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: SubscribePodcastInput) => {
      const res = await podcastService.subscribe(input);
      if (!res.status) throw new Error(res.message || i18n.t("podcasts.subscribe_failed", "Failed to subscribe"));
      return res.data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["podcasts"] });
    },
  });
}

export function useUpdatePodcastMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, input }: { id: string; input: UpdatePodcastInput }) => {
      const res = await podcastService.updatePodcast(id, input);
      if (!res.status) throw new Error(res.message || i18n.t("podcasts.update_failed", "Failed to update podcast"));
      return res.data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["podcasts"] });
    },
  });
}

export function useDeletePodcastMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await podcastService.deletePodcast(id);
      if (!res.status) throw new Error(res.message || i18n.t("podcasts.delete_failed", "Failed to delete podcast"));
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["podcasts"] });
    },
  });
}

export function useRefreshPodcastMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await podcastService.refreshPodcast(id);
      if (!res.status) throw new Error(res.message || i18n.t("podcasts.refresh_failed", "Failed to refresh podcast"));
      return res.data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["podcasts"] });
    },
  });
}

export function useDownloadEpisodeMutation(podcastId: string) {
  const queryClient = useQueryClient();
  const startDownload = usePodcastDownloadStore((state) => state.startDownload);

  return useMutation({
    mutationFn: async (params: string | DownloadEpisodeParams) => {
      const episodeId = typeof params === "string" ? params : params.episodeId;
      const episodeTitle = typeof params === "string" ? undefined : params.episodeTitle;

      startDownload(podcastId, episodeId, episodeTitle);

      const res = await podcastService.downloadEpisode(podcastId, episodeId);
      if (!res.status) throw new Error(res.message || i18n.t("podcasts.download_failed", "Failed to enqueue download"));
      return res.data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["podcastEpisodes", podcastId] });
      void queryClient.invalidateQueries({ queryKey: ["books"] });
    },
  });
}