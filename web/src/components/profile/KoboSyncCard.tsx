import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { Copy, Check, QrCode, Smartphone } from "lucide-react";
import { toast } from "react-toastify";
import { CustomQRCode } from "@/components/common/CustomQRCode";

export const KoboSyncCard: React.FC = () => {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);
  const [showQr, setShowQr] = useState(false);

  const koboEndpointUrl = `${window.location.origin}/kobo`;
  const koboConfigSnippet = `[FeatureSettings]\napi_endpoint=${koboEndpointUrl}`;

  const handleCopy = () => {
    navigator.clipboard.writeText(koboConfigSnippet);
    setCopied(true);
    toast.success(t("common.copied", "Configuration copied to clipboard!"));
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="rounded-2xl border border-base-300 bg-base-100 p-6 shadow-sm space-y-4">
      <div className="flex items-center justify-between border-b border-base-200 pb-3">
        <div className="flex items-center gap-3">
          <div className="grid h-10 w-10 place-items-center rounded-xl bg-accent/10 text-accent">
            <Smartphone className="h-5 w-5" />
          </div>
          <div>
            <h3 className="text-base font-bold flex items-center gap-2">
              {t("kobo.title", "Kobo E-Reader Wi-Fi Sync")}
              <span className="badge badge-success badge-sm">{t("common.active", "Active")}</span>
            </h3>
            <p className="text-xs text-base-content/60">
              {t("kobo.subtitle", "Sync books, progress, and covers natively over Wi-Fi on your Kobo Clara, Libra, Forma, or Sage.")}
            </p>
          </div>
        </div>
      </div>

      <div className="space-y-3">
        <div>
          <label className="block text-xs font-bold uppercase tracking-wider text-base-content/70 mb-1">
            {t("kobo.endpoint_label", "Kobo API Endpoint URL")}
          </label>
          <div className="flex items-center gap-2">
            <input
              type="text"
              readOnly
              value={koboEndpointUrl}
              className="input input-bordered w-full font-mono text-xs bg-base-200/50"
            />
            <button
              type="button"
              onClick={handleCopy}
              className="btn btn-outline btn-square"
              title={t("common.copy", "Copy Configuration")}
            >
              {copied ? <Check className="h-4 w-4 text-success" /> : <Copy className="h-4 w-4" />}
            </button>
            <button
              type="button"
              onClick={() => setShowQr(!showQr)}
              className="btn btn-outline btn-square"
              title={t("kobo.show_qr", "Show QR Code")}
            >
              <QrCode className="h-4 w-4" />
            </button>
          </div>
        </div>

        {showQr && (
          <CustomQRCode value={koboEndpointUrl} label={koboEndpointUrl} size={180} />
        )}

        <div className="rounded-xl bg-base-200/50 p-4 text-xs space-y-2">
          <p className="font-bold text-base-content/80">{t("kobo.instructions_header", "How to setup your Kobo device:")}</p>
          <ol className="list-decimal list-inside space-y-1 text-base-content/70">
            <li>{t("kobo.step1", "Connect your Kobo e-reader to your computer via USB cable.")}</li>
            <li>{t("kobo.step2", "Open the file .kobo/Kobo/KoboeReader.conf in a text editor.")}</li>
            <li>{t("kobo.step3", "Add or update the [FeatureSettings] section with api_endpoint=")}{<code className="bg-base-300 px-1 rounded">{koboEndpointUrl}</code>}</li>
            <li>{t("kobo.step4", "Safely eject your Kobo and tap 'Sync' over Wi-Fi!")}</li>
          </ol>
        </div>
      </div>
    </div>
  );
};
