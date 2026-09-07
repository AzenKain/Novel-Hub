import { DatabaseBackup, Loader2, X } from "lucide-react";
import React, { useState } from "react";
import { useTranslation } from "react-i18next";

import type { Library } from "@/types";

type CalibreImportModalProps = {
  open: boolean;
  libraries: Library[];
  importing: boolean;
  onClose: () => void;
  onImport: (path: string, library_id: string) => void;
};

export const CalibreImportModal: React.FC<CalibreImportModalProps> = ({
  open,
  libraries,
  importing,
  onClose,
  onImport,
}) => {
  const { t } = useTranslation();
  const [path, setPath] = useState("");
  const [library_id, setLibraryId] = useState("");

  const submit = (event: React.SyntheticEvent) => {
    event.preventDefault();
    if (!path.trim() || importing) return;
    onImport(path.trim(), library_id);
  };

  return (
    <dialog className={`modal ${open ? "modal-open" : ""}`}>
      <div className="modal-box max-w-md max-h-[80dvh] sm:max-h-[85vh] p-0 overflow-hidden flex flex-col">
        <div className="px-4 sm:px-6 py-3.5 sm:py-4 border-b border-base-200 flex items-center justify-between gap-3 shrink-0 bg-base-100">
          <h3 className="text-lg font-bold leading-tight">
            {t("admin.calibre_import", "Import from Calibre")}
          </h3>
          <button
            type="button"
            onClick={onClose}
            className="btn btn-ghost btn-circle btn-sm -mr-1.5 text-base-content/70 hover:text-base-content shrink-0"
            aria-label={t("common.close", "Close")}
          >
            <X className="w-4 h-4 sm:w-5 sm:h-5 stroke-[2.5]" />
          </button>
        </div>

        <form onSubmit={submit} className="p-4 sm:p-6 overflow-y-auto flex-1 min-h-0 flex flex-col gap-4">
          <div className="flex w-full flex-col gap-1.5">
            <label className="pl-1 text-sm font-medium" htmlFor="calibre-path">
              {t("admin.calibre_path", "Calibre library folder path")}
            </label>
            <input
              id="calibre-path"
              type="text"
              className="input input-bordered focus:input-primary"
              placeholder={t(
                "admin.calibre_path_placeholder",
                "/path/to/calibre-library",
              )}
              value={path}
              onChange={(event) => setPath(event.target.value)}
              disabled={importing}
            />
            <p className="pl-1 text-xs text-base-content/60">
              {t(
                "admin.calibre_path_hint",
                "Server-side path to the folder containing metadata.db. Re-importing the same library may create duplicates.",
              )}
            </p>
          </div>

          <div className="flex w-full flex-col gap-1.5">
            <label
              className="pl-1 text-sm font-medium"
              htmlFor="calibre-library"
            >
              {t("admin.calibre_target_library", "Target library (optional)")}
            </label>
            <select
              id="calibre-library"
              className="select select-bordered focus:select-primary"
              value={library_id}
              onChange={(event) => setLibraryId(event.target.value)}
              disabled={importing}
            >
              <option value="">
                {t("admin.calibre_no_library", "No library")}
              </option>
              {libraries.map((library) => (
                <option key={library.id} value={library.id}>
                  {library.name}
                </option>
              ))}
            </select>
          </div>

          <button
            type="submit"
            className="btn btn-primary mt-2 gap-2"
            disabled={!path.trim() || importing}
          >
            {importing ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <DatabaseBackup className="h-4 w-4" />
            )}
            {importing
              ? t("admin.calibre_importing", "Importing...")
              : t("admin.calibre_import", "Import from Calibre")}
          </button>
        </form>
      </div>
      <form method="dialog" className="modal-backdrop">
        <button onClick={onClose}>close</button>
      </form>
    </dialog>
  );
};
