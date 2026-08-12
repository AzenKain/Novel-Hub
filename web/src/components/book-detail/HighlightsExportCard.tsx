import { Download, Share2 } from "lucide-react";
import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";

import { API_BASE } from "@/config/api";
import { useConnectTrackerMutation, useExportHighlightsToReadwiseMutation } from "@/hooks";
import { useAuthStore } from "@/stores";
import { hasPermission } from "@/utils/permission";

type HighlightsExportCardProps = {
  book_id: string;
};

export const HighlightsExportCard: React.FC<HighlightsExportCardProps> = ({ book_id }) => {
  const { t } = useTranslation();
  const user = useAuthStore((state) => state.user);
  const [token, setToken] = useState("");

  const connectMutation = useConnectTrackerMutation();
  const exportMutation = useExportHighlightsToReadwiseMutation();

  if (!hasPermission(user, "book.highlight")) return null;

  const handleConnect = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = token.trim();
    if (!trimmed) {
      toast.error(t("highlights_export.enter_token", "Enter your Readwise API token"));
      return;
    }
    connectMutation.mutate(
      { provider: "readwise", access_token: trimmed },
      {
        onSuccess: () => {
          toast.success(t("highlights_export.connected", "Readwise connected"));
          setToken("");
        },
        onError: (err) =>
          toast.error(err.message || t("highlights_export.connect_failed", "Failed to connect Readwise")),
      }
    );
  };

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

  const handleMarkdown = () => {
    window.open(`${API_BASE}/highlights/${encodeURIComponent(book_id)}/export.md`, "_blank");
  };

  return (
    <div className="space-y-3">
      <h3 className="flex items-center gap-2 text-xl font-bold">
        <Share2 className="h-5 w-5" />
        {t("highlights_export.title", "Export Highlights")}
      </h3>
      <p className="text-xs text-base-content/60">
        {t(
          "highlights_export.subtitle",
          "Send this book's highlights to Readwise, or download them as Markdown for Obsidian / Logseq."
        )}
      </p>

      <form onSubmit={handleConnect} className="flex flex-col gap-2 sm:flex-row sm:items-end">
        <div className="flex flex-1 min-w-0 flex-col gap-1.5">
          <label className="pl-1 text-xs font-medium" htmlFor="readwise-token">
            {t("highlights_export.token_label", "Readwise API token")}
          </label>
          <input
            id="readwise-token"
            type="password"
            className="input input-bordered input-sm w-full focus:input-primary"
            placeholder={t("highlights_export.token_placeholder", "Paste your Readwise token")}
            value={token}
            onChange={(e) => setToken(e.target.value)}
          />
        </div>
        <button
          type="submit"
          className="btn btn-outline btn-sm shrink-0 gap-2"
          disabled={!token.trim() || connectMutation.isPending}
        >
          {connectMutation.isPending ? (
            <span className="loading loading-spinner loading-xs" />
          ) : (
            <Share2 className="h-4 w-4" />
          )}
          {t("highlights_export.connect_btn", "Connect")}
        </button>
      </form>

      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          onClick={handleExport}
          className="btn btn-primary btn-sm gap-2"
          disabled={exportMutation.isPending}
        >
          {exportMutation.isPending ? (
            <span className="loading loading-spinner loading-xs" />
          ) : (
            <Share2 className="h-4 w-4" />
          )}
          {t("highlights_export.export_btn", "Export to Readwise")}
        </button>
        <button type="button" onClick={handleMarkdown} className="btn btn-outline btn-sm gap-2">
          <Download className="h-4 w-4" />
          {t("highlights_export.markdown_btn", "Download .md")}
        </button>
      </div>
    </div>
  );
};