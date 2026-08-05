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
  RotateCcw,
  CloudDownload,
  CheckCircle2,
  Send,
  FileText
} from "lucide-react";
import { useTranslation } from "react-i18next";
import DOMPurify from "dompurify";
import { getMediaUrl } from "@/config/api";
import { bookService, featureService } from "@/services";
import { parseMetadata, toStringList } from "@/lib/bookDetail";
import { InfoLine, ShareDialog, ReviewSection, TrackerMapCard, SendToKindleModal, SeriesBooksSection } from "@/components/book-detail";
import { OfflineWarningModal, offlineWarningSuppressed } from "@/components/common";
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
  useRemoveBookFromCollectionMutation,
  useOfflineBook,
  useBookSeriesQuery
} from "@/hooks";

export const BookDetailPage: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { book_id } = useParams<{ book_id: string }>();
  const queryClient = useQueryClient();
  const { user } = useAuthStore(useShallow((state) => ({ user: state.user })));
  const publicSettings = usePublicSettings();
  
  const [shareOpen, setShareOpen] = useState(false);
  const [sendModalOpen, setSendModalOpen] = useState(false);
  const [imgError, setImgError] = useState(false);
  const { collections } = useLibraryStore(useShallow((state) => ({ collections: state.collections })));
  const [copied, setCopied] = useState(false);
  const [selectedFileId, setSelectedFileId] = useState<string>("");
  const offline = useOfflineBook(book_id, selectedFileId || undefined);
  const { data: seriesContext } = useBookSeriesQuery(book_id || "");
  const [offlineWarningOpen, setOfflineWarningOpen] = useState(false);

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
  const seriesEntry = seriesContext?.series?.[0];
  const nextInSeries = seriesContext?.next;

  const guestPerms = publicSettings?.guest_permissions;
  const allowCollection = hasPermission(user, "book.collection", book?.library_id, guestPerms);
  const allowBookmark = hasPermission(user, "book.bookmark", book?.library_id, guestPerms);
  const allowShare = hasPermission(user, "book.share", book?.library_id, guestPerms);
  const allowSendEmail = hasPermission(user, "book.send_email", book?.library_id, guestPerms);
  const allowDownload = hasPermission(user, "book.download", book?.library_id, guestPerms);
  const allowOffline = hasPermission(user, "book.offline", book?.library_id, guestPerms);
  const offlineSizeBytes = (book?.files || []).find((file) => file.id === (selectedFileId || book?.files?.[0]?.id))?.size_bytes;
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

  const handleShare = async () => {
    setShareOpen(true);
    if (book_id) {
      try {
        await featureService.recordShare(book_id);
        void queryClient.invalidateQueries({ queryKey: ["bookUserState", book_id] });
        void queryClient.invalidateQueries({ queryKey: ["bookEngagementStats", book_id] });
      } catch (e) {
        // ignore
      }
    }
  };

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(shareUrl);
      setCopied(true);
      if (book_id) {
        try {
          await featureService.recordShare(book_id);
          void queryClient.invalidateQueries({ queryKey: ["bookUserState", book_id] });
          void queryClient.invalidateQueries({ queryKey: ["bookEngagementStats", book_id] });
        } catch (e) {
          // ignore
        }
      }
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
          {allowSendEmail && (
            <button onClick={() => setSendModalOpen(true)} className="btn btn-ghost btn-sm">
              <Send className="w-4 h-4 mr-1" />
              {t("send_to_device", "Send to Device")}
            </button>
          )}
          {allowShare && (
            <button onClick={handleShare} className="btn btn-ghost btn-sm">
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
              <div className="w-full grid grid-cols-3 gap-2 text-center bg-base-100 p-3 rounded-xl border border-base-200 shadow-2xs">
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
              {seriesEntry && (
                <div
                  className="badge badge-primary badge-outline mt-1 mb-2 text-sm px-3 py-1 h-auto text-left whitespace-normal leading-tight cursor-pointer hover:bg-primary hover:text-primary-content"
                  onClick={() =>
                    navigate(
                      `/?nav=series&facet=series&facet_id=${encodeURIComponent(seriesEntry.series_id)}&name=${encodeURIComponent(seriesEntry.series_name)}`
                    )
                  }
                >
                  {seriesEntry.series_name} {seriesEntry.series_index ? `#${seriesEntry.series_index}` : ""}
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

            {/* Helper for smart file truncation: e.g. "Tap 09 - After St...epub" */}
            {(() => {
              const formatTruncatedFilename = (path: string, maxLength = 22) => {
                const filename = path.split('/').pop() || path;
                const extIdx = filename.lastIndexOf('.');
                if (extIdx <= 0 || filename.length <= maxLength) return filename;
                const ext = filename.slice(extIdx);
                const base = filename.slice(0, extIdx);
                const keep = Math.max(4, maxLength - ext.length - 3);
                return `${base.slice(0, keep)}...${ext}`;
              };

              const hasReadingHistory = Boolean(
                user &&
                readingProgress &&
                (readingProgress.chapter_id || typeof readingProgress.chapter_index === "number")
              );

              return (
                <div className="flex flex-col gap-2.5 my-3 w-full">
                  {/* Row 1: Primary Action & Smart Truncated File Dropdown */}
                  <div className="flex flex-wrap items-center gap-2.5 w-full">
                    {/* Primary Read CTA */}
                    {allowRead && (
                      <button
                        onClick={() =>
                          navigate(
                            `/reader/${encodeURIComponent(book.id)}?file_id=${encodeURIComponent(
                              selectedFileId || readingProgress?.file_id || book.files?.[0]?.id || ""
                            )}`
                          )
                        }
                        className="btn btn-primary btn-md gap-2 font-bold text-sm px-6 shadow-xs shrink-0"
                        disabled={!book.files?.length}
                      >
                        {hasReadingHistory ? (
                          <>
                            <Play className="w-4 h-4 fill-current shrink-0" />
                            <span className="whitespace-nowrap">{t("reader.continue_reading", "Continue Reading")}</span>
                          </>
                        ) : (
                          <>
                            <BookOpen className="w-4 h-4 shrink-0" />
                            <span className="whitespace-nowrap">{t("reader.read", "Read Now")}</span>
                          </>
                        )}
                      </button>
                    )}

                    {/* Smart Truncated File Format Dropdown */}
                    <select
                      className="select select-bordered select-md font-medium max-w-[210px] shrink-0"
                      value={selectedFileId || book.files?.[0]?.id || ""}
                      onChange={(e) => setSelectedFileId(e.target.value)}
                    >
                      {(book.files || []).map((file) => {
                        return (
                          <option key={file.id} value={file.id} title={file.path.split('/').pop() || file.path}>
                            {formatTruncatedFilename(file.path, 22)}
                          </option>
                        );
                      })}
                    </select>
                  </div>

                  {/* Reading Progress Subtitle Line (Dedicated Pill Bar) */}
                  {hasReadingHistory && readingProgress && (
                    <div className="flex items-center gap-2 text-xs text-base-content/75 font-medium px-3 py-1.5 bg-base-200/60 rounded-lg w-fit max-w-full">
                      <span className="badge badge-primary badge-sm font-bold text-[10px] px-2 py-0.5 shrink-0">
                        {typeof readingProgress.progress_percent === "number" && readingProgress.progress_percent > 0
                          ? `${readingProgress.progress_percent}%`
                          : "0%"}
                      </span>
                      <span className="truncate max-w-[450px]" title={readingProgress.chapter_title || undefined}>
                        <span className="opacity-70 mr-1">{t("reader.reading_chapter", "Currently at")}:</span>
                        <strong className="text-base-content font-semibold">
                          {readingProgress.chapter_title || t("reader.chapter_num", "Chapter {{num}}", { num: (readingProgress.chapter_index || 0) + 1 })}
                        </strong>
                      </span>
                    </div>
                  )}

                  {/* Row 2: Secondary Utility Action Buttons Bar */}
                  <div className="flex flex-wrap items-center gap-2.5 pt-1.5">
                    {allowRead && hasReadingHistory && (
                      <button
                        onClick={() =>
                          navigate(
                            `/reader/${encodeURIComponent(book.id)}?file_id=${encodeURIComponent(
                              selectedFileId || book.files?.[0]?.id || ""
                            )}&start_over=true`
                          )
                        }
                        className="btn btn-outline btn-md h-10 min-h-[2.5rem] px-4 text-sm font-medium gap-2"
                        disabled={!book.files?.length}
                        title={t("reader.read_from_beginning", "Start from Beginning")}
                      >
                        <RotateCcw className="w-4 h-4 shrink-0" />
                        <span>{t("reader.start_over", "Start Over")}</span>
                      </button>
                    )}

                    {allowRead && allowOffline && (
                      <button
                        className="btn btn-outline btn-md h-10 min-h-[2.5rem] px-4 text-sm font-medium gap-2"
                        disabled={offline.status === "downloading" || offline.status === "unknown"}
                        onClick={() => {
                          if (offline.status === "ready") {
                            void offline.remove();
                          } else if (offlineWarningSuppressed()) {
                            void offline.download();
                          } else {
                            setOfflineWarningOpen(true);
                          }
                        }}
                        title={offline.error || undefined}
                      >
                        {offline.status === "downloading" ? (
                          <>
                            <span className="loading loading-spinner loading-xs" />
                            <span>{offline.progress}%</span>
                          </>
                        ) : offline.status === "ready" ? (
                          <>
                            <CheckCircle2 className="w-4 h-4 text-success shrink-0" />
                            <span>{t("offline.remove", "Remove offline copy")}</span>
                          </>
                        ) : (
                          <>
                            <CloudDownload className="w-4 h-4 shrink-0" />
                            <span>{t("offline.save", "Save for offline")}</span>
                          </>
                        )}
                      </button>
                    )}

                    {allowDownload && (
                      <a
                        href={bookService.getDownloadUrl(book.id, selectedFileId || book.files?.[0]?.id || "")}
                        className="btn btn-outline btn-md h-10 min-h-[2.5rem] px-4 text-sm font-medium gap-2"
                        download
                        onClick={(e) => {
                          if (!book.files?.length) e.preventDefault();
                          setTimeout(() => {
                            void queryClient.invalidateQueries({ queryKey: ["bookUserState", book_id] });
                            void queryClient.invalidateQueries({ queryKey: ["bookEngagementStats", book_id] });
                          }, 1000);
                        }}
                      >
                        <Download className="w-4 h-4 shrink-0" />
                        <span>{t("common.download", "Download")}</span>
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

            {/* Books in Same Series Section */}
            {seriesEntry && (
              <SeriesBooksSection
                currentBookId={book.id!}
                seriesId={seriesEntry.series_id}
                seriesName={seriesEntry.series_name}
                seriesIndex={seriesEntry.series_index}
              />
            )}


            {nextInSeries && (
              <div className="pt-2 border-t border-base-200 mt-2">
                <p className="text-xs font-semibold uppercase tracking-wide opacity-60 mb-2">
                  {t("book.next_in_series", "Next in series")}
                </p>
                <button
                  className="flex items-center gap-3 w-full text-left p-3 rounded-xl border border-base-300 bg-base-100 hover:border-primary transition-colors"
                  onClick={() => navigate(`/books/${encodeURIComponent(nextInSeries.book_id)}`)}
                >
                  {nextInSeries.cover_url ? (
                    <img
                      src={getMediaUrl(nextInSeries.cover_url)}
                      alt=""
                      className="w-10 h-14 object-cover rounded shrink-0"
                    />
                  ) : (
                    <BookOpen className="w-6 h-6 opacity-50 shrink-0" />
                  )}
                  <div className="min-w-0">
                    <p className="text-sm font-medium truncate">{nextInSeries.title}</p>
                    <p className="text-xs opacity-60 truncate">
                      {nextInSeries.series_name}
                      {nextInSeries.series_index ? ` #${nextInSeries.series_index}` : ""}
                    </p>
                  </div>
                </button>
              </div>
            )}

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

      {allowSendEmail && (
        <SendToKindleModal
          open={sendModalOpen}
          book={book}
          t={t}
          onClose={() => setSendModalOpen(false)}
        />
      )}

      <OfflineWarningModal
        open={offlineWarningOpen}
        title={book.title}
        sizeBytes={offlineSizeBytes}
        onCancel={() => setOfflineWarningOpen(false)}
        onConfirm={() => {
          setOfflineWarningOpen(false);
          void offline.download();
        }}
      />
    </div>
  );
};
