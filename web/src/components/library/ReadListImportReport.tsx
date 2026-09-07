import type { CBLUnmatchedEntry } from "@/types";
import { FileWarning, X } from "lucide-react";
import React from "react";
import { useTranslation } from "react-i18next";

type ReadListImportReportProps = {
  open: boolean;
  total: number;
  matched: number;
  unmatched: CBLUnmatchedEntry[];
  onClose: () => void;
};

export const ReadListImportReport: React.FC<ReadListImportReportProps> = ({
  open,
  total,
  matched,
  unmatched,
  onClose,
}) => {
  const { t } = useTranslation();

  return (
    <dialog className={`modal ${open ? "modal-open" : ""}`}>
      <div className="modal-box max-w-lg max-h-[80dvh] sm:max-h-[85vh] p-0 overflow-hidden flex flex-col">
        <div className="px-5 py-3.5 border-b border-base-200 flex items-center justify-between gap-3 shrink-0 bg-base-100">
          <h3 className="flex items-center gap-2 text-lg font-bold leading-tight">
            <FileWarning className="h-5 w-5 text-warning shrink-0" />
            <span>{t("library.readlist_import_report", "Import report")}</span>
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

        <div className="p-5 overflow-y-auto flex-1 min-h-0 space-y-4">
          <p className="text-sm opacity-80">
            {t(
              "library.readlist_import_summary",
              "Matched {{matched}} of {{total}} entries in the file.",
              {
                matched,
                total,
              },
            )}
          </p>

          {unmatched.length > 0 && (
            <div className="rounded-xl border border-base-200 overflow-x-auto">
              <table className="table table-sm">
                <thead>
                  <tr>
                    <th>{t("library.readlist_import_series", "Series")}</th>
                    <th>{t("library.readlist_import_number", "Number")}</th>
                  </tr>
                </thead>
                <tbody>
                  {unmatched.map((entry, index) => (
                    <tr key={`${entry.series}-${entry.number}-${index}`}>
                      <td className="max-w-[16rem] truncate">{entry.series}</td>
                      <td className="font-mono text-xs">{entry.number}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          <div className="modal-action border-t border-base-200 pt-4 mt-6">
            <button className="btn btn-primary" onClick={onClose}>
              {t("common.close", "Close")}
            </button>
          </div>
        </div>
      </div>
      <form method="dialog" className="modal-backdrop">
        <button onClick={onClose}>close</button>
      </form>
    </dialog>
  );
};
