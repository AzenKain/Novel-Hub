import React, { useState } from "react";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Star, MessageSquare, Trash2, Send, Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ConfirmModal } from "@/components/common";
import { adminService, featureService } from "@/services";
import { useAuthStore } from "@/stores";
import type { BookReview } from "@/types";
import { toast } from "react-toastify";
import { hasPermission } from "@/utils/permission";
import { useShallow } from "zustand/react/shallow";

interface ReviewSectionProps {
  book_id: string;
  userReview?: BookReview;
}

export const ReviewSection: React.FC<ReviewSectionProps> = ({ book_id, userReview }) => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { user } = useAuthStore(useShallow((state) => ({ user: state.user })));
  
  const [rating, setRating] = useState(userReview?.rating || 0);
  const [hoverRating, setHoverRating] = useState(0);
  const [reviewText, setReviewText] = useState(userReview?.review || "");
  const [isDeleteReviewOpen, setIsDeleteReviewOpen] = useState(false);
  const [deleteReviewUserId, setDeleteReviewUserId] = useState<string | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);

  const handleConfirmDeleteReview = async () => {
    if (!deleteReviewUserId) return;
    setDeleteLoading(true);
    try {
      await adminService.deleteReview(book_id, deleteReviewUserId);
      toast.success(t("review.deleted", "Review deleted successfully"));
      queryClient.invalidateQueries({ queryKey: ["bookReviews", book_id] });
      queryClient.invalidateQueries({ queryKey: ["bookUserState", book_id] });
      setIsDeleteReviewOpen(false);
    } catch (error: any) {
      toast.error(error.message || t("error.unknown", "Failed to delete review"));
    } finally {
      setDeleteLoading(false);
      setDeleteReviewUserId(null);
    }
  };

  const {
    data: reviewsDataRaw,
    isLoading,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: ["bookReviews", book_id],
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) => {
      const res = await featureService.listBookReviews(book_id, pageParam, 10);
      if (!res.status) throw new Error(res.message || "Failed to fetch reviews");
      return res;
    },
    getNextPageParam: (lastPage) => lastPage.next_cursor || undefined,
  });

  const reviewsData = React.useMemo(() => reviewsDataRaw?.pages.flatMap(p => p.data) || [], [reviewsDataRaw]);

  const upsertMutation = useMutation({
    mutationFn: async () => {
      const res = await featureService.upsertBookReview(book_id, rating, reviewText);
      if (!res.status) throw new Error(res.message || "Failed to submit review");
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["bookReviews", book_id] });
      queryClient.invalidateQueries({ queryKey: ["bookUserState", book_id] });
      queryClient.invalidateQueries({ queryKey: ["metadata"] });
      queryClient.invalidateQueries({ queryKey: ["books"] });
      queryClient.invalidateQueries({ queryKey: ["book", book_id] });
      toast.success(t("review.submitted", "Review submitted successfully"));
    },
    onError: (err: any) => {
      toast.error(err.message || t("error.unknown", "Failed to submit review"));
    }
  });

  const deleteMutation = useMutation({
    mutationFn: async () => {
      const res = await featureService.deleteBookReview(book_id);
      if (!res.status) throw new Error(res.message || "Failed to delete review");
      return res.data;
    },
    onSuccess: () => {
      setRating(0);
      setReviewText("");
      queryClient.invalidateQueries({ queryKey: ["bookReviews", book_id] });
      queryClient.invalidateQueries({ queryKey: ["bookUserState", book_id] });
      queryClient.invalidateQueries({ queryKey: ["metadata"] });
      queryClient.invalidateQueries({ queryKey: ["books"] });
      queryClient.invalidateQueries({ queryKey: ["book", book_id] });
      toast.success(t("review.deleted", "Review deleted successfully"));
    },
    onError: (err: any) => {
      toast.error(err.message || t("error.unknown", "Failed to delete review"));
    }
  });

  const handleSubmit = (e: React.SyntheticEvent) => {
    e.preventDefault();
    if (rating === 0) {
      toast.warning(t("review.select_rating", "Please select a star rating"));
      return;
    }
    upsertMutation.mutate();
  };

  const reviews = reviewsData;

  return (
    <div>
      <h3 className="text-2xl font-bold mb-2 flex items-center gap-2">
        <MessageSquare className="w-6 h-6 text-primary" />
        {t("review.title", "Reviews & Ratings")}
      </h3>

      {/* Write Review Section */}
      {user ? (
        <div className="mb-8 mt-2">
          <h4 className="text-sm font-bold mb-3 opacity-80 uppercase tracking-wide">
            {userReview ? t("review.edit_yours", "Edit your review") : t("review.write_yours", "Write a review")}
          </h4>
          <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <div className="flex items-center">
              {[1, 2, 3, 4, 5].map((star) => (
                <button
                  type="button"
                  key={star}
                  onClick={() => setRating(star)}
                  onMouseEnter={() => setHoverRating(star)}
                  onMouseLeave={() => setHoverRating(0)}
                  className="p-1 focus:outline-none transition-transform hover:scale-110 -ml-1 first:ml-0"
                >
                  <Star
                    className={`w-6 h-6 ${(hoverRating || rating) >= star ? "fill-warning text-warning" : "text-base-content/20"}`}
                  />
                </button>
              ))}
            </div>
            
            <div className="relative">
              <textarea
                className="textarea textarea-bordered bg-base-100 w-full resize-none h-32 text-base focus:outline-none focus:ring-1 focus:ring-primary/30 transition-colors pb-12"
                placeholder={t("review.placeholder", "What did you think about this book?")}
                value={reviewText}
                onChange={(e) => setReviewText(e.target.value)}
                disabled={upsertMutation.isPending}
              />
              <div className="absolute bottom-2 right-2 flex justify-end gap-1">
                {userReview && (
                  <button 
                    type="button" 
                    className="btn btn-sm btn-ghost hover:bg-error/10 text-error/70 hover:text-error"
                    onClick={() => deleteMutation.mutate()}
                    disabled={deleteMutation.isPending || upsertMutation.isPending}
                    title={t("common.delete", "Delete")}
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                )}
                <button 
                  type="submit" 
                  className="btn btn-sm btn-ghost text-base-content/70 hover:text-primary hover:bg-primary/10"
                  disabled={upsertMutation.isPending || deleteMutation.isPending || !reviewText.trim()}
                >
                  {upsertMutation.isPending ? <Loader2 className="w-4 h-4 animate-spin mr-1" /> : <Send className="w-4 h-4 mr-1" />}
                  {t("common.submit", "Submit")}
                </button>
              </div>
            </div>
          </form>
        </div>
      ) : (
        <div className="mb-8 py-6 text-center text-base-content/60 border border-dashed border-base-300 rounded-xl">
          {t("review.login_required", "Please log in to write a review.")}
        </div>
      )}

      {/* Review List */}
      <div>
        {isLoading ? (
          <div className="flex justify-center py-8">
            <Loader2 className="w-8 h-8 animate-spin text-primary" />
          </div>
        ) : reviews.length > 0 ? (
          <>
            <div className="flex flex-col gap-4">
              {reviews.map((rv, idx) => (
                <div key={idx} className="bg-base-100 p-5 rounded-2xl shadow-sm border border-base-200">
                  <div className="flex justify-between items-start mb-3">
                    <div className="flex items-center gap-2">
                      <div className="grid h-8 w-8 place-items-center rounded-full bg-primary/10 text-primary font-bold text-xs ring-2 ring-primary/20 shrink-0">
                        {((rv as any).username || (rv as any).user_name || "U").charAt(0).toUpperCase()}
                      </div>
                      <div>
                        <div className="font-bold text-sm">{(rv as any).username || t("common.user", "User")}</div>
                        <div className="text-xs text-base-content/50">
                          {rv?.updated_at ? new Date(rv.updated_at).toLocaleDateString() : ""}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-3">
                      <div className="flex items-center gap-1">
                        <Star className="w-4 h-4 fill-warning text-warning" />
                        <span className="font-bold text-sm">{rv?.rating}</span>
                      </div>
                      {hasPermission(user, "book.review.delete") && (
                        <button
                          className="btn btn-ghost btn-xs text-error"
                          onClick={() => {
                            if (rv?.user_id) {
                              setDeleteReviewUserId(rv.user_id);
                              setIsDeleteReviewOpen(true);
                            }
                          }}
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      )}
                    </div>
                  </div>
                  {rv?.review && (
                    <p className="text-base-content/80 text-sm whitespace-pre-wrap">{rv.review}</p>
                  )}
                </div>
              ))}
            </div>
            
            {hasNextPage && (
              <div className="flex justify-center mt-6">
                <button
                  className="btn btn-outline btn-sm"
                  onClick={() => fetchNextPage()}
                  disabled={isFetchingNextPage}
                >
                  {isFetchingNextPage ? (
                    <Loader2 className="w-4 h-4 animate-spin mr-1" />
                  ) : null}
                  {t("common.load_more", "Load more")}
                </button>
              </div>
            )}
          </>
        ) : (
          <div className="text-center py-12 text-base-content/50 bg-base-100 rounded-2xl border border-dashed border-base-300 shadow-2xs">
            <MessageSquare className="w-8 h-8 mx-auto mb-2 opacity-30" />
            <p>{t("review.no_reviews", "No reviews yet. Be the first to review!")}</p>
          </div>
        )}
      </div>

      <ConfirmModal
        open={isDeleteReviewOpen}
        title={t("review.confirm_delete_title", "Delete Review")}
        message={t("review.confirm_delete", "Are you sure you want to delete this review?")}
        onClose={() => {
          setIsDeleteReviewOpen(false);
          setDeleteReviewUserId(null);
        }}
        onConfirm={handleConfirmDeleteReview}
        variant="danger"
        loading={deleteLoading}
      />
    </div>
  );
};
