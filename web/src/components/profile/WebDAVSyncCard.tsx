import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { Copy, Check, QrCode, HardDrive, FolderSync } from "lucide-react";
import { toast } from "react-toastify";
import { CustomQRCode } from "@/components/common/CustomQRCode";
import { copyText } from "@/utils/clipboard";
import { useAuthStore } from "@/stores";
import { hasPermission } from "@/utils/permission";

export const WebDAVSyncCard: React.FC = () => {
  const { t } = useTranslation();
  const user = useAuthStore((state) => state.user);
  const [copiedUrl, setCopiedUrl] = useState<string | null>(null);
  const [showQr, setShowQr] = useState<boolean>(false);

  if (!hasPermission(user, "webdav.read")) {
    return null;
  }

  const origin = window.location.origin;
  const webdavUrl = `${origin}/webdav/`;

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
        <div className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-secondary/10 text-secondary mt-0.5">
          <HardDrive className="h-5 w-5" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2 mb-1">
            <h3 className="text-base font-bold leading-tight text-base-content">
              {t("webdav.title", "WebDAV Server (RFC 4918)")}
            </h3>
            <span className="badge badge-secondary badge-sm shrink-0 font-medium">
              {t("common.active", "Active")}
            </span>
          </div>
          <p className="text-xs text-base-content/60 leading-relaxed">
            {t(
              "webdav.subtitle",
              "Connect Moon+ Reader, KyBook 3, FBReader, Marvin, Foliate, Zotero, or OS file explorers to browse and download books.",
            )}
          </p>
        </div>
      </div>

      <div className="space-y-3">
        <div>
          <div className="flex items-center justify-between mb-1">
            <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 flex items-center gap-1.5">
              <FolderSync className="h-3.5 w-3.5 text-secondary" />
              {t("webdav.url_label", "WebDAV Endpoint URL")}
            </label>
          </div>
          <div className="flex gap-2">
            <input
              type="text"
              readOnly
              value={webdavUrl}
              className="input input-sm input-bordered font-mono text-xs flex-1 bg-base-200/50"
            />
            <button
              type="button"
              onClick={() => copyToClipboard(webdavUrl, "WebDAV URL")}
              className="btn btn-sm btn-outline gap-1 px-3"
            >
              {copiedUrl === webdavUrl ? (
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
              className={`btn btn-sm ${showQr ? "btn-secondary" : "btn-outline"} px-3`}
              title={t("common.show_qr", "Show QR Code")}
            >
              <QrCode className="h-4 w-4" />
            </button>
          </div>
          {showQr && (
            <div className="mt-3 p-4 bg-base-200/40 rounded-xl flex flex-col items-center justify-center gap-2 border border-base-200">
              <CustomQRCode value={webdavUrl} size={160} />
              <span className="text-[11px] font-mono text-base-content/60 text-center break-all">
                {webdavUrl}
              </span>
            </div>
          )}
        </div>

        {/* Quick Guide */}
        <div className="rounded-xl bg-base-200/40 p-4 text-xs space-y-1.5 border border-base-200">
          <p className="font-bold text-base-content/80">
            {t("webdav.guide_title", "How to connect via WebDAV:")}
          </p>
          <ul className="list-disc list-inside space-y-1 text-base-content/70">
            <li>
              {t(
                "webdav.step1",
                "Moon+ Reader / KyBook 3: Open Net Library -> Add WebDAV -> Paste the URL above.",
              )}
            </li>
            <li>
              {t(
                "webdav.step2",
                "Authentication: Enter your NovelHub Email/Username and Password (or Magic Code).",
              )}
            </li>
            <li>
              {t(
                "webdav.step3",
                "Browse your libraries as folders and download books directly to your device.",
              )}
            </li>
          </ul>
        </div>
      </div>
    </div>
  );
};
