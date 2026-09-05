import { readListService } from "@/services";
import type {
  ImportCBLResult,
  ReadList,
  ReadListBook,
  ReadListNext,
} from "@/types";
import i18n from "@/i18n";
import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { toast } from "react-toastify";

const readListKeys = {
  lists: ["read-lists"] as const,
  detail: (id: string) => ["read-list", id] as const,
  books: (id: string) => ["read-list", id, "books"] as const,
};

// Invalidate card grid and ordered items together.
function useReadListInvalidation() {
  const queryClient = useQueryClient();
  return (id?: string) => {
    void queryClient.invalidateQueries({ queryKey: readListKeys.lists });
    if (id) void queryClient.invalidateQueries({ queryKey: ["read-list", id] });
  };
}

export function useReadListsQuery(enabled = true) {
  return useInfiniteQuery({
    queryKey: readListKeys.lists,
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) => {
      const res = await readListService.getReadLists(pageParam, 50);
      if (!res.status)
        throw new Error(res.message || "Failed to fetch read lists");
      return res;
    },
    getNextPageParam: (lastPage) => lastPage.next_cursor || undefined,
    enabled,
  });
}

export function useReadListQuery(id: string | undefined) {
  return useQuery<ReadList | undefined>({
    queryKey: readListKeys.detail(id || ""),
    queryFn: async () => {
      const res = await readListService.getReadList(id as string);
      if (!res.status)
        throw new Error(res.message || "Failed to fetch the read list");
      return res.data;
    },
    enabled: !!id,
  });
}

export function useReadListBooksQuery(id: string | undefined) {
  return useQuery<ReadListBook[]>({
    queryKey: readListKeys.books(id || ""),
    queryFn: async () => {
      const res = await readListService.getReadListBooks(id as string);
      if (!res.status)
        throw new Error(res.message || "Failed to fetch the read list books");
      return res.data || [];
    },
    enabled: !!id,
  });
}

export function useReadListNextQuery(
  readListId: string | undefined,
  afterBookId: string | undefined,
) {
  return useQuery<ReadListNext | undefined>({
    queryKey: ["read-list", readListId || "", "next", afterBookId || ""],
    queryFn: async () => {
      const res = await readListService.nextInOrder(
        readListId as string,
        afterBookId,
      );
      if (!res.status)
        throw new Error(
          res.message || "Failed to fetch the next book in the read list",
        );
      return res.data;
    },
    enabled: !!readListId && !!afterBookId,
  });
}

export function useCreateReadListMutation() {
  const invalidate = useReadListInvalidation();
  return useMutation({
    mutationFn: async ({
      name,
      description,
    }: {
      name: string;
      description?: string;
    }) => {
      const res = await readListService.createReadList(name, description);
      if (!res.status)
        throw new Error(res.message || "Failed to create the read list");
      return res.data;
    },
    onSuccess: () => {
      invalidate();
      toast.success(
        i18n.t("library.readlist_created", "Read list created successfully"),
      );
    },
    onError: (err) => {
      toast.error(
        err instanceof Error
          ? err.message
          : i18n.t(
              "library.readlist_create_failed",
              "Could not create the read list",
            ),
      );
    },
  });
}

export function useUpdateReadListMutation() {
  const invalidate = useReadListInvalidation();
  return useMutation({
    mutationFn: async ({
      id,
      name,
      description,
    }: {
      id: string;
      name: string;
      description?: string;
    }) => {
      const res = await readListService.updateReadList(id, name, description);
      if (!res.status)
        throw new Error(res.message || "Failed to rename the read list");
      return res.data;
    },
    onSuccess: (_data, variables) => {
      invalidate(variables.id);
      toast.success(
        i18n.t("library.readlist_updated", "Read list updated successfully"),
      );
    },
    onError: (err) => {
      toast.error(
        err instanceof Error
          ? err.message
          : i18n.t(
              "library.readlist_update_failed",
              "Could not update the read list",
            ),
      );
    },
  });
}

export function useDeleteReadListMutation() {
  const invalidate = useReadListInvalidation();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await readListService.deleteReadList(id);
      if (!res.status)
        throw new Error(res.message || "Failed to delete the read list");
      return res;
    },
    onSuccess: (_data, id) => {
      invalidate(id);
      toast.success(
        i18n.t("library.readlist_deleted", "Read list deleted successfully"),
      );
    },
    onError: (err) => {
      toast.error(
        err instanceof Error
          ? err.message
          : i18n.t(
              "library.readlist_delete_failed",
              "Could not delete the read list",
            ),
      );
    },
  });
}

export function useAddReadListBookMutation() {
  const invalidate = useReadListInvalidation();
  return useMutation({
    mutationFn: async ({ id, bookId }: { id: string; bookId: string }) => {
      const res = await readListService.addBook(id, bookId);
      if (!res.status) throw new Error(res.message || "Failed to add the book");
      return res;
    },
    onSuccess: (_data, variables) => {
      invalidate(variables.id);
      toast.success(
        i18n.t(
          "library.readlist_add_book_success",
          "Book added to the read list",
        ),
      );
    },
    onError: (err) => {
      toast.error(
        err instanceof Error
          ? err.message
          : i18n.t(
              "library.readlist_add_book_failed",
              "Could not add the book to the read list",
            ),
      );
    },
  });
}

export function useRemoveReadListBookMutation() {
  const invalidate = useReadListInvalidation();
  return useMutation({
    mutationFn: async ({ id, bookId }: { id: string; bookId: string }) => {
      const res = await readListService.removeBook(id, bookId);
      if (!res.status)
        throw new Error(res.message || "Failed to remove the book");
      return res;
    },
    onSuccess: (_data, variables) => {
      invalidate(variables.id);
      toast.success(
        i18n.t(
          "library.readlist_remove_book_success",
          "Book removed from the read list",
        ),
      );
    },
    onError: (err) => {
      toast.error(
        err instanceof Error
          ? err.message
          : i18n.t(
              "library.readlist_remove_book_failed",
              "Could not remove the book from the read list",
            ),
      );
    },
  });
}

export function useReorderReadListMutation() {
  const invalidate = useReadListInvalidation();
  return useMutation({
    mutationFn: async ({ id, bookIds }: { id: string; bookIds: string[] }) => {
      const res = await readListService.reorder(id, bookIds);
      if (!res.status)
        throw new Error(res.message || "Failed to reorder the read list");
      return res;
    },
    onSuccess: (_data, variables) => invalidate(variables.id),
    onError: (err) => {
      toast.error(
        err instanceof Error
          ? err.message
          : i18n.t(
              "library.readlist_reorder_failed",
              "Could not save the new order",
            ),
      );
    },
  });
}

export function useImportCBLMutation(
  onImported?: (result: ImportCBLResult) => void,
) {
  const invalidate = useReadListInvalidation();
  return useMutation({
    mutationFn: async (file: File) => {
      const res = await readListService.importCBL(file);
      if (!res.status || !res.data)
        throw new Error(res.message || "Failed to import the .cbl file");
      return res.data;
    },
    onSuccess: (result) => {
      invalidate(result.read_list.id);
      onImported?.(result);
    },
    onError: (err) => {
      toast.error(
        err instanceof Error
          ? err.message
          : i18n.t(
              "library.readlist_import_failed",
              "Could not import the .cbl file",
            ),
      );
    },
  });
}
