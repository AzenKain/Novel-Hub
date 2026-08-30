import React, { useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import {
  Stethoscope,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  Wrench,
  RefreshCw,
  FileCode,
  ShieldCheck,
  ChevronDown,
  ChevronUp,
} from "lucide-react";
import { toast } from "react-toastify";

import { useBookValidationQuery, useRepairBookMutation } from "@/hooks";
import type { RepairOptions } from "@/types";

interface BookDoctorModalProps {
  bookId: string;
  bookTitle?: string;
  isOpen: boolean;
  onClose: () => void;
}

export const BookDoctorModal: React.FC<BookDoctorModalProps> = ({
  bookId,
  bookTitle,
  isOpen,
  onClose,
}) => {
  const { t } = useTranslation();
  const [showLogs, setShowLogs] = useState(false);
  const [lastLogs, setLastLogs] = useState<string[]>([]);
  const [options, setOptions] = useState<RepairOptions>({
    normalize_mimetype: true,
    fix_container: true,
    fix_xhtml: true,
    reconcile_manifest: true,
    reconcile_spine: true,
    fix_toc: true,
    clean_broken_links: true,
    fix_metadata: true,
  });

  const {
    data: report,
    isLoading,
    isRefetching,
    refetch,
  } = useBookValidationQuery(bookId, undefined, isOpen);

  const repairMutation = useRepairBookMutation();

  if (!isOpen) return null;

  const handleRepair = () => {
    repairMutation.mutate(
      { bookId, options },
      {
        onSuccess: (res) => {
          setLastLogs(res.logs || []);
          setShowLogs(true);
          toast.success(
            t(
              "doctor.repair_success",
              "Repaired {{count}} structural issues successfully!",
              { count: res.fixed_count }
            )
          );
          refetch();
        },
        onError: (err: any) => {
          toast.error(
            err?.message ||
              t("doctor.repair_failed", "Failed to repair EPUB structure")
          );
        },
      }
    );
  };

  const toggleOption = (key: keyof RepairOptions) => {
    setOptions((prev) => ({ ...prev, [key]: !prev[key] }));
  };

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button
        type="button"
        className="absolute inset-0 bg-black/60 backdrop-blur-sm cursor-default"
        aria-label={t("common.close", "Close")}
        onClick={onClose}
      />
      <div className="relative z-10 w-full max-w-2xl max-h-[90vh] flex flex-col rounded-2xl border border-base-300 bg-base-100 p-6 shadow-2xl space-y-4">
        {/* Header */}
        <div className="flex items-center justify-between pb-4 border-b border-base-200">
          <div className="flex items-center gap-3">
            <div className="p-2.5 rounded-xl bg-primary/10 text-primary">
              <Stethoscope className="h-6 w-6" />
            </div>
            <div>
              <h3 className="font-bold text-lg flex items-center gap-2">
                {t("doctor.modal_title", "Book Doctor")}
                <span className="badge badge-sm badge-outline">EPUB Repair</span>
              </h3>
              <p className="text-xs text-base-content/60 truncate max-w-md">
                {bookTitle || bookId}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => refetch()}
              className="btn btn-ghost btn-circle btn-sm"
              disabled={isLoading || isRefetching}
              title={t("common.refresh", "Refresh")}
            >
              <RefreshCw
                className={`h-4 w-4 ${isLoading || isRefetching ? "animate-spin" : ""}`}
              />
            </button>
            <button
              type="button"
              onClick={onClose}
              className="btn btn-ghost btn-circle btn-sm"
            >
              ✕
            </button>
          </div>
        </div>

        {/* Content Body */}
        <div className="flex-1 overflow-y-auto py-4 space-y-4">
          {isLoading ? (
            <div className="flex flex-col items-center justify-center py-12 space-y-3">
              <span className="loading loading-spinner loading-lg text-primary" />
              <p className="text-sm text-base-content/70">
                {t("doctor.diagnosing", "Diagnosing EPUB structure and syntax...")}
              </p>
            </div>
          ) : report ? (
            <>
              {/* Status Banner */}
              {report.valid && report.issues.length === 0 ? (
                <div className="alert alert-success/15 border border-success/30 rounded-xl">
                  <CheckCircle2 className="h-5 w-5 text-success shrink-0" />
                  <div>
                    <h4 className="font-bold text-sm text-success">
                      {t("doctor.status_healthy", "EPUB is Healthy & Valid")}
                    </h4>
                    <p className="text-xs text-base-content/70">
                      {t(
                        "doctor.status_healthy_desc",
                        "No XML syntax errors, broken links, or missing manifest declarations found."
                      )}
                    </p>
                  </div>
                </div>
              ) : (
                <div className="alert alert-warning/15 border border-warning/30 rounded-xl">
                  <AlertTriangle className="h-5 w-5 text-warning shrink-0" />
                  <div>
                    <h4 className="font-bold text-sm text-warning">
                      {t("doctor.status_issues_found", "Structural Issues Detected")}
                    </h4>
                    <p className="text-xs text-base-content/70">
                      {t(
                        "doctor.status_issues_desc",
                        "Found {{errors}} errors and {{warnings}} warnings that may cause reader crashes or missing chapters.",
                        { errors: report.errors, warnings: report.warnings }
                      )}
                    </p>
                  </div>
                </div>
              )}

              {/* Counters */}
              <div className="grid grid-cols-3 gap-2">
                <div className="bg-base-200/50 rounded-xl p-3 text-center">
                  <span className="text-xs text-base-content/60 block">
                    {t("doctor.errors", "Errors")}
                  </span>
                  <span className={`text-xl font-bold ${report.errors > 0 ? "text-error" : "text-base-content"}`}>
                    {report.errors}
                  </span>
                </div>
                <div className="bg-base-200/50 rounded-xl p-3 text-center">
                  <span className="text-xs text-base-content/60 block">
                    {t("doctor.warnings", "Warnings")}
                  </span>
                  <span className={`text-xl font-bold ${report.warnings > 0 ? "text-warning" : "text-base-content"}`}>
                    {report.warnings}
                  </span>
                </div>
                <div className="bg-base-200/50 rounded-xl p-3 text-center">
                  <span className="text-xs text-base-content/60 block">
                    {t("doctor.info", "Info")}
                  </span>
                  <span className="text-xl font-bold text-base-content">
                    {report.infos}
                  </span>
                </div>
              </div>

              {/* Issues List */}
              {report.issues.length > 0 && (
                <div className="space-y-2">
                  <h4 className="text-xs font-semibold uppercase tracking-wider text-base-content/60">
                    {t("doctor.detected_issues", "Detected Issues ({{count}})", {
                      count: report.issues.length,
                    })}
                  </h4>
                  <div className="space-y-2 max-h-56 overflow-y-auto pr-1">
                    {report.issues.map((issue, idx) => (
                      <div
                        key={idx}
                        className="flex items-start gap-3 p-3 rounded-lg bg-base-200/60 text-xs"
                      >
                        {issue.severity === "error" ? (
                          <XCircle className="h-4 w-4 text-error shrink-0 mt-0.5" />
                        ) : issue.severity === "warning" ? (
                          <AlertTriangle className="h-4 w-4 text-warning shrink-0 mt-0.5" />
                        ) : (
                          <FileCode className="h-4 w-4 text-info shrink-0 mt-0.5" />
                        )}
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-0.5">
                            <span className="font-mono font-semibold text-[11px]">
                              {issue.code}
                            </span>
                            {issue.file && (
                              <span className="text-base-content/50 truncate text-[10px]">
                                ({issue.file})
                              </span>
                            )}
                            {issue.fixable && (
                              <span className="badge badge-success badge-xs ml-auto">
                                {t("doctor.auto_fixable", "Auto-Fixable")}
                              </span>
                            )}
                          </div>
                          <p className="text-base-content/80">{issue.message}</p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Repair Options Accordion */}
              <div className="space-y-2">
                <h4 className="text-xs font-semibold uppercase tracking-wider text-base-content/60">
                  {t("doctor.repair_options_title", "Repair Options")}
                </h4>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-xs">
                  <label className="flex items-center gap-2 p-2.5 rounded-lg bg-base-200/40 cursor-pointer hover:bg-base-200/70">
                    <input
                      type="checkbox"
                      className="checkbox checkbox-primary checkbox-xs"
                      checked={!!options.fix_xhtml}
                      onChange={() => toggleOption("fix_xhtml")}
                    />
                    <span>{t("doctor.opt_xhtml", "Fix XHTML & XML Entities")}</span>
                  </label>
                  <label className="flex items-center gap-2 p-2.5 rounded-lg bg-base-200/40 cursor-pointer hover:bg-base-200/70">
                    <input
                      type="checkbox"
                      className="checkbox checkbox-primary checkbox-xs"
                      checked={!!options.normalize_mimetype}
                      onChange={() => toggleOption("normalize_mimetype")}
                    />
                    <span>{t("doctor.opt_mimetype", "Normalize Mimetype Entry")}</span>
                  </label>
                  <label className="flex items-center gap-2 p-2.5 rounded-lg bg-base-200/40 cursor-pointer hover:bg-base-200/70">
                    <input
                      type="checkbox"
                      className="checkbox checkbox-primary checkbox-xs"
                      checked={!!options.reconcile_manifest}
                      onChange={() => toggleOption("reconcile_manifest")}
                    />
                    <span>{t("doctor.opt_manifest", "Reconcile Manifest & Spine")}</span>
                  </label>
                  <label className="flex items-center gap-2 p-2.5 rounded-lg bg-base-200/40 cursor-pointer hover:bg-base-200/70">
                    <input
                      type="checkbox"
                      className="checkbox checkbox-primary checkbox-xs"
                      checked={!!options.fix_toc}
                      onChange={() => toggleOption("fix_toc")}
                    />
                    <span>{t("doctor.opt_toc", "Fix NCX & Nav Navigation")}</span>
                  </label>
                  <label className="flex items-center gap-2 p-2.5 rounded-lg bg-base-200/40 cursor-pointer hover:bg-base-200/70">
                    <input
                      type="checkbox"
                      className="checkbox checkbox-primary checkbox-xs"
                      checked={!!options.clean_broken_links}
                      onChange={() => toggleOption("clean_broken_links")}
                    />
                    <span>{t("doctor.opt_links", "Clean Broken Links")}</span>
                  </label>
                  <label className="flex items-center gap-2 p-2.5 rounded-lg bg-base-200/40 cursor-pointer hover:bg-base-200/70">
                    <input
                      type="checkbox"
                      className="checkbox checkbox-primary checkbox-xs"
                      checked={!!options.fix_metadata}
                      onChange={() => toggleOption("fix_metadata")}
                    />
                    <span>{t("doctor.opt_metadata", "Ensure EPUB3 Metadata")}</span>
                  </label>
                </div>
              </div>

              {/* Repair Log Collapse */}
              {lastLogs.length > 0 && (
                <div className="border border-base-200 rounded-xl overflow-hidden">
                  <button
                    type="button"
                    onClick={() => setShowLogs(!showLogs)}
                    className="w-full flex items-center justify-between p-3 bg-base-200/40 text-xs font-semibold"
                  >
                    <span className="flex items-center gap-2">
                      <ShieldCheck className="h-4 w-4 text-success" />
                      {t("doctor.last_repair_logs", "Last Repair Logs ({{count}} fixes)", {
                        count: lastLogs.length,
                      })}
                    </span>
                    {showLogs ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                  </button>
                  {showLogs && (
                    <div className="p-3 bg-base-300/30 max-h-40 overflow-y-auto space-y-1 font-mono text-[11px] text-base-content/80">
                      {lastLogs.map((log, i) => (
                        <div key={i} className="leading-tight">
                          {log}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </>
          ) : null}
        </div>

        {/* Footer Actions */}
        <div className="modal-action border-t border-base-200 pt-4 flex items-center justify-between">
          <button
            type="button"
            onClick={onClose}
            className="btn btn-ghost btn-sm"
          >
            {t("common.close", "Close")}
          </button>
          <button
            type="button"
            onClick={handleRepair}
            className="btn btn-primary btn-sm gap-2"
            disabled={repairMutation.isPending || isLoading}
          >
            {repairMutation.isPending ? (
              <span className="loading loading-spinner loading-xs" />
            ) : (
              <Wrench className="h-4 w-4" />
            )}
            {t("doctor.start_repair_btn", "Auto-Repair EPUB")}
          </button>
        </div>
      </div>
    </div>,
    document.body
  );
};
