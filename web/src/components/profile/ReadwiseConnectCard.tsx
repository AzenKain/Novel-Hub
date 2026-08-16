import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { BookOpenCheck, Key, Check, Unlink } from "lucide-react";
import { toast } from "react-toastify";
import { useConnectTrackerMutation, useTrackerConnectionsQuery } from "@/hooks";
import { useAuthStore } from "@/stores";
import { hasPermission } from "@/utils/permission";

export const ReadwiseConnectCard: React.FC = () => {
  const { t } = useTranslation();
  const user = useAuthStore((state) => state.user);
  const [token, setToken] = useState("");
  const [showForm, setShowForm] = useState(false);

  const canUse = hasPermission(user, "tracker.sync");
  const { data: connections = [], isLoading } = useTrackerConnectionsQuery(!!user && canUse);
  const connected = connections.some((c) => c.provider === "readwise" && c.connected);

  const connectMutation = useConnectTrackerMutation();

  if (!canUse) return null;

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
          setShowForm(false);
          setToken("");
          toast.success(t("highlights_export.connected", "Readwise connected"));
        },
        onError: (err) =>
          toast.error(
            err instanceof Error && err.message
              ? err.message
              : t("highlights_export.connect_failed", "Failed to connect Readwise")
          ),
      }
    );
  };

  return (
    <div className="rounded-2xl border border-base-300 bg-base-100 p-6 shadow-sm space-y-4">
      <div className="flex items-center justify-between border-b border-base-200 pb-3">
        <div className="flex items-center gap-3">
          <div className="grid h-10 w-10 place-items-center rounded-xl bg-secondary/10 text-secondary">
            {connected ? <Check className="h-5 w-5" /> : <BookOpenCheck className="h-5 w-5" />}
          </div>
          <div>
            <h3 className="text-base font-bold flex items-center gap-2">
              {t("highlights_export.profile_title", "Readwise Highlights")}
              {connected && (
                <span className="badge badge-success badge-sm">{t("common.connected", "Connected")}</span>
              )}
            </h3>
            <p className="text-xs text-base-content/60">
              {t(
                "highlights_export.profile_subtitle",
                "Export your reading highlights to Readwise. The token is stored encrypted."
              )}
            </p>
          </div>
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center gap-2 text-sm text-base-content/60">
          <span className="loading loading-spinner loading-xs" />
          {t("common.loading", "Loading...")}
        </div>
      ) : connected && !showForm ? (
        <div className="flex items-center justify-between rounded-lg bg-success/10 p-3 text-sm">
          <span>{t("highlights_export.connected", "Readwise connected")}</span>
          <button
            type="button"
            className="btn btn-ghost btn-xs gap-1"
            onClick={() => setShowForm(true)}
          >
            <Unlink className="h-3.5 w-3.5" />
            {t("trackers.connect_new_token", "Use new token")}
          </button>
        </div>
      ) : (
        <form onSubmit={handleConnect} className="space-y-3">
          <div>
            <label className="block text-xs font-bold uppercase tracking-wider text-base-content/70 mb-1">
              {t("highlights_export.token_label", "Readwise API token")}
            </label>
            <div className="relative">
              <input
                id="readwise-profile-token"
                type="password"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder={t("highlights_export.token_placeholder", "Paste your Readwise token")}
                className="input input-bordered w-full font-mono text-xs pr-10"
                required
              />
              <Key className="absolute right-3 top-3 h-4 w-4 text-base-content/40" />
            </div>
          </div>
          <div className="flex items-center justify-between pt-2">
            <a
              href="https://readwise.io/access_token"
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-primary hover:underline flex items-center gap-1"
            >
              {t("highlights_export.get_token_link", "Get Readwise Access Token ↗")}
            </a>
            <button
              type="submit"
              className="btn btn-primary btn-sm gap-2"
              disabled={connectMutation.isPending}
            >
              {connectMutation.isPending ? (
                <span className="loading loading-spinner loading-xs" />
              ) : (
                <Check className="h-4 w-4" />
              )}
              {t("highlights_export.connect_readwise_btn", "Connect Readwise")}
            </button>
          </div>
        </form>
      )}
    </div>
  );
};
