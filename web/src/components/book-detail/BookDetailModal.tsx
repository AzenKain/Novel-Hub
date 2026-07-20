import React, { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import {
  BookOpen,
  Calendar,
  Download,
  Info,
  Loader2,
  Tag,
  User,
  X,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import DOMPurify from "dompurify";
import { bookService } from "@/services";
import { parseMetadata, toStringList } from "@/lib/bookDetail";
import { InfoLine } from "./InfoLine";

interface BookDetailModalProps {
  bookId: string;
  onClose: () => void;
}

export const BookDetailModal: React.FC<BookDetailModalProps> = ({
  bookId,
  onClose,
}) => {
  const { t } = useTranslation();
  const navigate = useNavigate();

  const { data: bookData, isLoading, error } = useQuery({
    queryKey: ["book", bookId],
    queryFn: async () => {
      const res = await bookService.getBook(bookId);
      if (!res.status) throw new Error(res.message || "Failed to fetch book");
      return res.data;
    },
    enabled: !!bookId,
  });

  // Lock body scroll when modal is open
  useEffect(() => {
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = "auto";
    };
  }, []);

  const book = bookData;
  const meta = book ? parseMetadata(book.metadataJson) : {};
  const tags = toStringList(meta.subject);

  // Close when clicking backdrop
  const handleBackdropClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 sm:p-6"
      onClick={handleBackdropClick}
      role="dialog"
      aria-modal="true"
    >
      <div className="relative w-full max-w-5xl max-h-[90vh] bg-base-100 rounded-2xl shadow-2xl overflow-hidden flex flex-col animate-in fade-in zoom-in-95 duration-200">
        {/* Header / Close button */}
        <button
          onClick={onClose}
          className="absolute top-4 right-4 z-10 btn btn-sm btn-circle btn-ghost bg-base-200/50 hover:bg-base-300"
          aria-label={t("common.close", "Close")}
        >
          <X className="w-5 h-5" />
        </button>

        {isLoading ? (
          <div className="flex-1 flex flex-col items-center justify-center min-h-[400px]">
            <Loader2 className="w-10 h-10 animate-spin text-primary mb-4" />
            <p className="text-base-content/60">
              {t("common.loading", "Loading...")}
            </p>
          </div>
        ) : error || !book ? (
          <div className="flex-1 flex flex-col items-center justify-center min-h-[400px] text-center p-6">
            <Info className="w-12 h-12 text-error mb-4" />
            <h3 className="text-xl font-bold mb-2">
              {t("error.book_not_found", "Book not found")}
            </h3>
            <p className="text-base-content/60 mb-6">
              {t("error.book_not_found_desc", "This book might have been deleted or is unavailable.")}
            </p>
            <button className="btn btn-primary" onClick={onClose}>
              {t("common.back", "Go Back")}
            </button>
          </div>
        ) : (
          <div className="flex flex-col md:flex-row h-full overflow-y-auto">
            {/* Left Pane - Cover & Actions */}
            <div className="w-full md:w-1/3 lg:w-1/4 p-6 bg-base-200/30 flex flex-col items-center gap-6">
              <div className="w-48 md:w-full aspect-[2/3] rounded-xl overflow-hidden shadow-lg border border-base-200 bg-base-200 relative group">
                {book.coverUrl ? (
                  <img
                    src={book.coverUrl}
                    alt={book.title}
                    className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
                  />
                ) : (
                  <div className="w-full h-full flex flex-col items-center justify-center text-base-content/30 p-4 text-center bg-gradient-to-br from-base-200 to-base-300">
                    <BookOpen className="w-12 h-12 mb-2 opacity-50" />
                    <span className="text-sm font-medium">{book.title}</span>
                  </div>
                )}
              </div>

              <div className="flex flex-col w-full gap-3">
                <button
                  onClick={() => navigate(`/reader/${encodeURIComponent(book.id)}`)}
                  className="btn btn-primary w-full shadow-lg shadow-primary/20 hover:shadow-primary/40 transition-shadow"
                >
                  <BookOpen className="w-5 h-5 mr-1" />
                  {t("reader.read_now", "Read Now")}
                </button>
                {book.files && book.files.length > 0 && (
                  <a
                    href={bookService.getDownloadUrl(book.id)}
                    className="btn btn-outline w-full"
                    download
                  >
                    <Download className="w-5 h-5 mr-1" />
                    {t("common.download", "Download")}
                  </a>
                )}
              </div>
            </div>

            {/* Right Pane - Details */}
            <div className="w-full md:w-2/3 lg:w-3/4 p-6 md:p-8 flex flex-col gap-6">
              <div>
                <h2 className="text-3xl md:text-4xl font-extrabold text-base-content mb-2 leading-tight">
                  {book.title}
                </h2>
                {meta.series && (
                  <div className="badge badge-primary badge-outline mt-1 mb-3">
                    {meta.series} {meta.seriesIndex ? `#${meta.seriesIndex}` : ""}
                  </div>
                )}
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 bg-base-200/50 p-4 rounded-xl border border-base-200/50">
                <InfoLine
                  icon={<User className="text-primary" />}
                  label={t("book.author", "Author")}
                  value={book.authorName || meta.creator || t("common.unknown", "Unknown")}
                />
                <InfoLine
                  icon={<Calendar className="text-secondary" />}
                  label={t("book.date", "Published Date")}
                  value={meta.date || t("common.unknown", "Unknown")}
                />
                <InfoLine
                  icon={<Info className="text-accent" />}
                  label={t("book.publisher", "Publisher")}
                  value={meta.publisher || t("common.unknown", "Unknown")}
                />
                <InfoLine
                  icon={<Tag className="text-info" />}
                  label={t("book.language", "Language")}
                  value={meta.language || t("common.unknown", "Unknown")}
                />
              </div>

              <div className="flex-1">
                <h3 className="text-lg font-bold mb-3 flex items-center gap-2">
                  {t("book.description", "Synopsis")}
                </h3>
                <div
                  className="prose prose-sm md:prose-base dark:prose-invert max-w-none text-base-content/80 leading-relaxed"
                  dangerouslySetInnerHTML={{
                    __html: DOMPurify.sanitize(book.description || meta.description || t("book.no_description", "No description available.")),
                  }}
                />
              </div>

              {tags.length > 0 && (
                <div className="pt-4 border-t border-base-200">
                  <div className="flex flex-wrap gap-2">
                    {tags.map((tag, idx) => (
                      <span
                        key={idx}
                        className="badge badge-ghost badge-sm py-3 px-3 hover:bg-base-300 transition-colors"
                      >
                        {tag}
                      </span>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
