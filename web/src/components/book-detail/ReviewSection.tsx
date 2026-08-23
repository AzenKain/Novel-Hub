import React, { useState, useRef } from "react";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Star,
  MessageSquare,
  Trash2,
  Send,
  Loader2,
  Bold,
  Italic,
  Underline as UnderlineIcon,
  Strikethrough as StrikeIcon,
  Code,
  Quote,
  Eye,
  EyeOff,
  Sparkles,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { ConfirmModal, DiscordMarkdown } from "@/components/common";
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
  const [isPreview, setIsPreview] = useState(false);
  const [isDeleteReviewOpen, setIsDeleteReviewOpen] = useState(false);
  const [deleteReviewUserId, setDeleteReviewUserId] = useState<string | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);

  const applyFormat = (prefix: string, suffix = prefix, placeholder = t("review.fmt_content")) => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const currentVal = textarea.value;

    let selectedText = currentVal.substring(start, end);
    if (!selectedText) {
      selectedText = placeholder;
    }

    const before = currentVal.substring(0, start);
    const after = currentVal.substring(end);
    const newVal = `${before}${prefix}${selectedText}${suffix}${after}`;

    setReviewText(newVal);

    setTimeout(() => {
      textarea.focus();
      const newCursorStart = start + prefix.length;
      const newCursorEnd = newCursorStart + selectedText.length;
      textarea.setSelectionRange(newCursorStart, newCursorEnd);
    }, 10);
  };

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
            
            {/* Markdown Toolbar & Preview Toggle */}
            <div className="rounded-2xl border border-base-300 bg-base-100 overflow-hidden shadow-xs focus-within:border-primary/40 focus-within:ring-2 focus-within:ring-primary/10 transition-all">
              {/* Toolbar */}
              <div className="flex flex-wrap items-center justify-between gap-1 border-b border-base-200 bg-base-200/50 px-3 py-1.5 text-xs">
                <div className="flex items-center gap-0.5 flex-wrap">
                  <button
                    type="button"
                    onClick={() => applyFormat("**", "**", t("review.fmt_bold"))}
                    className="btn btn-ghost btn-xs h-7 min-h-0 px-2 font-bold hover:bg-base-300 rounded"
                    title={t("review.fmt_bold_title")}
                  >
                    <Bold className="w-3.5 h-3.5" />
                  </button>
                  <button
                    type="button"
                    onClick={() => applyFormat("*", "*", t("review.fmt_italic"))}
                    className="btn btn-ghost btn-xs h-7 min-h-0 px-2 italic hover:bg-base-300 rounded"
                    title={t("review.fmt_italic_title")}
                  >
                    <Italic className="w-3.5 h-3.5" />
                  </button>
                  <button
                    type="button"
                    onClick={() => applyFormat("__", "__", t("review.fmt_underline"))}
                    className="btn btn-ghost btn-xs h-7 min-h-0 px-2 underline hover:bg-base-300 rounded"
                    title={t("review.fmt_underline_title")}
                  >
                    <UnderlineIcon className="w-3.5 h-3.5" />
                  </button>
                  <button
                    type="button"
                    onClick={() => applyFormat("~~", "~~", t("review.fmt_strikethrough"))}
                    className="btn btn-ghost btn-xs h-7 min-h-0 px-2 line-through hover:bg-base-300 rounded"
                    title={t("review.fmt_strikethrough_title")}
                  >
                    <StrikeIcon className="w-3.5 h-3.5" />
                  </button>
                  <button
                    type="button"
                    onClick={() => applyFormat("`", "`", "code")}
                    className="btn btn-ghost btn-xs h-7 min-h-0 px-2 font-mono hover:bg-base-300 rounded"
                    title={t("review.fmt_code_title")}
                  >
                    <Code className="w-3.5 h-3.5" />
                  </button>
                  <button
                    type="button"
                    onClick={() => applyFormat("> ", "", t("review.fmt_quote"))}
                    className="btn btn-ghost btn-xs h-7 min-h-0 px-2 hover:bg-base-300 rounded"
                    title={t("review.fmt_quote_title")}
                  >
                    <Quote className="w-3.5 h-3.5" />
                  </button>
                  <div className="h-4 w-px bg-base-300 mx-1" />
                  <button
                    type="button"
                    onClick={() => applyFormat("||", "||", t("review.fmt_spoiler"))}
                    className="btn btn-xs bg-neutral-800 text-white hover:bg-neutral-700 h-7 min-h-0 px-2 gap-1 rounded font-semibold text-[11px] shadow-xs"
                    title={t("review.fmt_spoiler_title")}
                  >
                    <EyeOff className="w-3 h-3 text-warning" />
                    <span>Spoiler</span>
                  </button>
                </div>

                <div className="flex items-center gap-1">
                  <button
                    type="button"
                    onClick={() => setIsPreview(false)}
                    className={`btn btn-xs h-6 min-h-0 px-2 rounded text-[11px] font-semibold transition-all ${
                      !isPreview
                        ? "btn-primary"
                        : "btn-ghost opacity-60 hover:opacity-100"
                    }`}
                  >
                    {t("common.write", "Write")}
                  </button>
                  <button
                    type="button"
                    onClick={() => setIsPreview(true)}
                    className={`btn btn-xs h-6 min-h-0 px-2 rounded text-[11px] font-semibold gap-1 transition-all ${
                      isPreview
                        ? "btn-primary"
                        : "btn-ghost opacity-60 hover:opacity-100"
                    }`}
                  >
                    <Sparkles className="w-3 h-3" />
                    {t("common.preview", "Image preview")}
                  </button>
                </div>
              </div>

              {/* Editor / Live Preview */}
              {isPreview ? (
                <div className="p-4 min-h-32 bg-base-100">
                  {reviewText.trim() ? (
                    <div className="space-y-1">
                      <div className="text-[10px] font-bold uppercase tracking-wider text-base-content/40 mb-2">
                        {t("review.preview_hint", "Preview format:")}
                      </div>
                      <DiscordMarkdown content={reviewText} className="text-sm text-base-content" />
                    </div>
                  ) : (
                    <div className="py-8 text-center text-sm text-base-content/40 italic">
                      {t("review.empty_preview", "No content to preview yet")}
                    </div>
                  )}
                </div>
              ) : (
                <textarea
                  ref={textareaRef}
                  className="textarea bg-transparent border-0 w-full resize-none h-32 text-sm focus:outline-hidden p-3 font-normal"
                  placeholder={t("review.placeholder", "What did you think about this book?")}
                  value={reviewText}
                  onChange={(e) => setReviewText(e.target.value)}
                  disabled={upsertMutation.isPending}
                />
              )}

              {/* Bottom bar */}
              <div className="flex items-center justify-between border-t border-base-200/60 bg-base-200/20 px-3 py-2">
                <span className="text-[11px] text-base-content/50">
                  💡 {t("review.spoiler_tip", "Use ||content|| to hide spoilers")}
                </span>
                <div className="flex items-center gap-1.5">
                  {userReview && (
                    <button 
                      type="button" 
                      className="btn btn-xs btn-ghost hover:bg-error/10 text-error/70 hover:text-error"
                      onClick={() => deleteMutation.mutate()}
                      disabled={deleteMutation.isPending || upsertMutation.isPending}
                      title={t("common.delete", "Delete")}
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                      <span>{t("common.delete", "Delete")}</span>
                    </button>
                  )}
                  <button 
                    type="submit" 
                    className="btn btn-xs btn-primary gap-1 px-3 shadow-xs"
                    disabled={upsertMutation.isPending || deleteMutation.isPending || !reviewText.trim()}
                  >
                    {upsertMutation.isPending ? <Loader2 className="w-3 h-3 animate-spin" /> : <Send className="w-3 h-3" />}
                    <span>{t("common.submit", "Submit")}</span>
                  </button>
                </div>
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
                <div key={idx} className="bg-base-100 p-5 rounded-2xl shadow-sm border border-base-200 hover:border-base-300 transition-colors">
                  <div className="flex justify-between items-start mb-3">
                    <div className="flex items-center gap-2.5">
                      <div className="grid h-9 w-9 place-items-center rounded-full bg-primary/10 text-primary font-bold text-xs ring-2 ring-primary/20 shrink-0">
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
                      <div className="flex items-center gap-1 bg-base-200/80 px-2 py-0.5 rounded-lg border border-base-300/50">
                        <Star className="w-3.5 h-3.5 fill-warning text-warning" />
                        <span className="font-bold text-xs">{rv?.rating}</span>
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
                    <div className="text-base-content/90 text-sm pl-0.5">
                      <DiscordMarkdown content={rv.review} />
                    </div>
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
