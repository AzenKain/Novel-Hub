import { useDeleteReviewMutation, useReviewsQuery } from "@/hooks";
import { useReviewAdminStore } from "@/stores";
import { AlertCircle, Loader2, MessageSquareText, RefreshCw, Star, Trash2 } from "lucide-react";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";
import { useShallow } from "zustand/react/shallow";

export function Reviews() {
  const { t } = useTranslation();
  const {
    reviews, setReviews,
    loading, setLoading,
    deleting, setDeleting,
    reviewToDelete, setReviewToDelete,
    page, setPage,
    hasMore, setHasMore,
    reset
  } = useReviewAdminStore(useShallow((state) => ({
    reviews: state.reviews, setReviews: state.setReviews,
    loading: state.loading, setLoading: state.setLoading,
    deleting: state.deleting, setDeleting: state.setDeleting,
    reviewToDelete: state.reviewToDelete, setReviewToDelete: state.setReviewToDelete,
    page: state.page, setPage: state.setPage,
    hasMore: state.hasMore, setHasMore: state.setHasMore,
    reset: state.reset
  })));

  const { data: pageData, isLoading, refetch } = useReviewsQuery(page);
  const deleteReviewMutation = useDeleteReviewMutation();

  useEffect(() => {
    if (pageData) {
      if (page === 0) {
        setReviews(pageData);
      } else {
        setReviews((prev) => {
          const existingKeys = new Set(prev.map(r => `${r.book_id}-${r.user_id}`));
          const uniqueNew = pageData.filter(r => !existingKeys.has(`${r.book_id}-${r.user_id}`));
          return [...prev, ...uniqueNew];
        });
      }
      setHasMore(pageData.length === 20);
    }
  }, [pageData, page, setReviews, setHasMore]);

  useEffect(() => {
    setLoading(isLoading);
  }, [isLoading, setLoading]);

  useEffect(() => {
    return () => {
      reset();
    };
  }, [reset]);

  function confirmDelete() {
    if (!reviewToDelete) return;
    const key = `${reviewToDelete.book_id}-${reviewToDelete.user_id}`;
    setDeleting(key);
    deleteReviewMutation.mutate(
      { book_id: reviewToDelete.book_id, user_id: reviewToDelete.user_id },
      {
        onSuccess: () => {
          toast.success(t("admin.review_deleted", "Review deleted"));
          setReviews((prev) =>
            prev.filter(
              (r) => !(r.book_id === reviewToDelete.book_id && r.user_id === reviewToDelete.user_id)
            )
          );
          setReviewToDelete(null);
          setDeleting(null);
        },
        onError: (err) => {
          toast.error(err instanceof Error ? err.message : String(err));
          setDeleting(null);
        },
      }
    );
  }

  function renderStars(rating: number) {
    return (
      <span className="flex items-center gap-0.5">
        {Array.from({ length: 5 }, (_, i) => (
          <Star
            key={i}
            className={`h-3 w-3 ${i < rating ? "fill-warning text-warning" : "text-base-content/20"}`}
          />
        ))}
      </span>
    );
  }

  return (
    <div className="flex flex-col h-full bg-base-100">
      {/* Header */}
      <header className="px-4 py-5 sm:px-6 lg:px-8 lg:py-6 border-b border-base-200 flex items-center justify-between bg-base-100/50 backdrop-blur-xl sticky top-0 z-10">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("admin.review_moderation", "Review Moderation")}</h1>
          <p className="text-sm text-base-content/60 mt-1">{t("admin.review_moderation_desc", "View and manage all user reviews")}</p>
        </div>
        <button
          onClick={() => {
            setPage(0);
            void refetch();
          }}
          className="btn btn-square btn-ghost btn-sm sm:btn-md"
          title={t("admin.operations.refresh", "Refresh")}
        >
          <RefreshCw className={`h-5 w-5 ${loading ? "animate-spin" : ""}`} />
        </button>
      </header>

      {/* Content */}
      <div className="flex-1 overflow-auto p-4 sm:p-6 lg:p-8">
        <div className="max-w-4xl mx-auto">
          {loading && reviews.length === 0 ? (
            <div className="flex items-center justify-center py-20 opacity-50">
              <Loader2 className="animate-spin h-8 w-8 text-primary mr-3" />
              <span className="text-lg">Loading reviews...</span>
            </div>
          ) : reviews.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-20 opacity-50 border-2 border-dashed border-base-300 rounded-2xl bg-base-200/50">
              <MessageSquareText className="h-12 w-12 mb-4" />
              <p className="text-lg font-medium">No reviews found</p>
              <p className="text-sm mt-1">User reviews will appear here as they are submitted.</p>
            </div>
          ) : (
            <div className="flex flex-col gap-3">
              {reviews.map((review, idx) => (
                <div
                  key={`${review.book_id}-${review.user_id}-${idx}`}
                  className="card bg-base-100 border border-base-200 shadow-sm p-4 sm:p-5"
                >
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-3 mb-2 flex-wrap">
                        <h3 className="font-bold text-sm truncate">{review.book_title || review.book_id}</h3>
                        {renderStars(review.rating)}
                        <span className="text-xs text-base-content/40">{review.rating}/5</span>
                      </div>
                      {review.review ? (
                        <p className="text-sm text-base-content/80 whitespace-pre-wrap break-words">
                          {review.review}
                        </p>
                      ) : (
                        <p className="text-sm italic text-base-content/40">Rating only</p>
                      )}
                      <div className="flex items-center gap-3 mt-3 text-xs text-base-content/50 flex-wrap">
                        <span>
                          by <strong className="text-base-content/70">{review.user_name || "User"}</strong>
                          {review.user_email && (
                            <span className="text-base-content/40"> ({review.user_email})</span>
                          )}
                        </span>
                        <span className="text-base-content/30">·</span>
                        <span>
                          {review.created_at
                            ? new Date(review.created_at).toLocaleDateString()
                            : "—"}
                        </span>
                        <span className="text-base-content/30">·</span>
                        <span className="font-mono text-[10px] opacity-40">
                          Book: {review.book_id}
                        </span>
                      </div>
                    </div>
                    <button
                      onClick={() => setReviewToDelete(review)}
                      disabled={deleting === `${review.book_id}-${review.user_id}`}
                      className="btn btn-ghost btn-sm text-error hover:bg-error/10 shrink-0"
                      title={t("admin.review_delete", "Delete review")}
                    >
                      {deleting === `${review.book_id}-${review.user_id}` ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <Trash2 className="h-4 w-4" />
                      )}
                    </button>
                  </div>
                </div>
              ))}

              {hasMore && (
                <div className="flex justify-center pt-4">
                  <button
                    onClick={() => setPage(page + 1)}
                    disabled={loading}
                    className="btn btn-ghost btn-wide"
                  >
                    {loading ? (
                      <Loader2 className="h-5 w-5 animate-spin" />
                    ) : (
                      t("common.load_more", "Load More")
                    )}
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Delete Confirmation Modal */}
      <dialog className={`modal ${reviewToDelete ? "modal-open" : ""}`}>
        <div className="modal-box">
          <h3 className="font-bold text-lg text-error flex items-center gap-2">
            <AlertCircle className="w-6 h-6" />
            {t("admin.review_delete_title", "Delete Review")}
          </h3>
          <div className="py-4">
            <p className="text-sm opacity-80 mb-3">
              {t("admin.review_delete_confirm", "Are you sure you want to delete this review? This action cannot be undone.")}
            </p>
            <div className="bg-base-200/50 p-3 rounded-lg text-sm">
              <p className="font-medium">{reviewToDelete?.book_title || reviewToDelete?.book_id}</p>
              <p className="text-xs opacity-60 mt-1">
                {t("admin.review_by_rating", "by {{name}} · Rating: {{rating}}/5", {
                  name: reviewToDelete?.user_name || reviewToDelete?.user_email || t("common.user", "User"),
                  rating: reviewToDelete?.rating,
                })}
              </p>
              {reviewToDelete?.review && (
                <p className="text-xs opacity-70 mt-2 italic line-clamp-3">"{reviewToDelete.review}"</p>
              )}
            </div>
          </div>
          <div className="modal-action">
            <button onClick={() => setReviewToDelete(null)} className="btn btn-ghost">
              {t("common.cancel", "Cancel")}
            </button>
            <button
              onClick={confirmDelete}
              className="btn btn-error"
              disabled={deleting !== null}
            >
              {deleting !== null ? (
                <span className="loading loading-spinner loading-xs"></span>
              ) : (
                t("admin.review_delete_title", "Delete Review")
              )}
            </button>
          </div>
        </div>
        <form method="dialog" className="modal-backdrop">
          <button onClick={() => setReviewToDelete(null)}>close</button>
        </form>
      </dialog>
    </div>
  );
}
