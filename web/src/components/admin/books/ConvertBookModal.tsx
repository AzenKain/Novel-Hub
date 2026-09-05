import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";
import { FileText, X, Loader2 } from "lucide-react";

import { useConvertBookMutation } from "@/hooks";
import type { BookFile } from "@/types";

type ConvertBookModalProps = {
  open: boolean;
  bookId: string;
  files: BookFile[];
  initialFileId?: string;
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

export const ConvertBookModal: React.FC<ConvertBookModalProps> = ({
  open,
  bookId,
  files,
  initialFileId,
  onClose,
}) => {
  const { t } = useTranslation();
  const convertMutation = useConvertBookMutation();
  const [selectedFileId, setSelectedFileId] = useState(
    initialFileId || files[0]?.id || "",
  );
  const [targetFormat, setTargetFormat] = useState(CONVERT_TARGETS[0]);
  const [overwriteChecked, setOverwriteChecked] = useState(false);

  const hasDuplicateFormat = files.some(
    (file) => file.format.toLowerCase() === targetFormat.toLowerCase(),
  );

  const handleFormatChange = (format: string) => {
    setTargetFormat(format);
    setOverwriteChecked(false);
  };

  if (!open) return null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedFileId) {
      toast.error(t("book.convert_no_file", "Please select a file to convert"));
      return;
    }
    if (hasDuplicateFormat && !overwriteChecked) {
      toast.error(
        t(
          "book.convert_no_overwrite_confirm",
          "Please confirm replacing the existing file",
        ),
      );
      return;
    }

    convertMutation.mutate(
      {
        id: bookId,
        payload: { file_id: selectedFileId, target_format: targetFormat },
      },
      {
        onSuccess: (res) => {
          const jobId = res.data?.job_id;
          toast.success(
            jobId
              ? t("book.convert_queued", "Conversion queued (job {{job_id}})", {
                  job_id: jobId,
                })
              : t("book.convert_done", "Conversion finished"),
          );
          onClose();
        },
        onError: (err) => {
          toast.error(
            err instanceof Error
              ? err.message
              : t("book.convert_failed", "Conversion failed"),
          );
        },
      },
    );
  };

  return (
    <dialog className="modal modal-open">
      <div className="modal-box max-w-md">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold flex items-center gap-2">
            <FileText className="w-5 h-5 text-primary" />
            {t("book.convert_title", "Convert Book Format")}
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
          <div>
            <label className="label label-text">
              {t("book.select_source_file", "Source file")}
            </label>
            <select
              className="select select-bordered w-full font-medium"
              value={selectedFileId}
              onChange={(e) => setSelectedFileId(e.target.value)}
            >
              {files.map((file) => (
                <option key={file.id} value={file.id}>
                  {file.path.split("/").pop()} ({file.format.toUpperCase()})
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="label label-text">
              {t("book.select_target_format", "Target format")}
            </label>
            <div className="grid grid-cols-3 gap-2">
              {CONVERT_TARGETS.map((target) => {
                const isActive = targetFormat === target;
                return (
                  <button
                    key={target}
                    type="button"
                    onClick={() => handleFormatChange(target)}
                    className={`btn btn-md font-bold uppercase transition-all duration-200 ${
                      isActive
                        ? "btn-primary shadow-md border-primary"
                        : "btn-outline btn-ghost border-base-300 hover:border-primary/50"
                    }`}
                  >
                    {target}
                  </button>
                );
              })}
            </div>
          </div>

          {hasDuplicateFormat && (
            <div className="alert bg-warning/10 border border-warning/30 text-xs p-3 flex flex-col items-start gap-2 rounded-lg">
              <div className="flex items-start gap-2 text-base-content">
                <span className="font-bold text-warning">
                  ⚠️ {t("common.warning", "Warning")}
                </span>
                <span>
                  {t(
                    "book.convert_replace_warning",
                    "This book already has a {{format}} file. Proceeding will replace the existing file.",
                    { format: targetFormat.toUpperCase() },
                  )}
                </span>
              </div>
              <label className="flex items-center gap-2 cursor-pointer mt-1 font-semibold select-none text-base-content">
                <input
                  type="checkbox"
                  className="checkbox checkbox-xs checkbox-warning"
                  checked={overwriteChecked}
                  onChange={(e) => setOverwriteChecked(e.target.checked)}
                />
                <span>
                  {t(
                    "book.convert_confirm_replace",
                    "Yes, replace the existing file",
                  )}
                </span>
              </label>
            </div>
          )}

          <div className="modal-action">
            <button type="button" className="btn btn-ghost" onClick={onClose}>
              {t("common.cancel")}
            </button>
            <button
              type="submit"
              className="btn btn-primary min-w-[120px]"
              disabled={
                convertMutation.isPending ||
                !selectedFileId ||
                (hasDuplicateFormat && !overwriteChecked)
              }
            >
              {convertMutation.isPending ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin mr-2" />
                  {t("common.loading", "Loading...")}
                </>
              ) : (
                t("book.start_conversion", "Start conversion")
              )}
            </button>
          </div>
        </form>
      </div>
    </dialog>
  );
};
