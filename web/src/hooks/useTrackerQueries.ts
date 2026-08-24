import { trackerService, featureService } from "@/services";
import type {
  ConnectTrackerInput,
  MapTrackerInput,
  SyncProgressInput,
  TrackerConnection,
  TrackerSearchResult,
} from "@/types";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export function useTrackerConnectionsQuery(enabled = true) {
  return useQuery<TrackerConnection[]>({
    queryKey: ["user", "trackers", "connections"],
    queryFn: async () => {
      const res = await trackerService.getConnections();
      if (!res.status) throw new Error(res.message || "Failed to load tracker connections");
      return res.data ?? [];
    },
    enabled,
    staleTime: 30_000,
    retry: false,
  });
}

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

export function useTrackerReadingProgressQuery(book_id: string, enabled = true) {
  return useQuery({
    queryKey: ["trackerReadingProgress", book_id],
    queryFn: async () => {
      const res = await featureService.getReadingProgress(book_id);
      return (res && res.status && res.data) ? res.data : null;
    },
    enabled: !!book_id && enabled,
    staleTime: 0,
    refetchOnMount: "always",
    retry: false,
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

export function useConnectHardcoverMutation() {
  return useMutation({
    mutationFn: async () => {
      const res = await trackerService.connectHardcover();
      if (!res.status || !res.data?.authorize_url) throw new Error(res.message || "Failed to start Hardcover connect");
      return res.data.authorize_url;
    },
  });
}

export function useSyncHardcoverMutation() {
  return useMutation<void, Error, { book_id: string; progress: number }>({
    mutationFn: async (input) => {
      const res = await trackerService.syncHardcoverProgress(input.book_id, input.progress);
      if (!res.status) throw new Error(res.message || "Failed to sync Hardcover progress");
    },
  });
}

export function useExportHighlightsToReadwiseMutation() {
  return useMutation<{ exported: number }, Error, string>({
    mutationFn: async (book_id: string) => {
      const res = await trackerService.exportHighlightsToReadwise(book_id);
      if (!res.status || !res.data) throw new Error(res.message || "Failed to export highlights");
      return res.data;
    },
  });
}
