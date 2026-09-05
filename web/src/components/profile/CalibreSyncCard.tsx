import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { Copy, Check, QrCode, Library, Server, Globe } from "lucide-react";
import { toast } from "react-toastify";
import { CustomQRCode } from "@/components/common/CustomQRCode";
import { copyText } from "@/utils/clipboard";

export const CalibreSyncCard: React.FC = () => {
  const { t } = useTranslation();
  const [copiedUrl, setCopiedUrl] = useState<string | null>(null);
  const [showQr, setShowQr] = useState<boolean>(false);

  const origin = window.location.origin;
  const calibreUrl = `${origin}/calibre`;
  const calibreAjaxUrl = `${origin}/calibre/ajax/library-info`;

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
        <div className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-accent/10 text-accent mt-0.5">
          <Library className="h-5 w-5" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2 mb-1">
            <h3 className="text-base font-bold leading-tight text-base-content">
              {t("calibre.title", "Calibre Content Server API")}
            </h3>
            <span className="badge badge-accent badge-sm shrink-0 font-medium whitespace-nowrap">
              {t("common.active", "Active")}
            </span>
          </div>
          <p className="text-xs text-base-content/60 leading-relaxed">
            {t(
              "calibre.subtitle",
              "Connect Calibre Companion, Aldiko, Moon+ Reader, or Calibre ecosystem apps to browse and download books.",
            )}
          </p>
        </div>
      </div>

      <div className="space-y-3">
        {/* Content Server Base URL */}
        <div>
          <div className="flex items-center justify-between mb-1">
            <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 flex items-center gap-1.5">
              <Server className="h-3.5 w-3.5 text-accent" />
              {t("calibre.url_label", "Content Server Base URL")}
            </label>
          </div>
          <div className="flex gap-2">
            <input
              type="text"
              readOnly
              value={calibreUrl}
              className="input input-sm input-bordered font-mono text-xs flex-1 bg-base-200/50"
            />
            <button
              type="button"
              onClick={() => copyToClipboard(calibreUrl, "Calibre Server URL")}
              className="btn btn-sm btn-outline gap-1 px-3"
            >
              {copiedUrl === calibreUrl ? (
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
              className={`btn btn-sm ${showQr ? "btn-accent" : "btn-outline"} px-3`}
              title={t("common.show_qr", "Show QR Code")}
            >
              <QrCode className="h-4 w-4" />
            </button>
          </div>
          {showQr && (
            <div className="mt-3 p-4 bg-base-200/40 rounded-xl flex flex-col items-center justify-center gap-2 border border-base-200">
              <CustomQRCode value={calibreUrl} size={160} />
              <span className="text-[11px] font-mono text-base-content/60 text-center break-all">
                {calibreUrl}
              </span>
            </div>
          )}
        </div>

        {/* AJAX Library Info URL */}
        <div>
          <div className="flex items-center justify-between mb-1">
            <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 flex items-center gap-1.5">
              <Globe className="h-3.5 w-3.5 text-accent" />
              {t("calibre.ajax_label", "AJAX Library Info URL")}
            </label>
          </div>
          <div className="flex gap-2">
            <input
              type="text"
              readOnly
              value={calibreAjaxUrl}
              className="input input-sm input-bordered font-mono text-xs flex-1 bg-base-200/50"
            />
            <button
              type="button"
              onClick={() =>
                copyToClipboard(calibreAjaxUrl, "AJAX Library Info URL")
              }
              className="btn btn-sm btn-outline gap-1 px-3"
            >
              {copiedUrl === calibreAjaxUrl ? (
                <Check className="h-4 w-4 text-success" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
              <span className="hidden sm:inline">
                {t("common.copy", "Copy")}
              </span>
            </button>
          </div>
        </div>

        {/* Quick Guide */}
        <div className="rounded-xl bg-base-200/40 p-4 text-xs space-y-1.5 border border-base-200">
          <p className="font-bold text-base-content/80">
            {t(
              "calibre.guide_title",
              "How to connect via Calibre Content Server:",
            )}
          </p>
          <ul className="list-disc list-inside space-y-1 text-base-content/70">
            <li>
              {t(
                "calibre.step1",
                "Calibre Companion (Android / E-Reader): In connection settings, select 'Calibre Content Server', and enter the Server URL above.",
              )}
            </li>
            <li>
              {t(
                "calibre.step2",
                "Authentication: Use your NovelHub Email/Username and Password, or Bearer token.",
              )}
            </li>
            <li>
              {t(
                "calibre.step3",
                "Calibre Desktop: Use 'Connect/share' -> 'Connect to folder' pointing to the WebDAV mount, or use the 'calibredb' CLI.",
              )}
            </li>
          </ul>
        </div>
      </div>
    </div>
  );
};
