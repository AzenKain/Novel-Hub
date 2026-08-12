import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { audiobookService } from "@/services";
import type { LookupAudiobookChaptersInput, MergeAudioInput, UpsertAudiobookChapterInput } from "@/types";

export function useAudiobookChaptersQuery(book_id: string) {
  return useQuery({
    queryKey: ["audiobookChapters", book_id],
    queryFn: async () => {
      const res = await audiobookService.listChapters(book_id);
      return res.status ? res.data : null;
    },
    enabled: !!book_id,
    staleTime: 0,
    refetchOnMount: "always",
    retry: false,
  });
}

export function useUpsertAudiobookChapterMutation(book_id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: { id?: string; chapter: UpsertAudiobookChapterInput }) =>
      audiobookService.upsertChapter(book_id, input.id, input.chapter),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["audiobookChapters", book_id] });
    },
  });
}

export function useDeleteAudiobookChapterMutation(book_id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => audiobookService.deleteChapter(book_id, id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["audiobookChapters", book_id] });
    },
  });
}

export function useLookupAudiobookChaptersMutation(book_id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: LookupAudiobookChaptersInput) =>
      audiobookService.lookupChapters(book_id, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["audiobookChapters", book_id] });
    },
  });
}

export function useMergeAudiobookMutation(book_id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: MergeAudioInput) => audiobookService.mergeAudio(book_id, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["book", book_id] });
    },
  });
}