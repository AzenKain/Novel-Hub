import type { CBLUnmatchedEntry } from "@/types";
import { FileWarning } from "lucide-react";
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
      <div className="modal-box max-w-lg">
        <h3 className="flex items-center gap-2 text-lg font-bold">
          <FileWarning className="h-5 w-5 text-warning" />
          {t("library.readlist_import_report", "Import report")}
        </h3>

        <p className="py-3 text-sm opacity-80">
          {t("library.readlist_import_summary", "Matched {{matched}} of {{total}} entries in the file.", {
            matched,
            total,
          })}
        </p>

        {unmatched.length > 0 && (
          <div className="max-h-64 overflow-y-auto rounded-xl border border-base-200">
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

        <div className="modal-action">
          <button className="btn btn-primary" onClick={onClose}>
            {t("common.close", "Close")}
          </button>
        </div>
      </div>
      <form method="dialog" className="modal-backdrop">
        <button onClick={onClose}>close</button>
      </form>
    </dialog>
  );
};
