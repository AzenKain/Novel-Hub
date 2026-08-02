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
  Globe,
  Building,
  Play,
  RotateCcw
} from "lucide-react";
import { useTranslation } from "react-i18next";
import DOMPurify from "dompurify";
import { getMediaUrl } from "@/config/api";
import { bookService } from "@/services";
import { parseMetadata, toStringList } from "@/lib/bookDetail";
import { InfoLine, ShareDialog, ReviewSection, TrackerMapCard } from "@/components/book-detail";
import { usePublicSettings } from "@/hooks/useSettings";
import { hasPermission } from "@/utils/permission";
import { toast } from "react-toastify";
import { useLibraryStore, useAuthStore } from "@/stores";
import { useShallow } from "zustand/react/shallow";
import {
  useBookQuery,
  useBookUserStateQuery,
  useTrackerReadingProgressQuery,
  useBookEngagementStatsQuery,
  useToggleBookmarkMutation,
  useAddBookToCollectionMutation,
  useRemoveBookFromCollectionMutation
} from "@/hooks";

export const BookDetailPage: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { book_id } = useParams<{ book_id: string }>();
  const queryClient = useQueryClient();
  const { user } = useAuthStore(useShallow((state) => ({ user: state.user })));
  const publicSettings = usePublicSettings();
  
  const [shareOpen, setShareOpen] = useState(false);
  const [imgError, setImgError] = useState(false);
  const { collections } = useLibraryStore(useShallow((state) => ({ collections: state.collections })));
  const [copied, setCopied] = useState(false);
  const [selectedFileId, setSelectedFileId] = useState<string>("");

  React.useEffect(() => {
    if (book_id) {
      void queryClient.invalidateQueries({ queryKey: ["trackerReadingProgress", book_id] });
      void queryClient.invalidateQueries({ queryKey: ["bookUserState", book_id] });
    }
    return () => {
      void queryClient.invalidateQueries({ queryKey: ["books"] });
      void queryClient.invalidateQueries({ queryKey: ["library"] });
    };
  }, [book_id, queryClient]);

  const { data: book, isLoading: isBookLoading, error: bookError } = useBookQuery(book_id || "");
  const { data: userState } = useBookUserStateQuery(book_id || "", !!user);
  const { data: readingProgress } = useTrackerReadingProgressQuery(book_id || "");
  const { data: engagementData } = useBookEngagementStatsQuery(book_id || "");

  const toggleBookmarkMutation = useToggleBookmarkMutation(book_id || "");
  const addBookToColMutation = useAddBookToCollectionMutation(book_id || "");
  const removeBookFromColMutation = useRemoveBookFromCollectionMutation(book_id || "");

  const meta = book ? parseMetadata(book.metadata_json) : {};
  const tags = toStringList(meta.subject);

  const guestPerms = publicSettings?.guest_permissions;
  const allowCollection = hasPermission(user, "book.collection", book?.library_id, guestPerms);
  const allowBookmark = hasPermission(user, "book.bookmark", book?.library_id, guestPerms);
  const allowShare = hasPermission(user, "book.share", book?.library_id, guestPerms);
  const allowDownload = hasPermission(user, "book.download", book?.library_id, guestPerms);
  const allowReview = hasPermission(user, "book.review.create", book?.library_id, guestPerms);
  const allowRead = hasPermission(user, "book.read", book?.library_id, guestPerms);
  const allowStats = hasPermission(user, "user.stats.read", book?.library_id, guestPerms);

  const showReads = allowStats && allowRead;
  const showDownloads = allowStats && allowDownload;
  const showBookmarks = allowStats && allowBookmark;
  const showCollections = allowStats && allowCollection;
  const showRating = allowStats && allowReview;
  const showShares = allowStats && allowShare;

  const hasAnyStatToShow = showReads || showDownloads || showBookmarks || showCollections || showRating || showShares;

  const read_stats = userState?.read_stats || engagementData?.read_stats;
  const download_stats = userState?.download_stats || engagementData?.download_stats;
  const social_stats = userState?.social_stats || engagementData?.social_stats;
  const rating_summary = userState?.rating_summary || (engagementData?.social_stats ? {
    book_id: book_id || "",
    rating_count: engagementData.social_stats.rating_count,
    average_rating: engagementData.social_stats.average_rating,
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
              {book.cover_url && !imgError ? (
                <img
                  src={getMediaUrl(book.cover_url)}
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
                    <span className="font-bold text-base">{read_stats?.total_open_count || 0}</span>
                    <span className="text-[11px] text-base-content/60">{t("book.reads", "Reads")}</span>
                  </div>
                )}
                {showDownloads && (
                  <div className="flex flex-col items-center p-1">
                    <Download className="w-4 h-4 text-secondary mb-1" />
                    <span className="font-bold text-base">{download_stats?.total_download_count || 0}</span>
                    <span className="text-[11px] text-base-content/60">{t("book.downloads", "Downloads")}</span>
                  </div>
                )}
                {showBookmarks && (
                  <div className="flex flex-col items-center p-1">
                    <Bookmark className="w-4 h-4 text-accent mb-1" />
                    <span className="font-bold text-base">{social_stats?.bookmark_count || 0}</span>
                    <span className="text-[11px] text-base-content/60">{t("book.bookmarks", "Bookmarks")}</span>
                  </div>
                )}
                {showCollections && (
                  <div className="flex flex-col items-center p-1">
                    <FolderPlus className="w-4 h-4 text-success mb-1" />
                    <span className="font-bold text-base">{social_stats?.collection_count ?? userState?.collections?.length ?? 0}</span>
                    <span className="text-[11px] text-base-content/60">{t("book.collections", "Collections")}</span>
                  </div>
                )}
                {showRating && (
                  <div className="flex flex-col items-center p-1">
                    <Star className="w-4 h-4 text-warning mb-1" />
                    <span className="font-bold text-base">{rating_summary?.average_rating ? rating_summary.average_rating.toFixed(1) : "0.0"}</span>
                    <span className="text-[11px] text-base-content/60">
                      {rating_summary?.rating_count ? `(${rating_summary.rating_count})` : t("book.rating", "Rating")}
                    </span>
                  </div>
                )}
                {showShares && (
                  <div className="flex flex-col items-center p-1">
                    <Share2 className="w-4 h-4 text-info mb-1" />
                    <span className="font-bold text-base">{social_stats?.share_count || 0}</span>
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
                  {meta.series} {meta.series_index ? `#${meta.series_index}` : ""}
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
              {/* Author */}
              {(() => {
                const authorVal = book.author_name || meta.creator;
                return (
                  <InfoLine
                    icon={<User />}
                    label={t("book.author", "Author")}
                    value={
                      authorVal ? (
                        <span 
                          className="text-primary font-medium text-sm cursor-pointer hover:underline"
                          onClick={() => navigate(`/?nav=authors&facet=author&name=${encodeURIComponent(authorVal)}`)}
                        >
                          {authorVal}
                        </span>
                      ) : (
                        <span className="text-base-content/60 font-medium text-sm">
                          {t("common.unknown", "Unknown")}
                        </span>
                      )
                    }
                  />
                );
              })()}

              {/* Publisher */}
              {(() => {
                const publisherVal = meta.publisher;
                return (
                  <InfoLine
                    icon={<Building />}
                    label={t("book.publisher", "Publisher")}
                    value={
                      publisherVal ? (
                        <span 
                          className="text-primary font-medium text-sm cursor-pointer hover:underline"
                          onClick={() => navigate(`/?nav=publishers&facet=publisher&name=${encodeURIComponent(publisherVal)}`)}
                        >
                          {publisherVal}
                        </span>
                      ) : (
                        <span className="text-base-content/60 font-medium text-sm">
                          {t("common.unknown", "Unknown")}
                        </span>
                      )
                    }
                  />
                );
              })()}

              {/* Date */}
              {(() => {
                const dateVal = meta.date;
                return (
                  <InfoLine
                    icon={<Calendar />}
                    label={t("book.date", "Published Date")}
                    value={
                      dateVal ? (
                        <span className="text-base-content/90 font-medium text-sm">{dateVal}</span>
                      ) : (
                        <span className="text-base-content/60 font-medium text-sm">
                          {t("common.unknown", "Unknown")}
                        </span>
                      )
                    }
                  />
                );
              })()}

              {/* Language */}
              {(() => {
                const langVal = meta.language;
                return (
                  <InfoLine
                    icon={<Globe />}
                    label={t("book.language", "Language")}
                    value={
                      langVal ? (
                        <span 
                          className="text-primary font-medium text-sm cursor-pointer hover:underline uppercase"
                          onClick={() => navigate(`/?nav=languages&facet=language&name=${encodeURIComponent(langVal)}`)}
                        >
                          {langVal}
                        </span>
                      ) : (
                        <span className="text-base-content/60 font-medium text-sm">
                          {t("common.unknown", "Unknown")}
                        </span>
                      )
                    }
                  />
                );
              })()}
            </div>

            {/* Quick Actions (Read/Continue Reading/Download) */}
            {(allowRead || allowDownload) && book.files && book.files.length > 0 && (() => {
              const hasReadingHistory = Boolean(
                user &&
                readingProgress &&
                (readingProgress.chapter_id || typeof readingProgress.chapter_index === "number")
              );

              return (
                <div className="flex flex-col xl:flex-row items-stretch xl:items-center gap-3 my-3 w-full">
                  {/* File Select Dropdown */}
                  <select
                    className="select select-bordered select-md w-full xl:w-64 font-medium text-ellipsis overflow-hidden whitespace-nowrap shrink-0"
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

                  {/* Buttons Container */}
                  <div className="flex flex-col sm:flex-row flex-wrap items-stretch sm:items-center gap-2.5 flex-1 min-w-0">
                    {allowRead && (
                      <>
                        {hasReadingHistory ? (
                          <>
                            {/* Continue Reading Button */}
                            <button
                              onClick={() =>
                                navigate(
                                  `/reader/${encodeURIComponent(book.id)}?file_id=${encodeURIComponent(
                                    selectedFileId || readingProgress?.file_id || book.files![0].id
                                  )}`
                                )
                              }
                              className="btn btn-primary btn-md flex-1 px-4 gap-2 min-w-0 h-auto py-2.5 overflow-hidden text-left"
                              disabled={!book.files.length}
                            >
                              <Play className="w-4 h-4 fill-current shrink-0" />
                              <div className="flex flex-col items-start text-left leading-tight min-w-0 overflow-hidden flex-1">
                                <span className="font-bold text-sm truncate w-full whitespace-nowrap">
                                  {t("reader.continue_reading", "Continue Reading")}
                                </span>
                                <span
                                  className="block w-full truncate text-[11px] opacity-85 font-normal whitespace-nowrap"
                                  title={`${readingProgress?.chapter_title || `${t("reader.chapter", "Chapter")} ${(readingProgress?.chapter_index || 0) + 1}`}${
                                    typeof readingProgress?.progress_percent === "number" && readingProgress.progress_percent > 0
                                      ? ` (${readingProgress.progress_percent}%)`
                                      : ""
                                  }`}
                                >
                                  {readingProgress?.chapter_title || `${t("reader.chapter", "Chapter")} ${(readingProgress?.chapter_index || 0) + 1}`}
                                  {typeof readingProgress?.progress_percent === "number" && readingProgress.progress_percent > 0
                                    ? ` (${readingProgress.progress_percent}%)`
                                    : ""}
                                </span>
                              </div>
                            </button>

                            {/* Read from Beginning Button */}
                            <button
                              onClick={() =>
                                navigate(
                                  `/reader/${encodeURIComponent(book.id)}?file_id=${encodeURIComponent(
                                    selectedFileId || book.files![0].id
                                  )}&start_over=true`
                                )
                              }
                              className="btn btn-outline btn-md sm:w-auto shrink-0 gap-1.5 whitespace-nowrap"
                              disabled={!book.files.length}
                              title={t("reader.read_from_beginning", "Start from Beginning")}
                            >
                              <RotateCcw className="w-4 h-4 shrink-0" />
                              <span className="whitespace-nowrap">{t("reader.start_over", "Start Over")}</span>
                            </button>
                          </>
                        ) : (
                          /* First time reading: Single Read Button */
                          <button
                            onClick={() =>
                              navigate(
                                `/reader/${encodeURIComponent(book.id)}?file_id=${encodeURIComponent(
                                  selectedFileId || book.files![0].id
                                )}`
                              )
                            }
                            className="btn btn-primary btn-md flex-1 sm:w-auto shrink-0 whitespace-nowrap gap-2"
                            disabled={!book.files.length}
                          >
                            <BookOpen className="w-5 h-5 shrink-0" />
                            <span className="whitespace-nowrap">{t("reader.read", "Read")}</span>
                          </button>
                        )}
                      </>
                    )}

                    {allowDownload && (
                      <a
                        href={bookService.getDownloadUrl(book.id, selectedFileId || book.files[0].id)}
                        className="btn btn-outline btn-md sm:w-auto shrink-0 whitespace-nowrap gap-2"
                        download
                        onClick={(e) => {
                          if (!book.files?.length) e.preventDefault();
                        }}
                      >
                        <Download className="w-5 h-5 shrink-0" />
                        <span className="whitespace-nowrap">{t("common.download", "Download")}</span>
                      </a>
                    )}
                  </div>
                </div>
              );
            })()}

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
            <div className="pt-2 border-t border-base-200 mt-2">
              <TrackerMapCard book_id={book.id!} title={book.title} />
            </div>

            {allowReview && (
              <div className="pt-2 border-t border-base-200 mt-2">
                <ReviewSection book_id={book.id!} userReview={userState?.my_review} />
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
