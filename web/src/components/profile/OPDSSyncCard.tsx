import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { Copy, Check, QrCode, BookOpen, Search } from "lucide-react";
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
        <div className="w-10 h-10 shrink-0 aspect-square grid place-items-center rounded-xl bg-primary/10 text-primary mt-0.5">
          <BookOpen className="h-5 w-5" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2 mb-1">
            <h3 className="text-base font-bold leading-tight text-base-content">
              {t("opds.title", "OPDS 1.2 & 2.0 Catalog Feeds")}
            </h3>
            <span className="badge badge-primary badge-sm shrink-0 font-medium whitespace-nowrap">
              {t("common.active", "Active")}
            </span>
          </div>
          <p className="text-xs text-base-content/60 leading-relaxed">
            {t(
              "opds.subtitle",
              "Connect Moon+ Reader, Thorium Reader, Cantook, Yomu, or Mihon to browse and download books.",
            )}
          </p>
        </div>
      </div>

      <div className="space-y-3">
        {/* OPDS 1.2 Catalog Feed */}
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
          <div className="flex gap-2">
            <input
              type="text"
              readOnly
              value={opdsV1Url}
              className="input input-sm input-bordered font-mono text-xs flex-1 bg-base-200/50"
            />
            <button
              type="button"
              onClick={() => copyToClipboard(opdsV1Url, "OPDS Feed URL")}
              className="btn btn-sm btn-outline gap-1 px-3"
            >
              {copiedUrl === opdsV1Url ? (
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
              onClick={() => toggleQr(opdsV1Url)}
              className={`btn btn-sm ${activeQrUrl === opdsV1Url ? "btn-primary" : "btn-outline"} px-3`}
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
          <div className="flex gap-2">
            <input
              type="text"
              readOnly
              value={openSearchUrl}
              className="input input-sm input-bordered font-mono text-xs flex-1 bg-base-200/50"
            />
            <button
              type="button"
              onClick={() => copyToClipboard(openSearchUrl, "OpenSearch URL")}
              className="btn btn-sm btn-outline gap-1 px-3"
            >
              {copiedUrl === openSearchUrl ? (
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

        {/* OPDS 2.0 JSON Catalog */}
        <div>
          <div className="flex items-center justify-between mb-1">
            <label className="text-xs font-bold uppercase tracking-wider text-base-content/70 flex items-center gap-1.5">
              <BookOpen className="h-3.5 w-3.5 text-accent" />
              {t(
                "opds.v2_label",
                "OPDS 2.0 JSON Catalog (Thorium Reader, Cantook)",
              )}
            </label>
          </div>
          <div className="flex gap-2">
            <input
              type="text"
              readOnly
              value={opdsV2Url}
              className="input input-sm input-bordered font-mono text-xs flex-1 bg-base-200/50"
            />
            <button
              type="button"
              onClick={() => copyToClipboard(opdsV2Url, "OPDS 2.0 URL")}
              className="btn btn-sm btn-outline gap-1 px-3"
            >
              {copiedUrl === opdsV2Url ? (
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
              onClick={() => toggleQr(opdsV2Url)}
              className={`btn btn-sm ${activeQrUrl === opdsV2Url ? "btn-accent" : "btn-outline"} px-3`}
              title={t("opds.show_qr", "Show QR Code")}
            >
              <QrCode className="h-4 w-4" />
            </button>
          </div>
        </div>

        {activeQrUrl && (
          <div className="mt-3 p-4 bg-base-200/40 rounded-xl flex flex-col items-center justify-center gap-2 border border-base-200">
            <CustomQRCode value={activeQrUrl} size={160} />
            <span className="text-[11px] font-mono text-base-content/60 text-center break-all">
              {activeQrUrl}
            </span>
          </div>
        )}

        <div className="rounded-xl bg-base-200/40 p-4 text-xs space-y-1.5 border border-base-200">
          <p className="font-bold text-base-content/80">
            {t(
              "opds.guide_title",
              "How to connect Moon+ Reader / Thorium / Mihon:",
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
                "opds.step_thorium",
                "Thorium Reader: Open Catalogs -> Add an OPDS Feed -> Enter OPDS 2.0 URL -> Browse and read.",
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
