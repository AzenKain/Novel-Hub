import { bookService, featureService } from "@/services";
import type { Book, SearchBookParams } from "@/types";
import { useQuery } from "@tanstack/react-query";

export function useBooksQuery(params: SearchBookParams, enabled = true) {
  return useQuery({
    queryKey: ["books", params],
    queryFn: async () => {
      const res = await bookService.getBooks(params);
      if (!res.status) throw new Error(res.message || "Failed to fetch books");
      return res.data || [];
    },
    enabled,
  });
}

export function useBookmarkedBooksQuery(enabled = true) {
  return useQuery<Book[]>({
    queryKey: ["books", "bookmarked"],
    queryFn: async () => {
      const res = await featureService.getBookmarkedBooks(200, 0);
      if (!res.status) throw new Error(res.message || "Failed to fetch bookmarked books");
      return res.data || [];
    },
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
