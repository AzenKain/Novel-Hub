import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { Copy, Check, QrCode, BookOpen, Search, RefreshCw } from "lucide-react";
import { toast } from "react-toastify";
import { CustomQRCode } from "@/components/common/CustomQRCode";
import { copyText } from "@/utils/clipboard";

export const OPDSSyncCard: React.FC = () => {
  const { t } = useTranslation();
  const [copiedUrl, setCopiedUrl] = useState<string | null>(null);
  const [activeQrUrl, setActiveQrUrl] = useState<string | null>(null);

  const origin = window.location.origin;
  const opdsV1Url = `${origin}/api/opds/v1`;
  const opdsV2Url = `${origin}/api/opds/v2/catalog`;
  const openSearchUrl = `${origin}/api/opds/v1/opensearch.xml`;
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

  const toggleQr = (url: string) => {
    setActiveQrUrl(activeQrUrl === url ? null : url);
  };

  return (
    <div className="rounded-2xl border border-base-300 bg-base-100 p-6 shadow-sm space-y-4">
      <div className="flex items-start gap-3 border-b border-base-200 pb-3">
        <div className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-primary/10 text-primary mt-0.5">
          <BookOpen className="h-5 w-5" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2 mb-1">
            <h3 className="text-base font-bold leading-tight text-base-content">
              {t("opds.title", "OPDS 2.0 Catalog & Reading Progress Sync")}
            </h3>
            <span className="badge badge-primary badge-sm shrink-0 font-medium">
              {t("common.active", "Active")}
            </span>
          </div>
          <p className="text-xs text-base-content/60 leading-relaxed">
            {t(
              "opds.subtitle",
              "Connect Moon+ Reader, KOReader, Mihon, Yomu, or any e-reader to download books & sync reading progress 2-way.",
            )}
          </p>
        </div>
      </div>

      <div className="space-y-4">
        {/* OPDS 1.2 / 2.0 Feed */}
        <div>
          <div className="flex items-center justify-between mb-1">
            <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 flex items-center gap-1.5">
              <BookOpen className="h-3.5 w-3.5 text-primary" />
              {t(
                "opds.v1_label",
                "OPDS 1.2 Catalog Feed (Moon+, KOReader, Yomu)",
              )}
            </label>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="text"
              readOnly
              value={opdsV1Url}
              className="input input-bordered w-full font-mono text-xs bg-base-200/50"
            />
            <button
              type="button"
              onClick={() => copyToClipboard(opdsV1Url, "OPDS Feed URL")}
              className="btn btn-outline btn-square"
              title={t("common.copy", "Copy URL")}
            >
              {copiedUrl === opdsV1Url ? (
                <Check className="h-4 w-4 text-success" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </button>
            <button
              type="button"
              onClick={() => toggleQr(opdsV1Url)}
              className="btn btn-outline btn-square"
              title={t("opds.show_qr", "Show QR Code")}
            >
              <QrCode className="h-4 w-4" />
            </button>
          </div>
        </div>

        {/* OpenSearch XML */}
        <div>
          <div className="flex items-center justify-between mb-1">
            <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 flex items-center gap-1.5">
              <Search className="h-3.5 w-3.5 text-secondary" />
              {t("opds.opensearch_label", "OpenSearch XML Template")}
            </label>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="text"
              readOnly
              value={openSearchUrl}
              className="input input-bordered w-full font-mono text-xs bg-base-200/50"
            />
            <button
              type="button"
              onClick={() => copyToClipboard(openSearchUrl, "OpenSearch URL")}
              className="btn btn-outline btn-square"
              title={t("common.copy", "Copy URL")}
            >
              {copiedUrl === openSearchUrl ? (
                <Check className="h-4 w-4 text-success" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </button>
          </div>
        </div>

        {/* OPDS 2.0 JSON Catalog */}
        <div>
          <div className="flex items-center justify-between mb-1">
            <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 flex items-center gap-1.5">
              <BookOpen className="h-3.5 w-3.5 text-accent" />
              {t("opds.v2_label", "OPDS 2.0 JSON Catalog")}
            </label>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="text"
              readOnly
              value={opdsV2Url}
              className="input input-bordered w-full font-mono text-xs bg-base-200/50"
            />
            <button
              type="button"
              onClick={() => copyToClipboard(opdsV2Url, "OPDS 2.0 URL")}
              className="btn btn-outline btn-square"
              title={t("common.copy", "Copy URL")}
            >
              {copiedUrl === opdsV2Url ? (
                <Check className="h-4 w-4 text-success" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </button>
          </div>
        </div>

        {/* KOReader Kosync 2-Way Progress Sync */}
        <div>
          <div className="flex items-center justify-between mb-1">
            <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 flex items-center gap-1.5">
              <RefreshCw className="h-3.5 w-3.5 text-info" />
              {t(
                "opds.koreader_sync_label",
                "KOReader 2-Way Progress Sync URL",
              )}
            </label>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="text"
              readOnly
              value={koreaderSyncUrl}
              className="input input-bordered w-full font-mono text-xs bg-base-200/50"
            />
            <button
              type="button"
              onClick={() =>
                copyToClipboard(koreaderSyncUrl, "KOReader Sync URL")
              }
              className="btn btn-outline btn-square"
              title={t("common.copy", "Copy URL")}
            >
              {copiedUrl === koreaderSyncUrl ? (
                <Check className="h-4 w-4 text-success" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </button>
            <button
              type="button"
              onClick={() => toggleQr(koreaderSyncUrl)}
              className="btn btn-outline btn-square"
              title={t("opds.show_qr", "Show QR Code")}
            >
              <QrCode className="h-4 w-4" />
            </button>
          </div>
        </div>

        {activeQrUrl && (
          <CustomQRCode value={activeQrUrl} label={activeQrUrl} size={180} />
        )}

        <div className="rounded-xl bg-base-200/50 p-4 text-xs space-y-2">
          <p className="font-bold text-base-content/80">
            {t(
              "opds.guide_title",
              "How to connect Moon+ Reader / KOReader / Mihon:",
            )}
          </p>
          <ul className="list-disc list-inside space-y-1 text-base-content/70">
            <li>
              {t(
                "opds.step1",
                "Moon+ Reader: Open Net Library -> Add new catalog -> Enter OPDS Feed URL.",
              )}
            </li>
            <li>
              {t(
                "opds.step2",
                "KOReader: Open OPDS Catalog -> Add catalog URL or set Kosync Progress Server URL.",
              )}
            </li>
            <li>
              {t(
                "opds.step3",
                "Authentication: Use your NovelHub account email and password for HTTP Basic Auth.",
              )}
            </li>
          </ul>
        </div>
      </div>
    </div>
  );
};
