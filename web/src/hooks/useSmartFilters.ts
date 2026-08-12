import { smartFilterService } from "@/services";
import type { SmartFilter, UpsertSmartFilterPayload, ReorderHomeShelfItem, Book } from "@/types";
import { useQuery, useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "react-toastify";
import i18n from "@/i18n";

export function useSmartFiltersQuery() {
  return useQuery<SmartFilter[]>({
    queryKey: ["smart-filters"],
    queryFn: async () => {
      const res = await smartFilterService.list();
      if (!res.status) throw new Error(res.message || "Failed to fetch smart filters");
      return res.data || [];
    },
  });
}

export function useSmartFilterBooksInfiniteQuery(id: string, libraryId?: string, limit = 20, enabled = true) {
  return useInfiniteQuery({
    queryKey: ["smart-filters", id, "books", { libraryId, limit }],
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) => {
      const res = await smartFilterService.getBooks(id, {
        library_id: libraryId,
        cursor: pageParam,
        limit,
      });
      if (!res.status) throw new Error("Failed to fetch books for smart filter");
      return res;
    },
    getNextPageParam: (lastPage) => lastPage.next_cursor || undefined,
    enabled: enabled && !!id,
  });
}

export function useSmartFilterBooksQuery(id: string, libraryId?: string, limit = 8, enabled = true) {
  return useQuery<Book[]>({
    queryKey: ["smart-filters", id, "books", "fixed", { libraryId, limit }],
    queryFn: async () => {
      const res = await smartFilterService.getBooks(id, {
        library_id: libraryId,
        limit,
      });
      if (!res.status) throw new Error("Failed to fetch books for smart filter");
      return res.data || [];
    },
    enabled: enabled && !!id,
  });
}

export function useCreateSmartFilterMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (payload: UpsertSmartFilterPayload) => {
      const res = await smartFilterService.create(payload);
      if (!res.status) throw new Error(res.message || "Failed to create smart filter");
      return res.data!;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["smart-filters"] });
      toast.success(i18n.t("library.smart_filter_created", "Smart filter created successfully"));
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : i18n.t("library.smart_filter_create_failed", "Failed to create smart filter"));
    },
  });
}

export function useUpdateSmartFilterMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, payload }: { id: string; payload: UpsertSmartFilterPayload }) => {
      const res = await smartFilterService.update(id, payload);
      if (!res.status) throw new Error(res.message || "Failed to update smart filter");
      return res.data!;
    },
    onSuccess: (data) => {
      void queryClient.invalidateQueries({ queryKey: ["smart-filters"] });
      void queryClient.invalidateQueries({ queryKey: ["smart-filters", data.id] });
      toast.success(i18n.t("library.smart_filter_updated", "Smart filter updated successfully"));
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : i18n.t("library.smart_filter_update_failed", "Failed to update smart filter"));
    },
  });
}

export function useDeleteSmartFilterMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await smartFilterService.delete(id);
      if (!res.status) throw new Error(res.message || "Failed to delete smart filter");
      return res;
    },
    onSuccess: (_, id) => {
      void queryClient.invalidateQueries({ queryKey: ["smart-filters"] });
      void queryClient.invalidateQueries({ queryKey: ["smart-filters", id] });
      toast.success(i18n.t("library.smart_filter_deleted", "Smart filter deleted successfully"));
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : i18n.t("library.smart_filter_delete_failed", "Failed to delete smart filter"));
    },
  });
}

export function usePinSmartFilterSidebarMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, isPinned }: { id: string; isPinned: boolean }) => {
      const res = await smartFilterService.pinSidebar(id, isPinned);
      if (!res.status) throw new Error(res.message || "Failed to pin smart filter to sidebar");
      return res.data!;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["smart-filters"] });
    },
  });
}

export function usePinSmartFilterHomeMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, isPinned }: { id: string; isPinned: boolean }) => {
      const res = await smartFilterService.pinHome(id, isPinned);
      if (!res.status) throw new Error(res.message || "Failed to pin smart filter to homepage");
      return res.data!;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["smart-filters"] });
    },
  });
}

export function useReorderSmartFiltersHomeMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (shelves: ReorderHomeShelfItem[]) => {
      const res = await smartFilterService.reorderHome(shelves);
      if (!res.status) throw new Error(res.message || "Failed to reorder homepage shelves");
      return res;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["smart-filters"] });
    },
  });
}
