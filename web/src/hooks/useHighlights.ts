import { highlightService } from '@/services';
import type { Highlight } from '@/types';
import { useAuthStore } from '@/stores';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useShallow } from 'zustand/react/shallow';

export const useHighlights = (bookId: string, chapterId: string | undefined) => {
  const queryClient = useQueryClient();
  const { user } = useAuthStore(useShallow((state) => ({ user: state.user })));

  const highlightsQuery = useQuery<Highlight[]>({
    queryKey: ['highlights', chapterId],
    queryFn: async () => {
      if (!chapterId) return [];
      const data = await highlightService.getHighlights(chapterId);
      return Array.isArray(data) ? data : [];
    },
    enabled: Boolean(chapterId && user),
  });

  const addMutation = useMutation({
    mutationFn: async ({ textContent, startIndex, endIndex, color }: { textContent: string; startIndex: number; endIndex: number; color: string }) => {
      if (!chapterId || !bookId) throw new Error("Missing chapterId or bookId");
      return await highlightService.createHighlight(bookId, chapterId, textContent, startIndex, endIndex, color);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['highlights', chapterId] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      await highlightService.deleteHighlight(id);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['highlights', chapterId] });
    },
  });

  const addHighlight = async (textContent: string, startIndex: number, endIndex: number, color: string = 'yellow') => {
    try {
      return await addMutation.mutateAsync({ textContent, startIndex, endIndex, color });
    } catch (err) {
      console.error("Failed to create highlight", err);
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
    removeHighlight,
    loading: highlightsQuery.isLoading || addMutation.isPending || deleteMutation.isPending,
  };
};
