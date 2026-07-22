import React, { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import {
  BookOpen,
  Calendar,
  Download,
  Info,
  Loader2,
  Tag,
  User,
  ArrowLeft,
  Share2,
  Bookmark,
  BookmarkPlus,
  BookmarkMinus,
  Star,
  FolderPlus,
  FolderCheck,
  Check,
  Eye,
  FileText,
  Globe,
  Building
} from "lucide-react";
import { useTranslation } from "react-i18next";
import DOMPurify from "dompurify";
import { bookService } from "@/services";
import { parseMetadata, toStringList } from "@/lib/bookDetail";
import { InfoLine, ShareDialog, ReviewSection } from "@/components/book-detail";
import { usePublicSettings } from "@/hooks/useSettings";
import { hasPermission } from "@/utils/permission";
import { toast } from "react-toastify";
import { useLibraryStore, useAuthStore } from "@/stores";
import { useShallow } from "zustand/react/shallow";
import {
  useBookQuery,
  useBookUserStateQuery,
  useBookEngagementStatsQuery,
  useToggleBookmarkMutation,
  useAddBookToCollectionMutation,
  useRemoveBookFromCollectionMutation
} from "@/hooks";

export const BookDetailPage: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { bookId } = useParams<{ bookId: string }>();
  const queryClient = useQueryClient();
  const { user } = useAuthStore(useShallow((state) => ({ user: state.user })));
  const publicSettings = usePublicSettings();
  
  const [shareOpen, setShareOpen] = useState(false);
  const [imgError, setImgError] = useState(false);
  const { collections } = useLibraryStore(useShallow((state) => ({ collections: state.collections })));
  const [copied, setCopied] = useState(false);
  const [selectedFileId, setSelectedFileId] = useState<string>("");

  React.useEffect(() => {
    return () => {
      void queryClient.invalidateQueries({ queryKey: ["books"] });
      void queryClient.invalidateQueries({ queryKey: ["library"] });
    };
  }, [queryClient]);

  const { data: book, isLoading: isBookLoading, error: bookError } = useBookQuery(bookId || "");
  const { data: userState } = useBookUserStateQuery(bookId || "", !!user);
  const { data: engagementData } = useBookEngagementStatsQuery(bookId || "");

  const toggleBookmarkMutation = useToggleBookmarkMutation(bookId || "");
  const addBookToColMutation = useAddBookToCollectionMutation(bookId || "");
  const removeBookFromColMutation = useRemoveBookFromCollectionMutation(bookId || "");

  const meta = book ? parseMetadata(book.metadataJson) : {};
  const tags = toStringList(meta.subject);

  const guestPerms = publicSettings?.guest_permissions;
  const allowCollection = hasPermission(user, "book.collection", book?.libraryId, guestPerms);
  const allowBookmark = hasPermission(user, "book.bookmark", book?.libraryId, guestPerms);
  const allowShare = hasPermission(user, "book.share", book?.libraryId, guestPerms);
  const allowDownload = hasPermission(user, "book.download", book?.libraryId, guestPerms);
  const allowReview = hasPermission(user, "book.review.create", book?.libraryId, guestPerms);
  const allowRead = hasPermission(user, "book.read", book?.libraryId, guestPerms);
  const allowStats = hasPermission(user, "user.stats.read", book?.libraryId, guestPerms);

  const showReads = allowStats && allowRead;
  const showDownloads = allowStats && allowDownload;
  const showBookmarks = allowStats && allowBookmark;
  const showCollections = allowStats && allowCollection;
  const showRating = allowStats && allowReview;
  const showShares = allowStats && allowShare;

  const hasAnyStatToShow = showReads || showDownloads || showBookmarks || showCollections || showRating || showShares;

  const readStats = userState?.readStats || engagementData?.readStats;
  const downloadStats = userState?.downloadStats || engagementData?.downloadStats;
  const socialStats = userState?.socialStats || engagementData?.socialStats;
  const ratingSummary = userState?.ratingSummary || (engagementData?.socialStats ? {
    bookId: bookId || "",
    ratingCount: engagementData.socialStats.ratingCount,
    averageRating: engagementData.socialStats.averageRating,
  } : undefined);

  const shareUrl = window.location.href;

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(shareUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  };

  const isLoading = isBookLoading;

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center bg-base-100">
        <Loader2 className="w-10 h-10 animate-spin text-primary mb-4" />
        <p className="text-base-content/60">{t("common.loading", "Loading...")}</p>
      </div>
    );
  }

  if (bookError || !book) {
    return (
      <div className="flex flex-col h-screen items-center justify-center text-center p-6 bg-base-100">
        <Info className="w-12 h-12 text-error mb-4" />
        <h3 className="text-xl font-bold mb-2">{t("error.book_not_found", "Book not found")}</h3>
        <p className="text-base-content/60 mb-6">{t("error.book_not_found_desc", "This book might have been deleted or is unavailable.")}</p>
        <button className="btn btn-primary" onClick={() => navigate("/")}>
          <ArrowLeft className="w-5 h-5 mr-1" />
          {t("common.back", "Go Back")}
        </button>
      </div>
    );
  }

  const isBookmarked = userState?.bookmarked || false;
  return (
    <div className="w-full pb-8">
      {/* Action Bar */}
      <div className="flex items-center justify-between mb-4 mt-2">
        <button onClick={() => navigate("/")} className="btn btn-ghost btn-sm">
          <ArrowLeft className="w-4 h-4 mr-1" />
          {t("common.back", "Back")}
        </button>
        
        <div className="flex items-center gap-1">
          {allowCollection && (
            <div className="dropdown dropdown-end">
              <div 
                tabIndex={0} 
                role="button" 
                className={`btn btn-ghost btn-sm ${(userState?.collections?.length || 0) > 0 ? "text-primary" : ""}`}
              >
                {(userState?.collections?.length || 0) > 0 ? <FolderCheck className="w-4 h-4" /> : <FolderPlus className="w-4 h-4" />}
                <span className="hidden sm:inline ml-1">
                  {t("book.collection", "Collection")}
                </span>
              </div>
              <ul tabIndex={0} className="dropdown-content z-[1] menu p-2 shadow bg-base-100 rounded-box w-52 border border-base-200 mt-1 max-h-60 overflow-y-auto flex-nowrap block">
                {collections.length === 0 && (
                  <li className="px-4 py-2 text-sm text-base-content/50 text-center">
                    {t("library.no_collections", "No collections")}
                  </li>
                )}
                {collections.map(col => {
                  const isInCol = userState?.collections?.includes(col.id);
                  return (
                    <li key={col.id}>
                      <a 
                        onClick={() => {
                          if (isInCol) {
                            removeBookFromColMutation.mutate(col.id, {
                              onError: (err: any) => {
                                toast.error(err.message || t("error.unknown", "An unknown error occurred"));
                              }
                            });
                          } else {
                            addBookToColMutation.mutate(col.id, {
                              onError: (err: any) => {
                                toast.error(err.message || t("error.unknown", "An unknown error occurred"));
                              }
                            });
                          }
                        }}
                        className={isInCol ? "text-primary bg-primary/10" : ""}
                      >
                        {isInCol ? <Check className="w-4 h-4 mr-1" /> : <span className="w-4 mr-1 inline-block"></span>}
                        <span className="truncate">{col.name}</span>
                      </a>
                    </li>
                  );
                })}
              </ul>
            </div>
          )}
          {allowBookmark && (
            <button 
              onClick={() => toggleBookmarkMutation.mutate(!isBookmarked, {
                onSuccess: () => {
                  toast.success(t("book.bookmark_updated", "Bookmark updated successfully"));
                },
                onError: (err: any) => {
                  toast.error(err.message || t("error.unknown", "An unknown error occurred"));
                }
              })} 
              className={`btn btn-ghost btn-sm ${isBookmarked ? "text-primary" : ""}`}
              disabled={toggleBookmarkMutation.isPending}
            >
              {isBookmarked ? <BookmarkMinus className="w-4 h-4" /> : <BookmarkPlus className="w-4 h-4" />}
              <span className="hidden sm:inline ml-1">
                {isBookmarked ? t("book.remove_bookmark", "Remove Bookmark") : t("book.add_bookmark", "Bookmark")}
              </span>
            </button>
          )}
          {allowShare && (
            <button onClick={() => setShareOpen(true)} className="btn btn-ghost btn-sm">
              <Share2 className="w-4 h-4 mr-1" />
              {t("common.share", "Share")}
            </button>
          )}
        </div>
      </div>

      <div className="flex flex-col md:flex-row gap-6 lg:gap-8">
          {/* Left Pane - Cover & Stats */}
          <div className="w-full md:w-1/3 lg:w-1/4 flex flex-col items-center gap-6">
            <div className="w-48 md:w-full aspect-[2/3] rounded-xl overflow-hidden shadow-2xl border border-base-200 bg-base-200 relative group">
              {book.coverUrl && !imgError ? (
                <img
                  src={book.coverUrl}
                  alt={book.title}
                  onError={() => setImgError(true)}
                  className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
                />
              ) : (
                <div className="w-full h-full flex flex-col items-center justify-center text-base-content/30 p-4 text-center bg-gradient-to-br from-base-200 to-base-300">
                  <BookOpen className="w-12 h-12 mb-2 opacity-50" />
                  <span className="text-sm font-medium">{book.title}</span>
                </div>
              )}
            </div>

            {/* Engagement Stats */}
            {hasAnyStatToShow && (
              <div className="w-full grid grid-cols-3 gap-2 text-center bg-base-200/50 p-3 rounded-xl border border-base-200">
                {showReads && (
                  <div className="flex flex-col items-center p-1">
                    <Eye className="w-4 h-4 text-primary mb-1" />
                    <span className="font-bold text-base">{readStats?.totalOpenCount || 0}</span>
                    <span className="text-[11px] text-base-content/60">{t("book.reads", "Reads")}</span>
                  </div>
                )}
                {showDownloads && (
                  <div className="flex flex-col items-center p-1">
                    <Download className="w-4 h-4 text-secondary mb-1" />
                    <span className="font-bold text-base">{downloadStats?.totalDownloadCount || 0}</span>
                    <span className="text-[11px] text-base-content/60">{t("book.downloads", "Downloads")}</span>
                  </div>
                )}
                {showBookmarks && (
                  <div className="flex flex-col items-center p-1">
                    <Bookmark className="w-4 h-4 text-accent mb-1" />
                    <span className="font-bold text-base">{socialStats?.bookmarkCount || 0}</span>
                    <span className="text-[11px] text-base-content/60">{t("book.bookmarks", "Bookmarks")}</span>
                  </div>
                )}
                {showCollections && (
                  <div className="flex flex-col items-center p-1">
                    <FolderPlus className="w-4 h-4 text-success mb-1" />
                    <span className="font-bold text-base">{socialStats?.collectionCount ?? userState?.collections?.length ?? 0}</span>
                    <span className="text-[11px] text-base-content/60">{t("book.collections", "Collections")}</span>
                  </div>
                )}
                {showRating && (
                  <div className="flex flex-col items-center p-1">
                    <Star className="w-4 h-4 text-warning mb-1" />
                    <span className="font-bold text-base">{ratingSummary?.averageRating ? ratingSummary.averageRating.toFixed(1) : "0.0"}</span>
                    <span className="text-[11px] text-base-content/60">
                      {ratingSummary?.ratingCount ? `(${ratingSummary.ratingCount})` : t("book.rating", "Rating")}
                    </span>
                  </div>
                )}
                {showShares && (
                  <div className="flex flex-col items-center p-1">
                    <Share2 className="w-4 h-4 text-info mb-1" />
                    <span className="font-bold text-base">{socialStats?.shareCount || 0}</span>
                    <span className="text-[11px] text-base-content/60">{t("common.share", "Shares")}</span>
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Right Pane - Details & Files & Reviews */}
          <div className="w-full md:w-2/3 lg:w-3/4 flex flex-col gap-4">
            <div>
              <h1 className="text-3xl md:text-4xl font-extrabold text-base-content mb-2 leading-tight">
                {book.title}
              </h1>
              {meta.series && (
                <div 
                  className="badge badge-primary badge-outline mt-1 mb-2 text-sm px-3 py-1 h-auto text-left whitespace-normal leading-tight cursor-pointer hover:bg-primary hover:text-primary-content"
                  onClick={() => navigate(`/?nav=series&facet=series&name=${encodeURIComponent(meta.series || '')}`)}
                >
                  {meta.series} {meta.seriesIndex ? `#${meta.seriesIndex}` : ""}
                </div>
              )}
            </div>

            {/* Tags immediately below title */}
            {tags.length > 0 && (
              <div className="flex flex-wrap gap-2 my-1">
                {tags.map((tag, idx) => (
                  <span
                    key={idx}
                    className="badge badge-secondary badge-outline px-3 py-3 font-medium text-xs hover:bg-secondary hover:text-secondary-content transition-colors cursor-pointer shadow-sm"
                    onClick={() => navigate(`/?nav=tags&facet=tag&name=${encodeURIComponent(tag)}`)}
                  >
                    <Tag className="w-3 h-3 mr-1 opacity-70" />
                    {tag}
                  </span>
                ))}
              </div>
            )}

            {/* Author, Publisher, Language Info */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-3 my-4">
              <InfoLine
                icon={<User />}
                label={t("book.author", "Author")}
                value={
                  <span 
                    className="text-primary font-medium cursor-pointer hover:underline"
                    onClick={() => navigate(`/?nav=authors&facet=author&name=${encodeURIComponent(book.authorName || meta.creator || '')}`)}
                  >
                    {book.authorName || meta.creator || t("common.unknown", "Unknown")}
                  </span>
                }
              />
              <InfoLine
                icon={<Building />}
                label={t("book.publisher", "Publisher")}
                value={
                  <span 
                    className="text-primary font-medium cursor-pointer hover:underline"
                    onClick={() => navigate(`/?nav=publishers&facet=publisher&name=${encodeURIComponent(meta.publisher || '')}`)}
                  >
                    {meta.publisher || t("common.unknown", "Unknown")}
                  </span>
                }
              />
              <InfoLine
                icon={<Calendar />}
                label={t("book.date", "Published Date")}
                value={<span className="font-medium text-base-content/80">{meta.date || t("common.unknown", "Unknown")}</span>}
              />
              <InfoLine
                icon={<Globe />}
                label={t("book.language", "Language")}
                value={
                  <span 
                    className="text-primary font-medium cursor-pointer hover:underline uppercase"
                    onClick={() => navigate(`/?nav=languages&facet=language&name=${encodeURIComponent(meta.language || '')}`)}
                  >
                    {meta.language || t("common.unknown", "Unknown")}
                  </span>
                }
              />
            </div>

            {/* Quick Actions (Read/Download) */}
            {(allowRead || allowDownload) && book.files && book.files.length > 0 && (
              <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 my-2">
                <select
                  className="select select-bordered select-md w-full sm:max-w-[240px] font-medium text-ellipsis overflow-hidden whitespace-nowrap"
                  value={selectedFileId || book.files[0].id}
                  onChange={(e) => setSelectedFileId(e.target.value)}
                >
                  {book.files.map((file) => {
                    const filename = file.path.split('/').pop() || file.path;
                    const truncated = filename.length > 25 
                      ? `${filename.slice(0, 18)}...${filename.slice(filename.lastIndexOf('.'))}`
                      : filename;
                    return (
                      <option key={file.id} value={file.id}>
                        {truncated}
                      </option>
                    );
                  })}
                </select>

                <div className="flex gap-2 w-full sm:w-auto shrink-0">
                  {allowRead && (
                    <button
                      onClick={() => navigate(`/reader/${encodeURIComponent(book.id)}?fileId=${encodeURIComponent(selectedFileId || book.files![0].id)}`)}
                      className="btn btn-primary btn-md flex-1 sm:w-[140px] whitespace-nowrap"
                      disabled={!book.files.length}
                    >
                      <BookOpen className="w-5 h-5 mr-1 shrink-0" />
                      {t("reader.read", "Read")}
                    </button>
                  )}
                  {allowDownload && (
                    <a
                      href={bookService.getDownloadUrl(book.id, selectedFileId || book.files[0].id)}
                      className="btn btn-outline btn-md flex-1 sm:w-[140px] whitespace-nowrap"
                      download
                      onClick={(e) => {
                        if (!book.files?.length) e.preventDefault();
                      }}
                    >
                      <Download className="w-5 h-5 mr-1 shrink-0" />
                      {t("common.download", "Download")}
                    </a>
                  )}
                </div>
              </div>
            )}

            {/* Synopsis */}
            <div className="mt-4">
              <h3 className="text-xl font-bold mb-3 flex items-center gap-2">
                {t("book.description", "Synopsis")}
              </h3>
              <div
                className="prose prose-sm md:prose-base dark:prose-invert max-w-none text-base-content/80 leading-relaxed"
                dangerouslySetInnerHTML={{
                  __html: DOMPurify.sanitize(book.description || meta.description || t("book.no_description", "No description available.")),
                }}
              />
            </div>


            {/* Reviews Section */}
            {allowReview && (
              <div className="pt-2 border-t border-base-200 mt-2">
                <ReviewSection bookId={book.id!} userReview={userState?.myReview} />
              </div>
            )}
          </div>
      </div>

      {allowShare && (
        <ShareDialog
          open={shareOpen}
          book={book}
          shareUrl={shareUrl}
          copied={copied}
          t={t}
          onClose={() => setShareOpen(false)}
          onCopy={handleCopy}
        />
      )}
    </div>
  );
};
