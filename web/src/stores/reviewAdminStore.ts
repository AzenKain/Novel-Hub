import { adminService } from "@/services";
import type { AdminReview } from "@/types";
import { create } from "zustand";

interface ReviewAdminState {
  reviews: AdminReview[];
  loading: boolean;
  deleting: string | null;
  reviewToDelete: AdminReview | null;
  page: number;
  hasMore: boolean;

  setReviews: (reviews: AdminReview[] | ((prev: AdminReview[]) => AdminReview[])) => void;
  setLoading: (loading: boolean) => void;
  setDeleting: (deleting: string | null) => void;
  setReviewToDelete: (review: AdminReview | null) => void;
  setPage: (page: number) => void;
  setHasMore: (hasMore: boolean) => void;

  loadData: (nextPage?: number) => Promise<void>;
  reset: () => void;
}

const initialState = {
  reviews: [],
  loading: true,
  deleting: null,
  reviewToDelete: null,
  page: 0,
  hasMore: true,
};

export const useReviewAdminStore = create<ReviewAdminState>((set, get) => ({
  ...initialState,

  setReviews: (reviews) => set((state) => ({ reviews: typeof reviews === "function" ? reviews(state.reviews) : reviews })),
  setLoading: (loading) => set({ loading }),
  setDeleting: (deleting) => set({ deleting }),
  setReviewToDelete: (reviewToDelete) => set({ reviewToDelete }),
  setPage: (page) => set({ page }),
  setHasMore: (hasMore) => set({ hasMore }),

  loadData: async (nextPage) => {
    const p = nextPage ?? get().page;
    set({ loading: true });
    try {
      const res = await adminService.listAllReviews(20, p * 20);
      const data = res.data || [];
      if (p === 0) {
        set({ reviews: data });
      } else {
        set((state) => ({ reviews: [...state.reviews, ...data] }));
      }
      set({ hasMore: data.length === 20 });
    } catch (err) {
      console.error(err);
    } finally {
      set({ loading: false });
    }
  },

  reset: () => set(initialState),
}));
