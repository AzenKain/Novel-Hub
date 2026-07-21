import { useEffect, useRef } from 'react';
import { api } from "@/config/api";
import { useAuthStore } from "@/stores";
import { useShallow } from "zustand/react/shallow";

export const useReadingStats = (bookId: string | undefined, isActive: boolean) => {
  const { user } = useAuthStore(useShallow((state) => ({ user: state.user })));
  const durationRef = useRef(0);
  const wordsRef = useRef(0);
  const lastSyncTimeRef = useRef(Date.now());
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Approximate words based on scrolling or reading progress if possible, 
  // but for now we'll increment words by a fixed rate (e.g. 200 wpm) when active, 
  // or track exact words via intersection observer in the future.
  // For simplicity, we assume 150 words per minute of active reading (2.5 words/sec)

  useEffect(() => {
    if (!bookId || !isActive || !user) {
      if (timerRef.current) clearInterval(timerRef.current);
      return;
    }

    timerRef.current = setInterval(() => {
      durationRef.current += 1;
      wordsRef.current += 2.5; // Average 150wpm

      // Sync every 30 seconds
      if (Date.now() - lastSyncTimeRef.current >= 30000) {
        syncStats(bookId);
      }
    }, 1000);

    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
      // Try to sync on unmount
      if (durationRef.current > 0) {
        syncStats(bookId);
      }
    };
  }, [bookId, isActive]);

  const syncStats = async (bookIdToSync: string) => {
    const dur = durationRef.current;
    const wrds = Math.floor(wordsRef.current);
    if (dur === 0) return;

    // Reset lastSyncTimeRef immediately to prevent repeated 1-second spam on error
    lastSyncTimeRef.current = Date.now();

    try {
      await api.post('/reader/stats/session', {
        bookId: bookIdToSync,
        duration: dur,
        words: wrds,
      });
      // Reset duration & words after successful sync
      durationRef.current = 0;
      wordsRef.current = 0;
    } catch (err) {
      console.error("Failed to sync reading stats", err);
    }
  };
};

export const getReadingHeatmap = async () => {
  const { data } = await api.get('/reader/stats/heatmap');
  return data.data;
};
