import { useDeleteReviewMutation, useReviewsQuery } from "@/hooks";
import { useReviewAdminStore } from "@/stores";
import {
  AlertCircle,
  Loader2,
  MessageSquareText,
  RefreshCw,
  Star,
  Trash2,
} from "lucide-react";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "react-toastify";
import { useShallow } from "zustand/react/shallow";
import { DiscordMarkdown } from "@/components/common";

export function Reviews() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const {
    reviews,
    setReviews,
    loading,
    setLoading,
    deleting,
    setDeleting,
    reviewToDelete,
    setReviewToDelete,
    page,
    setPage,
    hasMore,
    setHasMore,
    reset,
  } = useReviewAdminStore(
    useShallow((state) => ({
      reviews: state.reviews,
      setReviews: state.setReviews,
      loading: state.loading,
      setLoading: state.setLoading,
      deleting: state.deleting,
      setDeleting: state.setDeleting,
      reviewToDelete: state.reviewToDelete,
      setReviewToDelete: state.setReviewToDelete,
      page: state.page,
      setPage: state.setPage,
      hasMore: state.hasMore,
      setHasMore: state.setHasMore,
      reset: state.reset,
    })),
  );

  const {
    data: pageData,
    isLoading,
    isFetching,
    refetch,
  } = useReviewsQuery(page);
  const deleteReviewMutation = useDeleteReviewMutation();

  useEffect(() => {
    if (pageData) {
      if (page === 0) {
        setReviews(pageData);
      } else {
        setReviews((prev) => {
          const existingKeys = new Set(
            prev.map((r) => `${r.book_id}-${r.user_id}`),
          );
          const uniqueNew = pageData.filter(
            (r) => !existingKeys.has(`${r.book_id}-${r.user_id}`),
          );
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
              (r) =>
                !(
                  r.book_id === reviewToDelete.book_id &&
                  r.user_id === reviewToDelete.user_id
                ),
            ),
          );
          setReviewToDelete(null);
          setDeleting(null);
        },
        onError: (err) => {
          toast.error(err instanceof Error ? err.message : String(err));
          setDeleting(null);
        },
      },
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
      <header className="px-4 py-4 sm:px-6 lg:px-8 lg:py-6 border-b border-base-200 flex items-center justify-between gap-3 sm:gap-4 bg-base-100/50 backdrop-blur-xl sticky top-0 z-10">
        <div className="min-w-0 flex-1">
          <h1 className="text-xl sm:text-2xl font-bold tracking-tight">
            {t("admin.review_moderation", "Review Moderation")}
          </h1>
          <p className="text-xs sm:text-sm text-base-content/60 mt-0.5 sm:mt-1 line-clamp-1 sm:line-clamp-none">
            {t(
              "admin.review_moderation_desc",
              "View and manage all user reviews",
            )}
          </p>
        </div>
        <button
          onClick={async () => {
            setPage(0);
            await queryClient.invalidateQueries({
              queryKey: ["admin", "reviews"],
            });
            await refetch();
            toast.info(t("common.refreshed", "Data refreshed"));
          }}
          className="btn btn-square btn-ghost btn-sm sm:btn-md shrink-0"
          title={t("admin.operations.refresh", "Refresh")}
          disabled={isFetching}
        >
          <RefreshCw
            className={`h-4 w-4 sm:h-5 sm:w-5 ${isFetching ? "animate-spin" : ""}`}
          />
        </button>
      </header>

      {/* Content */}
      <div className="flex-1 overflow-auto p-4 sm:p-6 lg:p-8">
        <div className="max-w-7xl mx-auto w-full space-y-6">
          {loading && reviews.length === 0 ? (
            <div className="flex items-center justify-center py-20 opacity-50">
              <Loader2 className="animate-spin h-8 w-8 text-primary mr-3" />
              <span className="text-lg">{t("admin.loading_reviews")}</span>
            </div>
          ) : reviews.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-base-300 bg-base-100 p-12 sm:p-16 text-center flex flex-col items-center justify-center gap-3 shadow-xs">
              <div className="grid h-14 w-14 place-items-center rounded-2xl bg-primary/10 text-primary mb-1">
                <MessageSquareText className="h-7 w-7" />
              </div>
              <div>
                <h3 className="text-base sm:text-lg font-bold text-base-content">
                  {t("admin.no_reviews", "No user reviews found")}
                </h3>
                <p className="text-xs sm:text-sm text-base-content/60 mt-1 max-w-sm">
                  {t(
                    "admin.no_reviews_hint",
                    "User reviews submitted for library books will appear here.",
                  )}
                </p>
              </div>
            </div>
          ) : (
            <div className="flex flex-col gap-3">
              {reviews.map((review, idx) => (
                <div
                  key={`${review.book_id}-${review.user_id}-${idx}`}
                  className="card bg-base-100 border border-base-200 shadow-sm p-4 sm:p-5 hover:border-base-300 transition-colors"
                >
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-3 mb-2 flex-wrap">
                        <h3 className="font-bold text-base text-base-content truncate">
                          {review.book_title || review.book_id}
                        </h3>
                        {renderStars(review.rating)}
                        <span className="text-xs font-semibold text-base-content/70">
                          {review.rating}/5
                        </span>
                      </div>
                      {review.review ? (
                        <div className="text-sm text-base-content leading-relaxed">
                          <DiscordMarkdown content={review.review} />
                        </div>
                      ) : (
                        <p className="text-sm italic text-base-content/60">
                          {t("admin.rating_only")}
                        </p>
                      )}
                      <div className="flex items-center gap-2.5 mt-3 text-xs text-base-content/70 flex-wrap">
                        <span>
                          by{" "}
                          <strong className="font-semibold text-base-content">
                            {review.user_name || "User"}
                          </strong>
                          {review.user_email && (
                            <span className="text-base-content/60">
                              {" "}
                              ({review.user_email})
                            </span>
                          )}
                        </span>
                        <span className="text-base-content/40">•</span>
                        <span className="text-base-content/80 font-medium">
                          {review.created_at
                            ? new Date(review.created_at).toLocaleDateString()
                            : "—"}
                        </span>
                        <span className="text-base-content/40">•</span>
                        <span className="font-mono text-xs bg-base-200/80 px-1.5 py-0.5 rounded text-base-content/70">
                          Book: {review.book_id}
                        </span>
                      </div>
                    </div>
                    <button
                      onClick={() => setReviewToDelete(review)}
                      disabled={
                        deleting === `${review.book_id}-${review.user_id}`
                      }
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
              {t(
                "admin.review_delete_confirm",
                "Are you sure you want to delete this review? This action cannot be undone.",
              )}
            </p>
            <div className="bg-base-200/50 p-3 rounded-lg text-sm">
              <p className="font-medium">
                {reviewToDelete?.book_title || reviewToDelete?.book_id}
              </p>
              <p className="text-xs opacity-60 mt-1">
                {t(
                  "admin.review_by_rating",
                  "by {{name}} · Rating: {{rating}}/5",
                  {
                    name:
                      reviewToDelete?.user_name ||
                      reviewToDelete?.user_email ||
                      t("common.user", "User"),
                    rating: reviewToDelete?.rating,
                  },
                )}
              </p>
              {reviewToDelete?.review && (
                <p className="text-xs opacity-70 mt-2 italic line-clamp-3">
                  "{reviewToDelete.review}"
                </p>
              )}
            </div>
          </div>
          <div className="modal-action">
            <button
              onClick={() => setReviewToDelete(null)}
              className="btn btn-ghost"
            >
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
