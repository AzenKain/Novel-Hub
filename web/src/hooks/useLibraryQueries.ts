import { adminService, bookService, featureService, libraryService } from "@/services";
import type { Collection, DuplicateGroupResult, Library, LibraryStats, ReadingHistory, SmartCollection, SmartCollectionRule } from "@/types";
import i18n from "@/i18n";
import { useQuery, useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "react-toastify";

export function useLibrariesQuery() {
  return useQuery<Library[]>({
    queryKey: ["libraries"],
    queryFn: async () => {
      const res = await libraryService.getLibraries();
      if (!res.status) throw new Error(res.message || "Failed to fetch libraries");
      return res.data || [];
    },
  });
}

export function useLibraryStatsQuery() {
  return useQuery<LibraryStats>({
    queryKey: ["library", "stats"],
    queryFn: async () => {
      const res = await featureService.getLibraryStats();
      if (!res.status) throw new Error(res.message || "Failed to fetch library stats");
      return res.data || { total_books: 0, need_review: 0, series_tracked: 0 };
    },
  });
}

export function useCollectionsQuery(enabled = true) {
  return useInfiniteQuery<{ data: Collection[], nextCursor: string | null }>({
    queryKey: ["collections"],
    initialPageParam: undefined,
    queryFn: async ({ pageParam }) => {
      const cursor = pageParam as string | undefined;
      const res = await featureService.getCollections(cursor, 50);
      if (!res.status) throw new Error(res.message || "Failed to fetch collections");
      
      const collections = res.data || [];
      const nextCursor = collections.length === 50 ? collections[collections.length - 1].created_at : null;
      
      return { data: collections, nextCursor };
    },
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    enabled,
  });
}

export function useSmartCollectionsQuery(enabled = true) {
  return useQuery<SmartCollection[]>({
    queryKey: ["smart-collections"],
    queryFn: async () => {
      const res = await featureService.getSmartCollections();
      if (!res.status) throw new Error(res.message || "Failed to fetch smart collections");
      return res.data || [];
    },
    enabled,
  });
}

export function useCreateSmartCollectionMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ name, rule }: { name: string; rule: SmartCollectionRule }) => {
      const res = await featureService.createSmartCollection(name, rule);
      if (!res.status) throw new Error(res.message || "Failed to create smart collection");
      return res.data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["smart-collections"] });
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : i18n.t("library.smart_collection_create_failed", "Could not save the smart collection"));
    },
  });
}

export function useDeleteSmartCollectionMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await featureService.deleteSmartCollection(id);
      if (!res.status) throw new Error(res.message || "Failed to delete smart collection");
      return res;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["smart-collections"] });
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : i18n.t("library.smart_collection_delete_failed", "Could not delete the smart collection"));
    },
  });
}

export function useReadingHistoryQuery(enabled = true) {  return useInfiniteQuery({
    queryKey: ["reading", "history"],
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) => {
      const res = await featureService.getRecentReadingHistory(pageParam, 20);
      if (!res.status) throw new Error(res.message || "Failed to fetch reading history");
      return res;
    },
    getNextPageParam: (lastPage) => lastPage.next_cursor || undefined,
    enabled,
    staleTime: 0,
  });
}

export function useDuplicatesQuery() {
  return useQuery<DuplicateGroupResult[]>({
    queryKey: ["duplicates"],
    queryFn: async () => {
      const res = await bookService.getDuplicates();
      if (!res.status) throw new Error(res.message || "Failed to fetch duplicates");
      return res.data || [];
    },
  });
}

export function useDeleteBookFileMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (file_id: string) => {
      const res = await adminService.deleteBookFile(file_id);
      if (!res.status) throw new Error(res.message || "Failed to delete file");
      return res;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["duplicates"] });
      void queryClient.invalidateQueries({ queryKey: ["books"] });
      void queryClient.invalidateQueries({ queryKey: ["library"] });
    },
  });
}
