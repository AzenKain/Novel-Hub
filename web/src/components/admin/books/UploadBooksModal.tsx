import { Loader2, Upload, Zap } from "lucide-react";
import React, { useState } from "react";

import type { Library } from "@/types";

type UploadBooksModalProps = {
  open: boolean;
  libraries: Library[];
  uploadLibraryId: string;
  uploading: boolean;
  uploadProgress?: number;
  uploadSpeed?: string;
  uploadCurrentFile?: string;
  uploadBytesText?: string;
  uploadBatchInfo?: { current: number; total: number } | null;
  accept: string;
  onClose: () => void;
  onLibraryChange: (libraryId: string) => void;
  onUploadFiles: (filesOrEvent: FileList | File[] | React.ChangeEvent<HTMLInputElement>) => void;
};

export const UploadBooksModal: React.FC<UploadBooksModalProps> = ({
  open,
  libraries,
  uploadLibraryId,
  uploading,
  uploadProgress = 0,
  uploadSpeed = "0 B/s",
  uploadCurrentFile = "",
  uploadBytesText = "",
  uploadBatchInfo = null,
  accept,
  onClose,
  onLibraryChange,
  onUploadFiles,
}) => {
  const [isDragging, setIsDragging] = useState(false);

  return (
    <dialog className={`modal ${open ? "modal-open" : ""}`}>
      <div className="modal-box max-w-lg">
        <button
          onClick={onClose}
          disabled={uploading}
          className="btn btn-ghost btn-circle btn-sm absolute right-2 top-2"
        >
          ✕
        </button>
        <h3 className="mb-4 border-b border-base-200 pb-4 text-lg font-bold">
          Upload Books
        </h3>
        <div className="flex flex-col gap-4">
          <div className="flex w-full flex-col gap-1.5">
            <label className="pl-1 text-sm font-medium">Target Library</label>
            <select
              className="select select-bordered focus:select-primary w-full"
              value={uploadLibraryId}
              disabled={uploading}
              onChange={(event) => onLibraryChange(event.target.value)}
            >
              <option value="" disabled>
                Select a library...
              </option>
              {libraries.map((library) => (
                <option key={library.id} value={library.id}>
                  {library.name}
                </option>
              ))}
            </select>
          </div>

          <div className="mt-2 flex w-full flex-col gap-1.5">
            <label className="pl-1 text-sm font-medium">Select ebook files</label>

            {uploading ? (
              <div className="flex flex-col gap-3 rounded-2xl border border-primary/30 bg-primary/5 p-5 transition-all">
                <div className="flex items-center justify-between min-w-0">
                  <div className="flex items-center gap-2.5 min-w-0">
                    <Loader2 className="h-5 w-5 shrink-0 animate-spin text-primary" />
                    <span className="font-semibold text-xs sm:text-sm truncate min-w-0" title={uploadCurrentFile}>
                      {uploadCurrentFile || "Uploading..."}
                    </span>
                  </div>
                  {uploadBatchInfo && uploadBatchInfo.total > 1 && (
                    <span className="shrink-0 rounded-full bg-primary/20 px-2.5 py-0.5 font-mono text-[11px] font-bold text-primary">
                      {uploadBatchInfo.current} / {uploadBatchInfo.total}
                    </span>
                  )}
                </div>

                <div className="w-full">
                  <div className="mb-1.5 flex items-center justify-between text-xs">
                    <span className="font-mono text-primary font-bold text-sm">{uploadProgress}%</span>
                    <div className="flex items-center gap-2">
                      {uploadSpeed && (
                        <span className="flex items-center gap-1 font-mono text-xs font-semibold text-emerald-600 dark:text-emerald-400 bg-emerald-500/15 px-2 py-0.5 rounded-full">
                          <Zap className="h-3 w-3" />
                          {uploadSpeed}
                        </span>
                      )}
                      {uploadBytesText && (
                        <span className="font-mono opacity-60 text-xs">
                          {uploadBytesText}
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="h-2.5 w-full overflow-hidden rounded-full bg-base-300">
                    <div
                      className="h-full bg-primary transition-all duration-200 ease-out"
                      style={{ width: `${Math.max(0, Math.min(100, uploadProgress))}%` }}
                    />
                  </div>
                </div>
              </div>
            ) : (
              <div
                onDragOver={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  if (uploadLibraryId) setIsDragging(true);
                }}
                onDragLeave={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  setIsDragging(false);
                }}
                onDrop={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  setIsDragging(false);
                  if (uploadLibraryId && e.dataTransfer.files?.length > 0) {
                    onUploadFiles(e.dataTransfer.files);
                  }
                }}
                className={`flex flex-col items-center justify-center gap-3 rounded-2xl border-2 border-dashed p-8 transition-all ${
                  !uploadLibraryId
                    ? "cursor-not-allowed border-base-300 opacity-50"
                    : isDragging
                      ? "border-primary bg-primary/10 scale-[1.01]"
                      : "cursor-pointer border-primary/50 hover:bg-primary/5 hover:border-primary"
                }`}
              >
                <label className="flex flex-col items-center justify-center gap-3 w-full cursor-pointer">
                  <input
                    type="file"
                    multiple
                    accept={accept}
                    onChange={onUploadFiles}
                    className="hidden"
                    disabled={!uploadLibraryId || uploading}
                  />
                  <div className={`p-3 rounded-2xl transition-colors ${isDragging ? "bg-primary text-primary-content" : "bg-primary/10 text-primary"}`}>
                    <Upload size={28} />
                  </div>
                  <div className="text-center">
                    <span className="font-semibold text-sm block">
                      {isDragging
                        ? "Drop ebook files here to upload"
                        : "Click or drag & drop ebook files here"}
                    </span>
                    <span className="text-xs opacity-50 block mt-1">
                      Supports EPUB, MOBI, PDF, CBZ, FB2 files
                    </span>
                  </div>
                </label>
              </div>
            )}
          </div>
        </div>
      </div>
      <form method="dialog" className="modal-backdrop">
        <button onClick={onClose} disabled={uploading}>close</button>
      </form>
    </dialog>
  );
};

