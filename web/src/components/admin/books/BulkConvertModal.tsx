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
      <div className="modal-box max-w-3xl">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold flex items-center gap-2">
            <FileText className="w-5 h-5 text-primary" />
            {t("book.bulk_convert_title", "Bulk Convert Books")}
          </h3>
          <button
            className="btn btn-square btn-sm btn-ghost"
            onClick={onClose}
            aria-label={t("common.close")}
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Global Target Format Selection */}
          <div className="p-3 bg-base-200/50 rounded-2xl border border-base-300">
            <label className="text-xs font-bold text-base-content/80 block mb-2">
              {t(
                "book.bulk_convert_global_format",
                "Set target format for all selected books:",
              )}
            </label>
            <div className="flex flex-wrap gap-2">
              {CONVERT_TARGETS.map((target) => (
                <button
                  key={target}
                  type="button"
                  onClick={() => handleSetGlobalFormat(target)}
                  className="btn btn-xs sm:btn-sm btn-outline btn-primary uppercase font-bold"
                >
                  {target}
                </button>
              ))}
            </div>
          </div>

          {/* Individual Book List Settings */}
          <div className="max-h-96 overflow-y-auto space-y-3.5 pr-1">
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
                  className="p-3 border border-base-200 bg-base-100/50 rounded-xl flex flex-col gap-2 shadow-xs"
                >
                  <div className="flex items-center justify-between gap-3">
                    {/* Book Info */}
                    <div className="flex items-center gap-3 flex-1 min-w-0">
                      {book.cover_url ? (
                        <img
                          src={getMediaUrl(book.cover_url)}
                          alt={book.title}
                          className="w-10 h-14 object-cover rounded shadow-xs shrink-0"
                        />
                      ) : (
                        <div className="w-10 h-14 bg-base-200 rounded flex items-center justify-center text-[10px] font-bold text-base-content/50 border border-base-300 shrink-0">
                          NO COVER
                        </div>
                      )}
                      <div className="min-w-0">
                        <h4
                          className="text-sm font-bold truncate text-base-content"
                          title={book.title}
                        >
                          {book.title}
                        </h4>
                        <p className="text-xs text-base-content/60 truncate">
                          {book.author_name ||
                            t("library.unknown_author", "Unknown")}
                        </p>
                      </div>
                    </div>

                    {/* Controls */}
                    <div className="flex items-center gap-3 shrink-0">
                      {hasFiles ? (
                        <>
                          {/* Source File */}
                          <div className="flex flex-col gap-1">
                            <label className="text-[9px] font-bold uppercase opacity-50">
                              {t("book.select_source_file")}
                            </label>
                            <select
                              value={state.selectedFileId}
                              onChange={(e) =>
                                updateBookState(book.id, {
                                  selectedFileId: e.target.value,
                                })
                              }
                              className="select select-bordered select-xs max-w-30"
                            >
                              {book.files?.map((f) => (
                                <option key={f.id} value={f.id}>
                                  {f.format.toUpperCase()} (
                                  {formatBytes(f.size_bytes)})
                                </option>
                              ))}
                            </select>
                          </div>

                          {/* Target Format */}
                          <div className="flex flex-col gap-1">
                            <label className="text-[9px] font-bold uppercase opacity-50">
                              {t("book.select_target_format")}
                            </label>
                            <select
                              value={state.targetFormat}
                              onChange={(e) =>
                                updateBookState(book.id, {
                                  targetFormat: e.target.value,
                                  overwriteChecked: false,
                                })
                              }
                              className="select select-bordered select-xs uppercase font-bold text-primary"
                            >
                              {CONVERT_TARGETS.map((t) => (
                                <option key={t} value={t}>
                                  {t.toUpperCase()}
                                </option>
                              ))}
                            </select>
                          </div>
                        </>
                      ) : (
                        <span className="text-xs text-error font-medium">
                          {t("book.no_files_convert", "No files to convert")}
                        </span>
                      )}
                    </div>
                  </div>

                  {/* Duplicate Alert Row */}
                  {hasDuplicate && (
                    <div className="alert bg-warning/10 border border-warning/30 text-xs p-2 rounded-lg flex items-center justify-between gap-4 mt-1">
                      <div className="flex items-center gap-1.5 text-base-content">
                        <span className="font-bold text-warning">⚠️</span>
                        <span>
                          {t(
                            "book.convert_replace_warning",
                            "This book already has a {{format}} file.",
                            { format: state.targetFormat.toUpperCase() },
                          )}
                        </span>
                      </div>
                      <label className="flex items-center gap-1.5 cursor-pointer font-semibold select-none text-base-content shrink-0">
                        <input
                          type="checkbox"
                          className="checkbox checkbox-xs checkbox-warning"
                          checked={state.overwriteChecked}
                          onChange={(e) =>
                            updateBookState(book.id, {
                              overwriteChecked: e.target.checked,
                            })
                          }
                        />
                        <span>
                          {t("book.convert_confirm_replace", "Yes, replace")}
                        </span>
                      </label>
                    </div>
                  )}
                </div>
              );
            })}
          </div>

          <div className="modal-action">
            <button
              type="button"
              className="btn btn-ghost"
              onClick={onClose}
              disabled={bulkConvertMutation.isPending}
            >
              {t("common.cancel")}
            </button>
            <button
              type="submit"
              className="btn btn-primary min-w-30"
              disabled={bulkConvertMutation.isPending || needsConfirmation}
            >
              {bulkConvertMutation.isPending ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin mr-1.5" />
                  {t("common.loading")}
                </>
              ) : (
                t("book.bulk_convert_start", "Start bulk conversion")
              )}
            </button>
          </div>
        </form>
      </div>
    </dialog>
  );
};
