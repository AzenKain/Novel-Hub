import { bookService, featureService } from "@/services";
import type { Book, SearchBookParams } from "@/types";
import { useQuery, useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";

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
    staleTime: 1000 * 60, // 1 minute cache for hot books
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

export function useSendBookToEmailMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ bookId, recipientEmail }: { bookId: string; recipientEmail: string }) => {
      const res = await bookService.sendToEmail(bookId, recipientEmail);
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
    },
  });
}

export function useBulkMoveBooksMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ bookIds, targetLibraryId }: { bookIds: string[]; targetLibraryId: string }) => {
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
    mutationFn: async ({ bookIds, collectionIds }: { bookIds: string[]; collectionIds: string[] }) => {
      const res = await bookService.bulkAssignCollections(bookIds, collectionIds);
      if (!res.status) throw new Error(res.message || "Failed to assign collections");
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
    mutationFn: async ({ bookIds, tagNames }: { bookIds: string[]; tagNames: string[] }) => {
      const res = await bookService.bulkAddTags(bookIds, tagNames);
      if (!res.status) throw new Error(res.message || "Failed to add tags");
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["books"] });
    },
  });
}

export function useBookQuery(bookId: string) {
  return useQuery({
    queryKey: ["book", bookId],
    queryFn: async () => {
      if (!bookId) throw new Error("No book ID");
      const res = await bookService.getBook(bookId);
      if (!res.status) throw new Error(res.message || "Failed to fetch book");
      return res.data;
    },
    enabled: !!bookId,
  });
}

export function useBookUserStateQuery(bookId: string, enabled = true) {
  return useQuery({
    queryKey: ["bookUserState", bookId],
    queryFn: async () => {
      if (!bookId) throw new Error("No book ID");
      const res = await featureService.getBookUserState(bookId);
      if (!res.status) throw new Error(res.message || "Failed to fetch user state");
      return res.data;
    },
    enabled: !!bookId && enabled,
    retry: false,
  });
}

export function useBookEngagementStatsQuery(bookId: string) {
  return useQuery({
    queryKey: ["bookEngagement", bookId],
    queryFn: async () => {
      if (!bookId) throw new Error("No book ID");
      const res = await featureService.getBookEngagementStats(bookId);
      return res.status ? res.data : null;
    },
    enabled: !!bookId,
    retry: false,
  });
}

export function useToggleBookmarkMutation(bookId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (bookmarked: boolean) => {
      if (!bookId) throw new Error("No book ID");
      const res = await featureService.setBookmark(bookId, bookmarked);
      if (!res.status) throw new Error(res.message || "Failed to toggle bookmark");
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["bookUserState", bookId] });
      queryClient.invalidateQueries({ queryKey: ["books"] });
    },
  });
}

export function useAddBookToCollectionMutation(bookId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (collectionId: string) => {
      if (!bookId) throw new Error("No book ID");
      return featureService.addBookToCollection(collectionId, bookId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["bookUserState", bookId] });
      queryClient.invalidateQueries({ queryKey: ["collections"] });
      queryClient.invalidateQueries({ queryKey: ["books"] });
    },
  });
}

export function useRemoveBookFromCollectionMutation(bookId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (collectionId: string) => {
      if (!bookId) throw new Error("No book ID");
      return featureService.removeBookFromCollection(collectionId, bookId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["bookUserState", bookId] });
      queryClient.invalidateQueries({ queryKey: ["collections"] });
      queryClient.invalidateQueries({ queryKey: ["books"] });
    },
  });
}
