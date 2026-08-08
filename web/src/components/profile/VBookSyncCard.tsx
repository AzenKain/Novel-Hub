import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { Copy, Check, QrCode, BookOpen, Smartphone } from "lucide-react";
import { toast } from "react-toastify";
import { CustomQRCode } from "@/components/common/CustomQRCode";

export const VBookSyncCard: React.FC = () => {
  const { t } = useTranslation();
  const [copiedUrl, setCopiedUrl] = useState<string | null>(null);
  const [showQr, setShowQr] = useState(false);

  const origin = window.location.origin;
  const entryUrl = `${origin}/api/v1/vbook/entry.json`;

  const copyToClipboard = (url: string, label: string) => {
    navigator.clipboard.writeText(url);
    setCopiedUrl(url);
    toast.success(t("common.copied", `${label} copied to clipboard!`));
    setTimeout(() => setCopiedUrl(null), 2000);
  };

  return (
    <div className="rounded-2xl border border-base-300 bg-base-100 p-6 shadow-sm space-y-4">
      <div className="flex items-center justify-between border-b border-base-200 pb-3">
        <div className="flex items-center gap-3">
          <div className="grid h-10 w-10 place-items-center rounded-xl bg-primary/10 text-primary">
            <BookOpen className="h-5 w-5" />
          </div>
          <div>
            <h3 className="text-base font-bold flex items-center gap-2">
              {t("vbook.title", "VBook Plugin")}
              <span className="badge badge-primary badge-sm">{t("common.active", "Active")}</span>
            </h3>
            <p className="text-xs text-base-content/60">
              {t("vbook.subtitle", "Install the NovelHub plugin in VBook to browse & read your library from your phone.")}
            </p>
          </div>
        </div>
      </div>

      <div className="space-y-4">
        <div>
          <div className="flex items-center justify-between mb-1">
            <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 flex items-center gap-1.5">
              <Smartphone className="h-3.5 w-3.5 text-primary" />
              {t("vbook.entry_label", "VBook Plugin Entry URL")}
            </label>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="text"
              readOnly
              value={entryUrl}
              className="input input-bordered w-full font-mono text-xs bg-base-200/50"
            />
            <button
              type="button"
              onClick={() => copyToClipboard(entryUrl, "VBook Entry URL")}
              className="btn btn-outline btn-square"
              title={t("common.copy", "Copy URL")}
            >
              {copiedUrl === entryUrl ? <Check className="h-4 w-4 text-success" /> : <Copy className="h-4 w-4" />}
            </button>
            <button
              type="button"
              onClick={() => setShowQr(!showQr)}
              className="btn btn-outline btn-square"
              title={t("vbook.show_qr", "Show QR Code")}
            >
              <QrCode className="h-4 w-4" />
            </button>
          </div>
        </div>

        {showQr && <CustomQRCode value={entryUrl} label={entryUrl} size={180} />}

        <div className="rounded-xl bg-base-200/50 p-4 text-xs space-y-2">
          <p className="font-bold text-base-content/80">{t("vbook.guide_title", "How to install in VBook:")}</p>
          <ul className="list-disc list-inside space-y-1 text-base-content/70">
            <li>{t("vbook.step1", "Open VBook → More → Add extension → Scan the QR code or paste the entry URL.")}</li>
            <li>{t("vbook.step2", "VBook downloads the plugin and connects to your personal NovelHub server.")}</li>
            <li>{t("vbook.step3", "If your server requires login, configure your NovelHub credentials in the extension settings.")}</li>
          </ul>
        </div>
      </div>
    </div>
  );
};