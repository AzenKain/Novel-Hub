import { useReviewsQuery } from "@/hooks";
import { adminService } from "@/services";
import { useQueryClient } from "@tanstack/react-query";
import { AlertCircle, Loader2, MessageSquareText, RefreshCw, Star, Trash2 } from "lucide-react";
import { useEffect } from "react";
import { toast } from "react-toastify";
import { useShallow } from "zustand/react/shallow";
import { useReviewAdminStore } from "@/stores";

export function Reviews() {
  const queryClient = useQueryClient();
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

  useEffect(() => {
    if (pageData) {
      if (page === 0) {
        setReviews(pageData);
      } else {
        setReviews((prev) => {
          const existingKeys = new Set(prev.map(r => `${r.bookId}-${r.userId}`));
          const uniqueNew = pageData.filter(r => !existingKeys.has(`${r.bookId}-${r.userId}`));
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

  async function confirmDelete() {
    if (!reviewToDelete) return;
    const key = `${reviewToDelete.bookId}-${reviewToDelete.userId}`;
    setDeleting(key);
    try {
      await adminService.deleteReview(reviewToDelete.bookId, reviewToDelete.userId);
      toast.success("Review deleted");
      setReviews((prev) =>
        prev.filter(
          (r) => !(r.bookId === reviewToDelete.bookId && r.userId === reviewToDelete.userId)
        )
      );
      setReviewToDelete(null);
      await queryClient.invalidateQueries({ queryKey: ["admin", "reviews"] });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setDeleting(null);
    }
  }

  function loadMore() {
    setPage(page + 1);
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
          <h1 className="text-2xl font-bold tracking-tight">Review Moderation</h1>
          <p className="text-sm text-base-content/60 mt-1">View and manage all user reviews</p>
        </div>
        <button
          onClick={() => {
            setPage(0);
            void refetch();
          }}
          className="btn btn-square btn-ghost btn-sm sm:btn-md"
          title="Refresh"
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
                  key={`${review.bookId}-${review.userId}-${idx}`}
                  className="card bg-base-100 border border-base-200 shadow-sm p-4 sm:p-5"
                >
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-3 mb-2 flex-wrap">
                        <h3 className="font-bold text-sm truncate">{review.bookTitle || review.bookId}</h3>
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
                          by <strong className="text-base-content/70">{review.userName || "User"}</strong>
                          {review.userEmail && (
                            <span className="text-base-content/40"> ({review.userEmail})</span>
                          )}
                        </span>
                        <span className="text-base-content/30">·</span>
                        <span>
                          {review.createdAt
                            ? new Date(review.createdAt).toLocaleDateString()
                            : "—"}
                        </span>
                        <span className="text-base-content/30">·</span>
                        <span className="font-mono text-[10px] opacity-40">
                          Book: {review.bookId}
                        </span>
                      </div>
                    </div>
                    <button
                      onClick={() => setReviewToDelete(review)}
                      disabled={deleting === `${review.bookId}-${review.userId}`}
                      className="btn btn-ghost btn-sm text-error hover:bg-error/10 shrink-0"
                      title="Delete review"
                    >
                      {deleting === `${review.bookId}-${review.userId}` ? (
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
                    onClick={loadMore}
                    disabled={loading}
                    className="btn btn-ghost btn-wide"
                  >
                    {loading ? (
                      <Loader2 className="h-5 w-5 animate-spin" />
                    ) : (
                      "Load More"
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
            Delete Review
          </h3>
          <div className="py-4">
            <p className="text-sm opacity-80 mb-3">
              Are you sure you want to delete this review? This action cannot be undone.
            </p>
            <div className="bg-base-200/50 p-3 rounded-lg text-sm">
              <p className="font-medium">{reviewToDelete?.bookTitle || reviewToDelete?.bookId}</p>
              <p className="text-xs opacity-60 mt-1">
                by {reviewToDelete?.userName || reviewToDelete?.userEmail || "User"} · Rating: {reviewToDelete?.rating}/5
              </p>
              {reviewToDelete?.review && (
                <p className="text-xs opacity-70 mt-2 italic line-clamp-3">"{reviewToDelete.review}"</p>
              )}
            </div>
          </div>
          <div className="modal-action">
            <button onClick={() => setReviewToDelete(null)} className="btn btn-ghost">
              Cancel
            </button>
            <button
              onClick={() => void confirmDelete()}
              className="btn btn-error"
              disabled={deleting !== null}
            >
              {deleting !== null ? (
                <span className="loading loading-spinner loading-xs"></span>
              ) : (
                "Delete Review"
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
