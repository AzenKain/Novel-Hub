import { useState, useEffect } from 'react';
import { getHighlights, createHighlight, deleteHighlight, Highlight } from '../api/highlights';
import { useAuthStore } from '@/stores';

export const useHighlights = (bookId: string, chapterId: string | undefined) => {
  const [highlights, setHighlights] = useState<Highlight[]>([]);
  const [loading, setLoading] = useState(false);

  const { user } = useAuthStore();

  useEffect(() => {
    if (!chapterId || !user) {
      setHighlights([]);
      return;
    }
    const fetchHighlights = async () => {
      try {
        setLoading(true);
        const data = await getHighlights(chapterId);
        setHighlights(Array.isArray(data) ? data : []);
      } catch (err) {
        console.error("Failed to fetch highlights", err);
      } finally {
        setLoading(false);
      }
    };
    fetchHighlights();
  }, [chapterId]);

  const addHighlight = async (textContent: string, startIndex: number, endIndex: number, color: string = 'yellow') => {
    if (!chapterId || !bookId) return null;
    try {
      const newHighlight = await createHighlight(bookId, chapterId, textContent, startIndex, endIndex, color);
      setHighlights(prev => [...prev, newHighlight]);
      return newHighlight;
    } catch (err) {
      console.error("Failed to create highlight", err);
      return null;
    }
  };

  const removeHighlight = async (id: string) => {
    try {
      await deleteHighlight(id);
      setHighlights(prev => prev.filter(h => h.id !== id));
    } catch (err) {
      console.error("Failed to delete highlight", err);
    }
  };

  return { highlights, addHighlight, removeHighlight, loading };
};
