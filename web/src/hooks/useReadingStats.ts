import { useEffect, useRef } from 'react';
import { readerService } from "@/services";
import { useAuthStore } from "@/stores";
import { useShallow } from "zustand/react/shallow";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export const useReadingStats = (book_id: string | undefined, isActive: boolean) => {
  const { user } = useAuthStore(useShallow((state) => ({ user: state.user })));
  const durationRef = useRef(0);
  const wordsRef = useRef(0);
  const lastSyncTimeRef = useRef(Date.now());
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const syncInFlightRef = useRef(false);

  useEffect(() => {
    if (!book_id || !isActive || !user) {
      if (timerRef.current) clearInterval(timerRef.current);
      return;
    }

    timerRef.current = setInterval(() => {
      durationRef.current += 1;
      wordsRef.current += 2.5; 
      if (Date.now() - lastSyncTimeRef.current >= 30000) {
        syncStats(book_id);
      }
    }, 1000);

    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
      if (durationRef.current > 0) {
        syncStats(book_id);
      }
    };
  }, [book_id, isActive]);

  const syncStats = async (bookIdToSync: string) => {
    const snapshotDuration = durationRef.current;
    const snapshotWords = Math.floor(wordsRef.current);
    if (snapshotDuration < 1 || syncInFlightRef.current) return;
    syncInFlightRef.current = true;
    lastSyncTimeRef.current = Date.now();

    try {
      await readerService.syncReadingSession(bookIdToSync, snapshotDuration, snapshotWords);
      durationRef.current = Math.max(0, durationRef.current - snapshotDuration);
      wordsRef.current = Math.max(0, wordsRef.current - snapshotWords);
    } catch (err) {
      console.error("Failed to sync reading stats", err);
    } finally {
      syncInFlightRef.current = false;
    }
  };
};

export function useReadingHeatmapQuery() {
  return useQuery({
    queryKey: ["reader", "heatmap"],
    queryFn: () => readerService.getReadingHeatmap(),
  });
}

export function useReadingGoalQuery() {
  return useQuery({
    queryKey: ["reader", "goal"],
    queryFn: () => readerService.getReadingGoal(),
  });
}

export function useUpsertReadingGoalMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ wordsPerDay, booksPerYear }: { wordsPerDay: number; booksPerYear: number }) =>
      readerService.upsertReadingGoal(wordsPerDay, booksPerYear),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["reader", "goal"] });
    },
  });
}
