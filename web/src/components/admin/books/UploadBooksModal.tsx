import { Loader2, Upload } from "lucide-react";
import React from "react";

import type { Library } from "@/types";

type UploadBooksModalProps = {
  open: boolean;
  libraries: Library[];
  uploadLibraryId: string;
  uploading: boolean;
  accept: string;
  onClose: () => void;
  onLibraryChange: (libraryId: string) => void;
  onUploadFiles: (event: React.ChangeEvent<HTMLInputElement>) => void;
};

export const UploadBooksModal: React.FC<UploadBooksModalProps> = ({
  open,
  libraries,
  uploadLibraryId,
  uploading,
  accept,
  onClose,
  onLibraryChange,
  onUploadFiles,
}) => (
  <dialog className={`modal ${open ? "modal-open" : ""}`}>
    <div className="modal-box">
      <button
        onClick={onClose}
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
            className="select select-bordered focus:select-primary"
            value={uploadLibraryId}
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
          <label
            className={`flex flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed p-8 transition-colors ${uploadLibraryId ? "cursor-pointer border-primary/50 hover:bg-primary/5" : "cursor-not-allowed border-base-300 opacity-50"}`}
          >
            <input
              type="file"
              multiple
              accept={accept}
              onChange={onUploadFiles}
              className="hidden"
              disabled={!uploadLibraryId || uploading}
            />
            <Upload
              size={32}
              className={uploadLibraryId ? "text-primary" : "text-base-content/50"}
            />
            <span className="font-medium">
              {uploading ? "Uploading..." : "Click to select ebook files"}
            </span>
            {uploading && (
              <Loader2 className="mt-2 h-5 w-5 animate-spin text-primary" />
            )}
          </label>
        </div>
      </div>
    </div>
    <form method="dialog" className="modal-backdrop">
      <button onClick={onClose}>close</button>
    </form>
  </dialog>
);
