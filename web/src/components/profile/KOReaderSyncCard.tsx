import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { Copy, Check, QrCode, RefreshCw, Server } from "lucide-react";
import { toast } from "react-toastify";
import { CustomQRCode } from "@/components/common/CustomQRCode";
import { copyText } from "@/utils/clipboard";

export const KOReaderSyncCard: React.FC = () => {
  const { t } = useTranslation();
  const [copiedUrl, setCopiedUrl] = useState<string | null>(null);
  const [showQr, setShowQr] = useState<boolean>(false);

  const origin = window.location.origin;
  const koreaderSyncUrl = `${origin}/api/v1/sync/koreader`;

  const copyToClipboard = (url: string, label: string) => {
    copyText(url).then((success) => {
      if (success) {
        setCopiedUrl(url);
        toast.success(t("common.copied", `${label} copied to clipboard!`));
        setTimeout(() => setCopiedUrl(null), 2000);
      }
    });
  };

  return (
    <div className="rounded-2xl border border-base-300 bg-base-100 p-6 shadow-sm space-y-4">
      <div className="flex items-start gap-3 border-b border-base-200 pb-3">
        <div className="w-10 h-10 shrink-0 aspect-square grid place-items-center rounded-xl bg-info/10 text-info mt-0.5">
          <RefreshCw className="h-5 w-5" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2 mb-1">
            <h3 className="text-base font-bold leading-tight text-base-content">
              {t("koreader.title", "KOReader 2-Way Progress Sync (Kosync)")}
            </h3>
            <span className="badge badge-info badge-sm shrink-0 font-medium whitespace-nowrap">
              {t("common.active", "Active")}
            </span>
          </div>
          <p className="text-xs text-base-content/60 leading-relaxed">
            {t(
              "koreader.subtitle",
              "Real-time two-way reading progress synchronization between KOReader devices and NovelHub.",
            )}
          </p>
        </div>
      </div>

      <div className="space-y-3">
        {/* Kosync Server URL */}
        <div>
          <div className="flex items-center justify-between mb-1">
            <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 flex items-center gap-1.5">
              <Server className="h-3.5 w-3.5 text-info" />
              {t("koreader.sync_url_label", "Kosync Progress Sync Server URL")}
            </label>
          </div>
          <div className="flex gap-2">
            <input
              type="text"
              readOnly
              value={koreaderSyncUrl}
              className="input input-sm input-bordered font-mono text-xs flex-1 bg-base-200/50"
            />
            <button
              type="button"
              onClick={() => copyToClipboard(koreaderSyncUrl, "KOReader Sync URL")}
              className="btn btn-sm btn-outline gap-1 px-3"
            >
              {copiedUrl === koreaderSyncUrl ? (
                <Check className="h-4 w-4 text-success" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
              <span className="hidden sm:inline">
                {t("common.copy", "Copy")}
              </span>
            </button>
            <button
              type="button"
              onClick={() => setShowQr(!showQr)}
              className={`btn btn-sm ${showQr ? "btn-info" : "btn-outline"} px-3`}
              title={t("opds.show_qr", "Show QR Code")}
            >
              <QrCode className="h-4 w-4" />
            </button>
          </div>
          {showQr && (
            <div className="mt-3 p-4 bg-base-200/40 rounded-xl flex flex-col items-center justify-center gap-2 border border-base-200">
              <CustomQRCode value={koreaderSyncUrl} size={160} />
              <span className="text-[11px] font-mono text-base-content/60 text-center break-all">
                {koreaderSyncUrl}
              </span>
            </div>
          )}
        </div>

        {/* Quick Guide */}
        <div className="rounded-xl bg-base-200/40 p-4 text-xs space-y-1.5 border border-base-200">
          <p className="font-bold text-base-content/80">
            {t("koreader.guide_title", "How to configure on KOReader:")}
          </p>
          <ul className="list-disc list-inside space-y-1 text-base-content/70">
            <li>
              {t(
                "koreader.step1",
                "Open top menu in KOReader -> Go to Settings (gear icon) -> Select 'Progress sync' (Kosync).",
              )}
            </li>
            <li>
              {t(
                "koreader.step2",
                "Enable 'Progress sync' -> Tap 'Custom server' -> Enter the Sync Server URL above.",
              )}
            </li>
            <li>
              {t(
                "koreader.step3",
                "Register or log in using your NovelHub account email and password.",
              )}
            </li>
          </ul>
        </div>
      </div>
    </div>
  );
};
