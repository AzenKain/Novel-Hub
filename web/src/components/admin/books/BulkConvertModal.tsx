import { FileText, X, AlertTriangle, Loader2 } from "lucide-react";
import React, { useState, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";

import { useBulkConvertBookMutation } from "@/hooks";
import { getMediaUrl } from "@/config/api";
import type { Book, BookFile } from "@/types";

type BulkConvertModalProps = {
  open: boolean;
  books: Book[];
  onClose: () => void;
};

const CONVERT_TARGETS = [
  "epub",
  "fb2",
  "txt",
  "docx",
  "cbz",
  "kepub.epub",
  "mobi",
  "azw",
  "pdf",
];

const formatBytes = (n: number) => {
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(0)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
};

interface BookConvertState {
  bookId: string;
  selectedFileId: string;
  targetFormat: string;
  overwriteChecked: boolean;
}

export const BulkConvertModal: React.FC<BulkConvertModalProps> = ({
  open,
  books,
  onClose,
}) => {
  const { t } = useTranslation();
  const bulkConvertMutation = useBulkConvertBookMutation();

  const [states, setStates] = useState<Record<string, BookConvertState>>(() => {
    const initial: Record<string, BookConvertState> = {};
    books.forEach((book) => {
      initial[book.id] = {
        bookId: book.id,
        selectedFileId: book.files?.[0]?.id || "",
        targetFormat: "epub",
        overwriteChecked: false,
      };
    });
    return initial;
  });

  const updateBookState = (
    bookId: string,
    updates: Partial<BookConvertState>,
  ) => {
    setStates((prev) => ({
      ...prev,
      [bookId]: {
        ...prev[bookId],
        ...updates,
      },
    }));
  };

  const handleSetGlobalFormat = (format: string) => {
    setStates((prev) => {
      const next = { ...prev };
      for (const id in next) {
        next[id] = {
          ...next[id],
          targetFormat: format,
          overwriteChecked: false,
        };
      }
      return next;
    });
  };

  const needsConfirmation = useMemo(() => {
    return books.some((book) => {
      const state = states[book.id];
      if (!state) return false;
      const hasFiles = book.files && book.files.length > 0;
      if (!hasFiles) return false;
      const hasDuplicate =
        book.files?.some(
          (f) => f.format.toLowerCase() === state.targetFormat.toLowerCase(),
        ) || false;
      return hasDuplicate && !state.overwriteChecked;
    });
  }, [books, states]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const items = books
      .map((book) => {
        const state = states[book.id];
        if (!state || !state.selectedFileId) return null;
        return {
          book_id: book.id,
          file_id: state.selectedFileId,
          target_format: state.targetFormat,
        };
      })
      .filter(Boolean) as {
      book_id: string;
      file_id: string;
      target_format: string;
    }[];

    if (items.length === 0) {
      toast.error(t("book.no_files_convert", "No files available to convert"));
      return;
    }

    bulkConvertMutation.mutate(
      { items },
      {
        onSuccess: (res) => {
          const count = res?.job_ids?.length || items.length;
          toast.success(
            t(
              "book.bulk_convert_success",
              "Enqueued {{count}} conversion jobs successfully",
              { count },
            ),
          );
          onClose();
        },
        onError: (err) => {
          toast.error(
            err instanceof Error
              ? err.message
              : t(
                  "book.bulk_convert_failed",
                  "Failed to start bulk conversion",
                ),
          );
        },
      },
    );
  };

  if (!open) return null;

  return (
    <dialog className="modal modal-open">
      <div className="modal-box w-[96vw] sm:w-11/12 max-w-2xl max-h-[80dvh] sm:max-h-[85vh] p-0 overflow-hidden flex flex-col rounded-2xl sm:rounded-3xl border border-base-300 shadow-2xl">
        {/* Header */}
        <div className="px-4 sm:px-6 py-3.5 sm:py-4 border-b border-base-200 flex items-center justify-between gap-3 shrink-0 bg-base-100">
          <div className="flex items-center gap-2.5 min-w-0">
            <div className="p-2 rounded-xl bg-primary/10 text-primary shrink-0">
              <FileText className="w-5 h-5" />
            </div>
            <h3 className="text-base sm:text-lg font-black text-base-content truncate">
              {t("book.bulk_convert_title", "Bulk Convert Books")}
            </h3>
          </div>
          <button
            type="button"
            className="btn btn-circle btn-sm btn-ghost -mr-1.5 text-base-content/70 hover:text-base-content shrink-0"
            onClick={onClose}
            aria-label={t("common.close", "Close")}
          >
            <X className="w-4 h-4 sm:w-5 sm:h-5 stroke-[2.5]" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-4 sm:p-6 overflow-y-auto flex-1 min-h-0 space-y-4">
          {/* Global Target Format Selection */}
          <div className="p-3 sm:p-3.5 bg-base-200/50 rounded-xl sm:rounded-2xl border border-base-300">
            <label className="text-xs font-bold text-base-content/80 block mb-2">
              {t(
                "book.bulk_convert_global_format",
                "Set target format for all selected books:",
              )}
            </label>
            <div className="flex flex-wrap gap-1.5 sm:gap-2">
              {CONVERT_TARGETS.map((target) => (
                <button
                  key={target}
                  type="button"
                  onClick={() => handleSetGlobalFormat(target)}
                  className="btn btn-xs sm:btn-sm btn-outline btn-primary uppercase font-bold rounded-lg"
                >
                  {target}
                </button>
              ))}
            </div>
          </div>

          {/* Individual Book List Settings */}
          <div className="space-y-3">
            {books.map((book) => {
              const state = states[book.id];
              if (!state) return null;

              const hasFiles = book.files && book.files.length > 0;
              const hasDuplicate =
                hasFiles &&
                (book.files?.some(
                  (f) =>
                    f.format.toLowerCase() === state.targetFormat.toLowerCase(),
                ) ||
                  false);

              return (
                <div
                  key={book.id}
                  className="p-3 sm:p-3.5 border border-base-200 bg-base-100/60 rounded-xl sm:rounded-2xl flex flex-col gap-2.5 shadow-xs"
                >
                  {/* Book Info Row */}
                  <div className="flex items-center gap-3 min-w-0">
                    {book.cover_url ? (
                      <img
                        src={getMediaUrl(book.cover_url)}
                        alt={book.title}
                        className="w-10 h-14 sm:w-11 sm:h-16 object-cover rounded-lg shadow-xs border border-base-200 shrink-0 aspect-2/3"
                      />
                    ) : (
                      <div className="w-10 h-14 sm:w-11 sm:h-16 bg-base-200 rounded-lg flex items-center justify-center text-[9px] font-bold text-base-content/50 border border-base-300 shrink-0 aspect-2/3">
                        NO COVER
                      </div>
                    )}
                    <div className="min-w-0 flex-1">
                      <h4
                        className="text-xs sm:text-sm font-bold truncate text-base-content"
                        title={book.title}
                      >
                        {book.title}
                      </h4>
                      <p className="text-[11px] sm:text-xs text-base-content/60 truncate mt-0.5">
                        {book.author_name ||
                          t("library.unknown_author", "Unknown")}
                      </p>
                    </div>
                  </div>

                  {/* Controls Row: 2-column on mobile */}
                  {hasFiles ? (
                    <div className="grid grid-cols-2 gap-2 w-full pt-2 border-t border-base-200/70">
                      {/* Source File */}
                      <div className="flex flex-col gap-1 min-w-0">
                        <label className="text-[10px] font-bold uppercase tracking-wider text-base-content/60 truncate">
                          {t("book.select_source_file", "Source file")}
                        </label>
                        <select
                          value={state.selectedFileId}
                          onChange={(e) =>
                            updateBookState(book.id, {
                              selectedFileId: e.target.value,
                            })
                          }
                          className="select select-bordered select-xs sm:select-sm w-full font-medium rounded-lg text-xs"
                        >
                          {book.files?.map((f) => (
                            <option key={f.id} value={f.id}>
                              {f.format.toUpperCase()} ({formatBytes(f.size_bytes)})
                            </option>
                          ))}
                        </select>
                      </div>

                      {/* Target Format */}
                      <div className="flex flex-col gap-1 min-w-0">
                        <label className="text-[10px] font-bold uppercase tracking-wider text-base-content/60 truncate">
                          {t("book.select_target_format", "Target format")}
                        </label>
                        <select
                          value={state.targetFormat}
                          onChange={(e) =>
                            updateBookState(book.id, {
                              targetFormat: e.target.value,
                              overwriteChecked: false,
                            })
                          }
                          className="select select-bordered select-xs sm:select-sm w-full font-bold uppercase text-primary rounded-lg text-xs"
                        >
                          {CONVERT_TARGETS.map((t) => (
                            <option key={t} value={t}>
                              {t.toUpperCase()}
                            </option>
                          ))}
                        </select>
                      </div>
                    </div>
                  ) : (
                    <div className="text-xs text-error font-medium pt-1">
                      {t("book.no_files_convert", "No files to convert")}
                    </div>
                  )}

                  {/* Duplicate Alert Row */}
                  {hasDuplicate && (
                    <div className="bg-warning/10 border border-warning/30 text-xs p-2.5 sm:p-3 rounded-xl flex flex-col gap-2">
                      <div className="flex items-start gap-2 text-base-content leading-snug">
                        <AlertTriangle className="w-4 h-4 text-warning shrink-0 mt-0.5" />
                        <span>
                          {t(
                            "book.convert_replace_warning",
                            "This book already has a {{format}} file. Continuing will overwrite the existing file.",
                            { format: state.targetFormat.toUpperCase() },
                          )}
                        </span>
                      </div>
                      <label className="flex items-center gap-2 cursor-pointer font-semibold select-none text-base-content self-start bg-warning/15 hover:bg-warning/25 px-2.5 py-1.5 rounded-lg transition-colors text-xs">
                        <input
                          type="checkbox"
                          className="checkbox checkbox-xs checkbox-warning rounded"
                          checked={state.overwriteChecked}
                          onChange={(e) =>
                            updateBookState(book.id, {
                              overwriteChecked: e.target.checked,
                            })
                          }
                        />
                        <span className="whitespace-nowrap">
                          {t("book.convert_confirm_replace", "Yes, replace")}
                        </span>
                      </label>
                    </div>
                  )}
                </div>
              );
            })}
          </div>

          <div className="flex items-center justify-end gap-2 pt-3 border-t border-base-200 mt-4">
            <button
              type="button"
              className="btn btn-ghost btn-sm sm:btn-md rounded-lg sm:rounded-xl font-semibold px-3 sm:px-4"
              onClick={onClose}
              disabled={bulkConvertMutation.isPending}
            >
              {t("common.cancel", "Cancel")}
            </button>
            <button
              type="submit"
              className="btn btn-primary btn-sm sm:btn-md gap-1.5 sm:gap-2 font-bold rounded-lg sm:rounded-xl px-4 sm:px-6 shadow-md shadow-primary/20 text-xs sm:text-sm whitespace-nowrap"
              disabled={bulkConvertMutation.isPending || needsConfirmation}
            >
              {bulkConvertMutation.isPending ? (
                <>
                  <Loader2 className="w-3.5 h-3.5 sm:w-4 sm:h-4 animate-spin" />
                  <span>{t("common.loading", "Loading...")}</span>
                </>
              ) : (
                <span>{t("book.bulk_convert_start", "Start")}</span>
              )}
            </button>
          </div>
        </form>
      </div>
      <form method="dialog" className="modal-backdrop" onClick={onClose}>
        <button type="button">{t("common.close", "Close")}</button>
      </form>
    </dialog>
  );
};
