import { useEffect, useRef } from 'react';
import { readerService } from "@/services";
import { useAuthStore } from "@/stores";
import { useShallow } from "zustand/react/shallow";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ReadingGoal } from "@/types";

export const useReadingStats = (book_id: string | undefined, isActive: boolean) => {
  const { user } = useAuthStore(useShallow((state) => ({ user: state.user })));
  const statsRef = useRef(new Map<string, { duration: number; words: number }>());
  const syncInFlightRef = useRef(new Set<string>());
  const pendingSyncRef = useRef(new Set<string>());
  const lastSyncTimeRef = useRef(new Map<string, number>());
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const syncStats = async (bookIdToSync: string) => {
    const stats = statsRef.current.get(bookIdToSync);
    if (!stats || stats.duration < 1) return;
    if (syncInFlightRef.current.has(bookIdToSync)) {
      pendingSyncRef.current.add(bookIdToSync);
      return;
    }
    syncInFlightRef.current.add(bookIdToSync);
    lastSyncTimeRef.current.set(bookIdToSync, Date.now());
    const snapshot = { ...stats, words: Math.floor(stats.words) };
    try {
      await readerService.syncReadingSession(bookIdToSync, snapshot.duration, snapshot.words);
      const current = statsRef.current.get(bookIdToSync);
      if (current) {
        current.duration = Math.max(0, current.duration - snapshot.duration);
        current.words = Math.max(0, current.words - snapshot.words);
      }
    } catch (err) {
      console.error("Failed to sync reading stats", err);
    } finally {
      syncInFlightRef.current.delete(bookIdToSync);
      if (pendingSyncRef.current.delete(bookIdToSync)) void syncStats(bookIdToSync);
    }
  };

  useEffect(() => {
    if (!book_id || !isActive || !user) {
      if (timerRef.current) clearInterval(timerRef.current);
      return;
    }
    if (!statsRef.current.has(book_id)) statsRef.current.set(book_id, { duration: 0, words: 0 });
    if (!lastSyncTimeRef.current.has(book_id)) lastSyncTimeRef.current.set(book_id, Date.now());
    timerRef.current = setInterval(() => {
      const stats = statsRef.current.get(book_id)!;
      stats.duration += 1;
      stats.words += 2.5;
      if (Date.now() - (lastSyncTimeRef.current.get(book_id) || 0) >= 30000) void syncStats(book_id);
    }, 1000);
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
      void syncStats(book_id);
    };
  }, [book_id, isActive, user]);
};

export function useReadingHeatmapQuery() {
  return useQuery<Record<string, { duration: number; words: number }>>({
    queryKey: ["reader", "heatmap"],
    queryFn: async () => (await readerService.getReadingHeatmap()).data ?? {},
  });
}
export function useReadingGoalQuery() {
  return useQuery<ReadingGoal | undefined>({
    queryKey: ["reader", "goal"],
    queryFn: async () => (await readerService.getReadingGoal()).data,
  });
}
export function useUpsertReadingGoalMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ wordsPerDay, booksPerYear }: { wordsPerDay: number; booksPerYear: number }) => readerService.upsertReadingGoal(wordsPerDay, booksPerYear),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["reader", "goal"] }),
  });
}
