import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link2, Key, Check, Unlink } from "lucide-react";
import { useConnectTrackerMutation, useTrackerConnectionsQuery } from "@/hooks";
import { usePublicSettings } from "@/hooks/useSettings";
import { toast } from "react-toastify";
import { useAuthStore } from "@/stores";
import { hasPermission } from "@/utils/permission";

export const TrackerConnectCard: React.FC = () => {
  const { t } = useTranslation();
  const user = useAuthStore((state) => state.user);
  const settings = usePublicSettings();
  const [accessToken, setAccessToken] = useState("");

  const canSync = hasPermission(user, "tracker.sync");
  const { data: connections = [] } = useTrackerConnectionsQuery(!!user && canSync);
  const serverConnected = connections.some((c) => c.provider === "anilist" && c.connected);
  const [showForm, setShowForm] = useState(false);
  const connected = serverConnected && !showForm;

  const connectMutation = useConnectTrackerMutation();

  if (!settings?.enable_anilist_tracking) return null;
  if (!canSync) return null;

  const handleConnect = (e: React.FormEvent) => {
    e.preventDefault();
    if (!accessToken.trim()) {
      toast.error(t("trackers.enter_token", "Please paste your AniList Access Token"));
      return;
    }

    connectMutation.mutate(
      { provider: "anilist", access_token: accessToken.trim() },
      {
        onSuccess: () => {
          setShowForm(false);
          setAccessToken("");
          toast.success(t("trackers.connect_success", "AniList account connected successfully!"));
        },
        onError: (err: any) => {
          toast.error(err?.message || t("trackers.connect_failed", "Failed to connect AniList account"));
        },
      }
    );
  };

  return (
    <div className="rounded-2xl border border-base-300 bg-base-100 p-6 shadow-sm space-y-4">
      <div className="flex items-start gap-3 border-b border-base-200 pb-3">
        <div className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-info/10 text-info mt-0.5">
          {connected ? <Check className="h-5 w-5" /> : <Link2 className="h-5 w-5" />}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2 mb-1">
            <h3 className="text-base font-bold leading-tight text-base-content">
              {t("trackers.anilist_title", "AniList Reading Tracker")}
            </h3>
            {connected && (
              <span className="badge badge-success badge-sm shrink-0 font-medium whitespace-nowrap">
                {t("common.connected", "Connected")}
              </span>
            )}
          </div>
          <p className="text-xs text-base-content/60 leading-relaxed">
            {t("trackers.anilist_subtitle", "Sync manga and light novel reading progress directly to AniList.")}
          </p>
        </div>
      </div>

      {connected ? (
        <div className="flex items-center justify-between rounded-lg bg-success/10 p-3 text-sm">
          <span>{t("trackers.connect_success", "AniList account connected successfully!")}</span>
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
              {t("trackers.token_label", "AniList OAuth Access Token")}
            </label>
            <div className="relative">
              <input
                type="password"
                value={accessToken}
                onChange={(e) => setAccessToken(e.target.value)}
                placeholder="Paste your token (eyJ0eXAi...)"
                className="input input-bordered w-full font-mono text-xs pr-10"
                required
              />
              <Key className="absolute right-3 top-3 h-4 w-4 text-base-content/40" />
            </div>
          </div>

          <div className="flex items-center justify-between pt-2">
            <a
              href="https://anilist.co/api/v2/oauth/authorize?client_id=19807&response_type=token"
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-primary hover:underline flex items-center gap-1"
            >
              {t("trackers.get_token_link", "Get AniList Access Token ↗")}
            </a>

            <button
              type="submit"
              className="btn btn-primary btn-sm gap-2"
              disabled={connectMutation.isPending}
            >
              {connectMutation.isPending ? <span className="loading loading-spinner loading-xs" /> : <Check className="h-4 w-4" />}
              {t("trackers.connect_btn", "Connect AniList")}
            </button>
          </div>
        </form>
      )}
    </div>
  );
};
