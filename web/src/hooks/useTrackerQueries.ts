import { trackerService } from "@/services";
import type {
  ConnectTrackerInput,
  MapTrackerInput,
  SyncProgressInput,
  TrackerSearchResult,
} from "@/types";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function useConnectTrackerMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: ConnectTrackerInput) => {
      const res = await trackerService.connectTracker(input.provider, input.access_token);
      if (!res.status) throw new Error(res.message || "Failed to connect tracker");
      return res;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["user", "trackers"] });
    },
  });
}

export function useSearchTrackerMutation() {
  return useMutation<TrackerSearchResult, Error, string>({
    mutationFn: async (title: string) => {
      const res = await trackerService.searchAniList(title);
      if (!res.status || !res.data) throw new Error(res.message || "No AniList match found");
      return res.data;
    },
  });
}

export function useMapBookTrackerMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: MapTrackerInput) => {
      const res = await trackerService.mapBookTracker(input.book_id, input.provider, input.external_series_id);
      if (!res.status) throw new Error(res.message || "Failed to map book tracker");
      return res;
    },
  });
}

export function useSyncTrackerProgressMutation() {
  return useMutation({
    mutationFn: async (input: SyncProgressInput) => {
      const res = await trackerService.syncProgress(input.book_id, input.title, input.progress);
      if (!res.status) throw new Error(res.message || "Failed to sync reading progress");
      return res;
    },
  });
}
