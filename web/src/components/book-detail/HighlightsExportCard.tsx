import { Download, Share2, Check, Layers, FileSpreadsheet } from "lucide-react";
import React, { useState } from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";

import { highlightService } from "@/services";
import { useExportHighlightsToReadwiseMutation, useTrackerConnectionsQuery } from "@/hooks";
import { useAuthStore } from "@/stores";
import { hasPermission } from "@/utils/permission";

type HighlightsExportCardProps = {
  book_id: string;
};

export const HighlightsExportCard: React.FC<HighlightsExportCardProps> = ({ book_id }) => {
  const { t } = useTranslation();
  const user = useAuthStore((state) => state.user);
  const [downloading, setDownloading] = useState<string | null>(null);

  const canHighlight = hasPermission(user, "book.highlight");
  const { data: connections = [] } = useTrackerConnectionsQuery(!!user && canHighlight);
  const readwiseConnected = connections.some((c) => c.provider === "readwise" && c.connected);

  const exportMutation = useExportHighlightsToReadwiseMutation();

  if (!canHighlight) return null;

  const handleExport = () => {
    exportMutation.mutate(book_id, {
      onSuccess: (data) =>
        toast.success(
          t("highlights_export.exported", "Exported {{count}} highlights to Readwise", {
            count: data.exported,
          })
        ),
      onError: (err) =>
        toast.error(err.message || t("highlights_export.export_failed", "Failed to export highlights")),
    });
  };

  const handleDownloadBlob = async (type: "md" | "anki" | "csv") => {
    setDownloading(type);
    try {
      let blob: Blob;
      let filename: string;
      if (type === "anki") {
        blob = await highlightService.exportAnki(book_id);
        filename = "highlights.apkg";
      } else if (type === "csv") {
        blob = await highlightService.exportCSV(book_id);
        filename = "highlights.csv";
      } else {
        blob = await highlightService.exportMarkdown(book_id);
        filename = "highlights.md";
      }

      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch (err) {
      const message = await highlightService.extractErrorMessage(
        err,
        t("highlights_export.export_failed", "Failed to export highlights")
      );
      toast.error(message || t("highlights_export.no_highlights", "This book has no highlights to export"));
    } finally {
      setDownloading(null);
    }
  };

  return (
    <div className="space-y-3">
      <h3 className="text-xl font-bold flex items-center gap-2">
        <Share2 className="h-5 w-5" />
        {t("highlights_export.title", "Export Highlights")}
      </h3>
      <p className="text-xs text-base-content/60">
        {t(
          "highlights_export.subtitle",
          "Send highlights to Readwise, export to Anki flashcards, or download as Markdown/CSV."
        )}
      </p>

      {/* Readwise connection is managed in Profile; this card only reflects its state. */}
      {readwiseConnected ? (
        <div className="flex items-center gap-2 rounded-lg bg-success/10 px-3 py-2 text-xs text-base-content/70">
          <Check className="h-3.5 w-3.5 text-success" />
          <span>{t("highlights_export.connected", "Readwise connected")}</span>
          <Link
            to="/profile?tab=trackers"
            className="ml-auto link link-hover text-primary"
          >
            {t("highlights_export.manage_in_profile", "Manage in Profile")}
          </Link>
        </div>
      ) : (
        <div className="flex items-center gap-2 rounded-lg bg-base-200/60 px-3 py-2.5 text-xs text-base-content/70">
          <span>{t("highlights_export.connect_in_profile_hint", "Connect your Readwise account to export highlights.")}</span>
          <Link
            to="/profile?tab=trackers"
            className="ml-auto shrink-0 link link-hover text-primary"
          >
            {t("highlights_export.manage_in_profile", "Manage in Profile")}
          </Link>
        </div>
      )}

      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          onClick={handleExport}
          className="btn btn-primary btn-sm gap-2"
          disabled={!readwiseConnected || exportMutation.isPending}
          title={readwiseConnected ? undefined : t("highlights_export.connect_in_profile_hint", "Connect your Readwise account to export highlights.")}
        >
          {exportMutation.isPending ? (
            <span className="loading loading-spinner loading-xs" />
          ) : (
            <Share2 className="h-4 w-4" />
          )}
          {t("highlights_export.export_btn", "Export to Readwise")}
        </button>
        <button
          type="button"
          onClick={() => handleDownloadBlob("anki")}
          className="btn btn-outline btn-sm gap-2"
          disabled={downloading !== null}
        >
          {downloading === "anki" ? <span className="loading loading-spinner loading-xs" /> : <Layers className="h-4 w-4 text-primary" />}
          {t("highlights_export.anki_btn", "Anki Deck (.apkg)")}
        </button>
        <button
          type="button"
          onClick={() => handleDownloadBlob("md")}
          className="btn btn-outline btn-sm gap-2"
          disabled={downloading !== null}
        >
          {downloading === "md" ? <span className="loading loading-spinner loading-xs" /> : <Download className="h-4 w-4" />}
          {t("highlights_export.markdown_btn", "Markdown (.md)")}
        </button>
        <button
          type="button"
          onClick={() => handleDownloadBlob("csv")}
          className="btn btn-outline btn-sm gap-2"
          disabled={downloading !== null}
        >
          {downloading === "csv" ? <span className="loading loading-spinner loading-xs" /> : <FileSpreadsheet className="h-4 w-4" />}
          {t("highlights_export.csv_btn", "CSV (.csv)")}
        </button>
      </div>
    </div>
  );
};
