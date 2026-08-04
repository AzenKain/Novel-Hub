import { highlightService } from '@/services';
import type { Highlight } from '@/types';
import i18n from '@/i18n';
import { useAuthStore } from '@/stores';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'react-toastify';
import { useShallow } from 'zustand/react/shallow';

export const useHighlights = (book_id: string, chapter_id: string | undefined, enabled = true) => {
  const queryClient = useQueryClient();
  const { user } = useAuthStore(useShallow((state) => ({ user: state.user })));

  const highlightsQuery = useQuery<Highlight[]>({
    queryKey: ['highlights', chapter_id],
    queryFn: async () => {
      if (!chapter_id) return [];
      const res = await highlightService.getHighlights(chapter_id);
      if (!res.status) throw new Error(res.message || "Failed to fetch highlights");
      return Array.isArray(res.data) ? res.data : [];
    },
    enabled: Boolean(enabled && chapter_id && user),
  });

  const addMutation = useMutation({
    mutationFn: async ({ text_content, start_index, end_index, color }: { text_content: string; start_index: number; end_index: number; color: string }) => {
      if (!chapter_id || !book_id) throw new Error("Missing chapter_id or book_id");
      const res = await highlightService.createHighlight(book_id, chapter_id, text_content, start_index, end_index, color);
      if (!res.status) throw new Error(res.message || "Failed to create highlight");
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['highlights', chapter_id] });
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({ id, color, note }: { id: string; color: string; note?: string }) => {
      const res = await highlightService.updateHighlightNote(id, color, note);
      if (!res.status) throw new Error(res.message || "Failed to update highlight");
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['highlights', chapter_id] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      const res = await highlightService.deleteHighlight(id);
      if (!res.status) throw new Error(res.message || "Failed to delete highlight");
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['highlights', chapter_id] });
    },
  });

  const addHighlight = async (text_content: string, start_index: number, end_index: number, color: string = 'yellow') => {
    try {
      return await addMutation.mutateAsync({ text_content, start_index, end_index, color });
    } catch (err) {
      toast.error(i18n.t('reader.highlight_create_failed', 'Could not save the highlight'));
      console.error("Failed to create highlight", err);
      return null;
    }
  };

  const updateHighlight = async (id: string, color: string, note?: string) => {
    try {
      return await updateMutation.mutateAsync({ id, color, note });
    } catch (err) {
      toast.error(i18n.t('reader.highlight_update_failed', 'Could not update the highlight'));
      console.error("Failed to update highlight", err);
      return null;
    }
  };

  const removeHighlight = async (id: string) => {
    try {
      await deleteMutation.mutateAsync(id);
    } catch (err) {
      toast.error(i18n.t('reader.highlight_delete_failed', 'Could not delete the highlight'));
      console.error("Failed to delete highlight", err);
    }
  };

  return {
    highlights: highlightsQuery.data || [],
    addHighlight,
    updateHighlight,
    removeHighlight,
    loading: highlightsQuery.isLoading || addMutation.isPending || updateMutation.isPending || deleteMutation.isPending,
  };
};
