import { bookService, featureService } from "@/services";
import type { Book, SearchBookParams } from "@/types";
import { useQuery, useInfiniteQuery } from "@tanstack/react-query";

export function useBooksQuery(params: SearchBookParams, enabled = true) {
  return useInfiniteQuery({
    queryKey: ["books", params],
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) => {
      const res = await bookService.getBooks({ ...params, cursor: pageParam });
      if (!res.status) throw new Error(res.message || "Failed to fetch books");
      return res;
    },
    getNextPageParam: (lastPage) => lastPage.next_cursor || undefined,
    enabled,
  });
}

export function useBookmarkedBooksQuery(enabled = true) {
  return useInfiniteQuery({
    queryKey: ["books", "bookmarked"],
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) => {
      const res = await featureService.getBookmarkedBooks(pageParam, 20);
      if (!res.status) throw new Error(res.message || "Failed to fetch bookmarked books");
      return res;
    },
    getNextPageParam: (lastPage) => lastPage.next_cursor || undefined,
    enabled,
  });
}

export function useHotBooksQuery(limit = 6) {
  return useQuery<Book[]>({
    queryKey: ["books", "hot", limit],
    queryFn: async () => {
      const res = await bookService.getBooks({ nav: "hot", limit });
      if (!res.status) throw new Error(res.message || "Failed to fetch hot books");
      return res.data || [];
    },
    staleTime: 1000 * 60 * 10, // 10 minutes cache for hot books
  });
}

export function useRandomBooksQuery(limit = 6) {
  return useQuery<Book[]>({
    queryKey: ["books", "random", limit],
    queryFn: async () => {
      const res = await bookService.getBooks({ nav: "random", limit });
      if (!res.status) throw new Error(res.message || "Failed to fetch random books");
      return res.data || [];
    },
    refetchOnWindowFocus: false,
    staleTime: 0, // Random books can be refetched on demand
  });
}
