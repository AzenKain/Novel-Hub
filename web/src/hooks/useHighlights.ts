import { highlightService } from '@/services';
import type { Highlight } from '@/types';
import { useAuthStore } from '@/stores';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useShallow } from 'zustand/react/shallow';

export const useHighlights = (book_id: string, chapter_id: string | undefined, enabled = true) => {
  const queryClient = useQueryClient();
  const { user } = useAuthStore(useShallow((state) => ({ user: state.user })));

  const highlightsQuery = useQuery<Highlight[]>({
    queryKey: ['highlights', chapter_id],
    queryFn: async () => {
      if (!chapter_id) return [];
      const data = await highlightService.getHighlights(chapter_id);
      return Array.isArray(data) ? data : [];
    },
    enabled: Boolean(enabled && chapter_id && user),
  });

  const addMutation = useMutation({
    mutationFn: async ({ text_content, start_index, end_index, color }: { text_content: string; start_index: number; end_index: number; color: string }) => {
      if (!chapter_id || !book_id) throw new Error("Missing chapter_id or book_id");
      return await highlightService.createHighlight(book_id, chapter_id, text_content, start_index, end_index, color);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['highlights', chapter_id] });
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({ id, color, note }: { id: string; color: string; note?: string }) => {
      return await highlightService.updateHighlightNote(id, color, note);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['highlights', chapter_id] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      await highlightService.deleteHighlight(id);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['highlights', chapter_id] });
    },
  });

  const addHighlight = async (text_content: string, start_index: number, end_index: number, color: string = 'yellow') => {
    try {
      return await addMutation.mutateAsync({ text_content, start_index, end_index, color });
    } catch (err) {
      console.error("Failed to create highlight", err);
      return null;
    }
  };

  const updateHighlight = async (id: string, color: string, note?: string) => {
    try {
      return await updateMutation.mutateAsync({ id, color, note });
    } catch (err) {
      console.error("Failed to update highlight", err);
      return null;
    }
  };

  const removeHighlight = async (id: string) => {
    try {
      await deleteMutation.mutateAsync(id);
    } catch (err) {
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
