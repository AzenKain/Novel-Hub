import { bookService, featureService, libraryService } from "@/services";
import type { Collection, DuplicateFileResult, Library, LibraryStats, ReadingHistory } from "@/types";
import { useQuery } from "@tanstack/react-query";

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
  return useQuery<Collection[]>({
    queryKey: ["collections"],
    queryFn: async () => {
      const res = await featureService.getCollections();
      if (!res.status) throw new Error(res.message || "Failed to fetch collections");
      return res.data || [];
    },
    enabled,
  });
}

export function useReadingHistoryQuery(enabled = true) {
  return useQuery<ReadingHistory[]>({
    queryKey: ["reading", "history"],
    queryFn: async () => {
      const res = await featureService.getRecentReadingHistory();
      if (!res.status) throw new Error(res.message || "Failed to fetch reading history");
      return res.data || [];
    },
    enabled,
  });
}

export function useDuplicatesQuery() {
  return useQuery<DuplicateFileResult[]>({
    queryKey: ["duplicates"],
    queryFn: async () => {
      const res = await bookService.getDuplicates();
      if (!res.status) throw new Error(res.message || "Failed to fetch duplicates");
      return res.data || [];
    },
  });
}
