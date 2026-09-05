import { bookService, featureService } from "@/services";
import type { Book, SearchBookParams } from "@/types";
import {
  useQuery,
  useInfiniteQuery,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";

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
    refetchInterval: (query) =>
      query.state.data?.pages.some((page) =>
        page.data?.some((book) => book.status === "processing"),
      )
        ? 3000
        : false,
    enabled,
  });
}

export function useBookmarkedBooksQuery(enabled = true) {
  return useInfiniteQuery({
    queryKey: ["books", "bookmarked"],
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) => {
      const res = await featureService.getBookmarkedBooks(pageParam, 60);
      if (!res.status)
        throw new Error(res.message || "Failed to fetch bookmarked books");
      return res;
    },
    getNextPageParam: (lastPage) => lastPage.next_cursor || undefined,
    enabled,
  });
}

export function useGuestBookmarkedBooksQuery(
  bookIds: string[],
  enabled = true,
) {
  return useQuery<Book[]>({
    queryKey: ["books", "guest-bookmarked", bookIds],
    queryFn: async () => {
      const res = await bookService.getBooks({ limit: 100 });
      if (!res.status) throw new Error(res.message || "Failed to fetch books");
      const idSet = new Set(bookIds);
      return (res.data || []).filter((b) => idSet.has(b.id));
    },
    enabled: enabled && bookIds.length > 0,
  });
}

export function useHotBooksQuery(limit = 6) {
  return useQuery<Book[]>({
    queryKey: ["books", "hot", limit],
    queryFn: async () => {
      const res = await bookService.getBooks({ nav: "hot", limit });
      if (!res.status)
        throw new Error(res.message || "Failed to fetch hot books");
      return res.data || [];
    },
    staleTime: 1000 * 60,
  });
}

export function useRandomBooksQuery(limit = 8) {
  return useQuery<Book[]>({
    queryKey: ["books", "random", limit],
    queryFn: async () => {
      const res = await bookService.getBooks({ nav: "random", limit });
      if (!res.status)
        throw new Error(res.message || "Failed to fetch random books");
      return res.data || [];
    },
    refetchOnWindowFocus: false,
    staleTime: 1000 * 30,
  });
}

export function useSendBookToEmailMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      book_id,
      recipientEmail,
    }: {
      book_id: string;
      recipientEmail: string;
    }) => {
      const res = await bookService.sendToEmail(book_id, recipientEmail);
      if (!res.status) throw new Error(res.message || "Failed to send email");
      return res;
    },
  });
}

export function useBulkDeleteBooksMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (bookIds: string[]) => {
      const res = await bookService.bulkDeleteBooks(bookIds);
      if (!res.status) throw new Error(res.message || "Failed to delete books");
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["books"] });
      void queryClient.invalidateQueries({ queryKey: ["podcasts"] });
      void queryClient.invalidateQueries({ queryKey: ["podcastEpisodes"] });
    },
  });
}

export function useBulkMoveBooksMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      bookIds,
      targetLibraryId,
    }: {
      bookIds: string[];
      targetLibraryId: string;
    }) => {
      const res = await bookService.bulkMoveBooks(bookIds, targetLibraryId);
      if (!res.status) throw new Error(res.message || "Failed to move books");
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["books"] });
    },
  });
}

export function useBulkAssignCollectionsMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      bookIds,
      collectionIds,
    }: {
      bookIds: string[];
      collectionIds: string[];
    }) => {
      const res = await bookService.bulkAssignCollections(
        bookIds,
        collectionIds,
      );
      if (!res.status)
        throw new Error(res.message || "Failed to assign collections");
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["books"] });
    },
  });
}

export function useBulkAddTagsMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      bookIds,
      tagNames,
    }: {
      bookIds: string[];
      tagNames: string[];
    }) => {
      const res = await bookService.bulkAddTags(bookIds, tagNames);
      if (!res.status) throw new Error(res.message || "Failed to add tags");
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["books"] });
    },
  });
}

export function useBulkUpdateMetadataMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (
      payload: import("@/types").BulkUpdateMetadataRequest,
    ) => {
      const res = await bookService.bulkUpdateMetadata(payload);
      if (!res.status)
        throw new Error(res.message || "Failed to update metadata");
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["books"] });
    },
  });
}

export function useBookQuery(book_id: string) {
  return useQuery({
    queryKey: ["book", book_id],
    queryFn: async () => {
      if (!book_id) throw new Error("No book ID");
      const res = await bookService.getBook(book_id);
      if (!res.status) throw new Error(res.message || "Failed to fetch book");
      return res.data;
    },
    enabled: !!book_id,
  });
}

export function useBookSeriesQuery(book_id: string) {
  return useQuery({
    queryKey: ["book", book_id, "series"],
    queryFn: async () => {
      const res = await bookService.getBookSeries(book_id);
      if (!res.status) throw new Error(res.message || "Failed to fetch series");
      return res.data;
    },
    enabled: !!book_id,
    retry: false,
  });
}

export function useSeriesBooksQuery(seriesId: string, enabled = true) {
  return useQuery({
    queryKey: ["books", "series", seriesId],
    queryFn: async () => {
      if (!seriesId) return [];
      const res = await bookService.getBooks({
        facet: "series",
        facet_id: seriesId,
        sort: "series_order",
        limit: 50,
      });
      if (!res.status)
        throw new Error(res.message || "Failed to fetch books in series");
      return res.data || [];
    },
    enabled: !!seriesId && enabled,
  });
}

export function useBookUserStateQuery(book_id: string, enabled = true) {
  return useQuery({
    queryKey: ["bookUserState", book_id],
    queryFn: async () => {
      if (!book_id) throw new Error("No book ID");
      const res = await featureService.getBookUserState(book_id);
      if (!res.status)
        throw new Error(res.message || "Failed to fetch user state");
      return res.data;
    },
    enabled: !!book_id && enabled,
    staleTime: 0,
    refetchOnMount: "always",
    retry: false,
  });
}

export function useBookEngagementStatsQuery(book_id: string, enabled = true) {
  return useQuery({
    queryKey: ["bookEngagement", book_id],
    queryFn: async () => {
      if (!book_id) throw new Error("No book ID");
      const res = await featureService.getBookEngagementStats(book_id);
      return res.status ? res.data : null;
    },
    enabled: !!book_id && enabled,
    retry: false,
  });
}

export function useToggleBookmarkMutation(book_id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (bookmarked: boolean) => {
      if (!book_id) throw new Error("No book ID");
      const res = await featureService.setBookmark(book_id, bookmarked);
      if (!res.status)
        throw new Error(res.message || "Failed to toggle bookmark");
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["bookUserState", book_id] });
      queryClient.invalidateQueries({ queryKey: ["books"] });
    },
  });
}

export function useAddBookToCollectionMutation(book_id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (collectionId: string) => {
      if (!book_id) throw new Error("No book ID");
      return featureService.addBookToCollection(collectionId, book_id);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["bookUserState", book_id] });
      queryClient.invalidateQueries({ queryKey: ["collections"] });
      queryClient.invalidateQueries({ queryKey: ["books"] });
    },
  });
}

export function useRemoveBookFromCollectionMutation(book_id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (collectionId: string) => {
      if (!book_id) throw new Error("No book ID");
      return featureService.removeBookFromCollection(collectionId, book_id);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["bookUserState", book_id] });
      queryClient.invalidateQueries({ queryKey: ["collections"] });
      queryClient.invalidateQueries({ queryKey: ["books"] });
    },
  });
}
