import React, { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { BookOpen, Check, ExternalLink, Loader2 } from "lucide-react";
import { useConnectHardcoverMutation } from "@/hooks";
import { usePublicSettings } from "@/hooks/useSettings";
import { toast } from "react-toastify";
import { useAuthStore } from "@/stores";
import { hasPermission } from "@/utils/permission";

export const HardcoverTrackerCard: React.FC = () => {
  const { t } = useTranslation();
  const user = useAuthStore((state) => state.user);
  const settings = usePublicSettings();
  const [connected, setConnected] = useState(false);

  const connectMutation = useConnectHardcoverMutation();

  useEffect(() => {
    if (new URLSearchParams(window.location.search).get("hardcover") === "connected") {
      setConnected(true);
      window.history.replaceState({}, "", window.location.pathname);
    }
  }, []);

  if (!settings?.enable_hardcover_scrobbling) return null;
  if (!hasPermission(user, "tracker.sync")) return null;

  const handleConnect = (e: React.FormEvent) => {
    e.preventDefault();
    connectMutation.mutate(
      undefined,
      {
        onSuccess: (authorizeUrl) => {
          window.location.href = authorizeUrl;
        },
        onError: (err: any) => {
          toast.error(err?.message || t("trackers.hardcover_connect_failed", "Failed to start Hardcover connect"));
        },
      }
    );
  };

  return (
    <div className="rounded-2xl border border-base-300 bg-base-100 p-6 shadow-sm space-y-4">
      <div className="flex items-start gap-3 border-b border-base-200 pb-3">
        <div className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-info/10 text-info mt-0.5">
          {connected ? <Check className="h-5 w-5" /> : <BookOpen className="h-5 w-5" />}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2 mb-1">
            <h3 className="text-base font-bold leading-tight text-base-content">
              {t("trackers.hardcover_title", "Hardcover Scrobbling")}
            </h3>
            {connected && (
              <span className="badge badge-success badge-sm shrink-0 font-medium whitespace-nowrap">
                {t("common.connected", "Connected")}
              </span>
            )}
          </div>
          <p className="text-xs text-base-content/60 leading-relaxed">
            {t("trackers.hardcover_subtitle", "Scrobble your reading progress to your Hardcover profile automatically.")}
          </p>
        </div>
      </div>

      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
        <button
          type="button"
          className="btn btn-primary btn-sm gap-2"
          onClick={handleConnect}
          disabled={connectMutation.isPending}
        >
          {connectMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <ExternalLink className="h-4 w-4" />}
          {t("trackers.hardcover_connect_btn", "Connect Hardcover")}
        </button>
        <a
          href="https://hardcover.app/account/developer"
          target="_blank"
          rel="noopener noreferrer"
          className="text-xs text-primary hover:underline flex items-center gap-1"
        >
          {t("trackers.hardcover_dev_link", "Register a developer app ↗")}
        </a>
      </div>
    </div>
  );
};