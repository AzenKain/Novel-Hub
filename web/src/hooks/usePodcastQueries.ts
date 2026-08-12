import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { podcastService } from "@/services";
import type { Podcast, PodcastEpisode, SubscribePodcastInput, UpdatePodcastInput } from "@/types";
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
  return useQuery<PodcastEpisode[]>({
    queryKey: ["podcastEpisodes", podcastId],
    queryFn: async () => {
      const res = await podcastService.listEpisodes(podcastId);
      if (!res.status) throw new Error(res.message || i18n.t("podcasts.load_failed", "Failed to load episodes"));
      return res.data || [];
    },
    enabled: !!podcastId,
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
  return useMutation({
    mutationFn: async (episodeId: string) => {
      const res = await podcastService.downloadEpisode(podcastId, episodeId);
      if (!res.status) throw new Error(res.message || i18n.t("podcasts.download_failed", "Failed to enqueue download"));
      return res.data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["podcastEpisodes", podcastId] });
    },
  });
}