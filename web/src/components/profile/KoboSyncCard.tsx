import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Copy,
  Check,
  QrCode,
  Smartphone,
  RefreshCw,
  Trash2,
  AlertTriangle,
} from "lucide-react";
import { toast } from "react-toastify";
import { CustomQRCode } from "@/components/common/CustomQRCode";
import { ConfirmModal } from "@/components/common";
import {
  useKoboSetupQuery,
  useRegenerateKoboSetupMutation,
  useRevokeKoboSetupMutation,
} from "@/hooks";
import { copyText } from "@/utils/clipboard";

export const KoboSyncCard: React.FC = () => {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);
  const [showQr, setShowQr] = useState(false);

  const { data: setup, isLoading, error } = useKoboSetupQuery();
  const regenerate = useRegenerateKoboSetupMutation();
  const revoke = useRevokeKoboSetupMutation();

  const [confirmState, setConfirmState] = useState<{
    open: boolean;
    title: string;
    message: string;
    onConfirm: () => void;
    variant: "warning" | "danger" | "info" | "success";
  }>({
    open: false,
    title: "",
    message: "",
    onConfirm: () => {},
    variant: "info",
  });

  const endpointUrl = setup?.endpoint_url ?? "";
  const configSnippet = `[FeatureSettings]\napi_endpoint=${endpointUrl}`;

  const handleCopy = () => {
    if (!endpointUrl) return;
    copyText(configSnippet).then((success) => {
      if (success) {
        setCopied(true);
        toast.success(t("common.copied", "Configuration copied to clipboard!"));
        setTimeout(() => setCopied(false), 2000);
      }
    });
  };

  const handleRegenerate = () => {
    setConfirmState({
      open: true,
      title: t("kobo.regenerate_confirm_title", "Regenerate Kobo Endpoint"),
      message: t(
        "kobo.regenerate_confirm",
        "This unpairs any device using the current URL. Continue?",
      ),
      variant: "warning",
      onConfirm: () => {
        setConfirmState((prev) => ({ ...prev, open: false }));
        regenerate.mutate(undefined, {
          onSuccess: () => {
            setShowQr(false);
            toast.success(
              t(
                "kobo.regenerate_success",
                "New Kobo endpoint generated. Update your device.",
              ),
            );
          },
          onError: (err) =>
            toast.error(
              err.message ||
                t("kobo.regenerate_error", "Failed to regenerate endpoint"),
            ),
        });
      },
    });
  };

  const handleRevoke = () => {
    setConfirmState({
      open: true,
      title: t("kobo.revoke_confirm_title", "Revoke Kobo Sync"),
      message: t(
        "kobo.revoke_confirm",
        "This stops all Kobo devices from syncing. Continue?",
      ),
      variant: "danger",
      onConfirm: () => {
        setConfirmState((prev) => ({ ...prev, open: false }));
        revoke.mutate(undefined, {
          onSuccess: () => {
            setShowQr(false);
            toast.success(t("kobo.revoke_success", "Kobo sync revoked."));
          },
          onError: (err) =>
            toast.error(
              err.message ||
                t("kobo.revoke_error", "Failed to revoke endpoint"),
            ),
        });
      },
    });
  };

  return (
    <div className="rounded-2xl border border-base-300 bg-base-100 p-6 shadow-sm space-y-4">
      <div className="flex items-start gap-3 border-b border-base-200 pb-3">
        <div className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-accent/10 text-accent mt-0.5">
          <Smartphone className="h-5 w-5" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2 mb-1">
            <h3 className="text-base font-bold leading-tight text-base-content">
              {t("kobo.title", "Kobo E-Reader Wi-Fi Sync")}
            </h3>
            {endpointUrl ? (
              <span className="badge badge-success badge-sm shrink-0 font-medium whitespace-nowrap">
                {t("kobo.paired", "Paired")}
              </span>
            ) : (
              <span className="badge badge-ghost badge-sm shrink-0 font-medium whitespace-nowrap">
                {t("kobo.not_paired", "Not set up")}
              </span>
            )}
          </div>
          <p className="text-xs text-base-content/60 leading-relaxed">
            {t(
              "kobo.subtitle",
              "Sync your library and reading position to a Kobo e-reader over Wi-Fi.",
            )}
          </p>
        </div>
      </div>

      {error && (
        <div className="alert alert-error text-xs">
          <span>
            {t("kobo.load_error", "Could not load your Kobo endpoint.")}
          </span>
        </div>
      )}

      {/* Warned inline rather than blocking: a Kobo cannot resolve localhost, and pasting a
          loopback URL is the most common way this setup silently fails. */}
      {setup?.is_local_address && (
        <div className="alert alert-warning text-xs">
          <AlertTriangle className="h-4 w-4 shrink-0" />
          <span>
            {t(
              "kobo.local_address_warning",
              "This URL points at localhost, which a Kobo device cannot reach. Set the Server URL under Admin → Settings to an address on your network.",
            )}
          </span>
        </div>
      )}

      <div className="space-y-3">
        <div>
          <label className="block text-xs font-bold uppercase tracking-wider text-base-content/70 mb-1">
            {t("kobo.endpoint_label", "Kobo API Endpoint URL")}
          </label>
          <div className="flex items-center gap-2">
            <input
              type="text"
              readOnly
              value={
                isLoading ? t("common.loading", "Loading...") : endpointUrl
              }
              placeholder={t("kobo.endpoint_placeholder", "No endpoint yet")}
              className="input input-bordered w-full font-mono text-xs bg-base-200/50"
            />
            <button
              type="button"
              onClick={handleCopy}
              disabled={!endpointUrl}
              className="btn btn-outline btn-square"
              title={t("common.copy", "Copy Configuration")}
            >
              {copied ? (
                <Check className="h-4 w-4 text-success" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </button>
            <button
              type="button"
              onClick={() => setShowQr(!showQr)}
              disabled={!endpointUrl}
              className="btn btn-outline btn-square"
              title={t("kobo.show_qr", "Show QR Code")}
            >
              <QrCode className="h-4 w-4" />
            </button>
          </div>
          {/* The URL is the credential — there is no second factor, so say so plainly. */}
          <p className="mt-1 text-[11px] text-base-content/60">
            {t(
              "kobo.secret_warning",
              "Anyone with this URL can read your library. Do not share it; regenerate if it leaks.",
            )}
          </p>
        </div>

        {showQr && endpointUrl && (
          <CustomQRCode value={endpointUrl} label={endpointUrl} size={180} />
        )}

        <div className="rounded-xl bg-base-200/50 p-4 text-xs space-y-2">
          <p className="font-bold text-base-content/80">
            {t("kobo.instructions_header", "How to setup your Kobo device:")}
          </p>
          <ol className="list-decimal list-inside space-y-1 text-base-content/70">
            <li>
              {t(
                "kobo.step1",
                "Connect your Kobo e-reader to your computer via USB cable.",
              )}
            </li>
            <li>
              {t(
                "kobo.step2",
                "Open the file .kobo/Kobo/KoboeReader.conf in a text editor.",
              )}
            </li>
            <li>
              {t(
                "kobo.step3",
                "Add or update the [FeatureSettings] section with api_endpoint=",
              )}
              <code className="bg-base-300 px-1 rounded break-all">
                {endpointUrl}
              </code>
            </li>
            <li>
              {t(
                "kobo.step4",
                "Safely eject your Kobo and tap 'Sync' over Wi-Fi!",
              )}
            </li>
          </ol>
        </div>

        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={handleRegenerate}
            disabled={regenerate.isPending}
            className="btn btn-outline btn-sm gap-2"
          >
            <RefreshCw
              className={`h-4 w-4 ${regenerate.isPending ? "animate-spin" : ""}`}
            />
            {t("kobo.regenerate", "Regenerate URL")}
          </button>
          <button
            type="button"
            onClick={handleRevoke}
            disabled={revoke.isPending || !endpointUrl}
            className="btn btn-outline btn-error btn-sm gap-2"
          >
            <Trash2 className="h-4 w-4" />
            {t("kobo.revoke", "Revoke access")}
          </button>
        </div>
      </div>

      <ConfirmModal
        open={confirmState.open}
        title={confirmState.title}
        message={confirmState.message}
        onClose={() => setConfirmState((prev) => ({ ...prev, open: false }))}
        onConfirm={confirmState.onConfirm}
        variant={confirmState.variant}
        loading={regenerate.isPending || revoke.isPending}
      />
    </div>
  );
};
