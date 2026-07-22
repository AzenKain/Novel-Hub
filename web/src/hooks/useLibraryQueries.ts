import { adminService, bookService, featureService, libraryService } from "@/services";
import type { Collection, DuplicateGroupResult, Library, LibraryStats, ReadingHistory } from "@/types";
import { useQuery, useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";

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
      return res.data || { totalBooks: 0, needReview: 0, seriesTracked: 0 };
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
      const nextCursor = collections.length === 50 ? collections[collections.length - 1].createdAt : null;
      
      return { data: collections, nextCursor };
    },
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    enabled,
  });
}

export function useReadingHistoryQuery(enabled = true) {
  return useInfiniteQuery({
    queryKey: ["reading", "history"],
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) => {
      const res = await featureService.getRecentReadingHistory(pageParam, 20);
      if (!res.status) throw new Error(res.message || "Failed to fetch reading history");
      return res;
    },
    getNextPageParam: (lastPage) => lastPage.next_cursor || undefined,
    enabled,
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
    mutationFn: async (fileId: string) => {
      const res = await adminService.deleteBookFile(fileId);
      if (!res.status) throw new Error(res.message || "Failed to delete file");
      return res;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["duplicates"] });
    },
  });
}
