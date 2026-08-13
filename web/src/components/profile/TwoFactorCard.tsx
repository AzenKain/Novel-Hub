import { CustomQRCode } from "@/components/common/CustomQRCode";
import { ConfirmModal } from "@/components/common";
import {
  useTOTPConfirmMutation,
  useTOTPDisableMutation,
  useTOTPEnrollMutation,
  useTOTPRecoveryCodesMutation,
  useTOTPStatusQuery,
} from "@/hooks";
import { AlertTriangle, Check, Copy, KeyRound, ShieldCheck } from "lucide-react";
import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "react-toastify";

export const TwoFactorCard: React.FC = () => {
  const { t } = useTranslation();
  const { data: status, isLoading } = useTOTPStatusQuery();
  const enroll = useTOTPEnrollMutation();
  const confirm = useTOTPConfirmMutation();
  const disable = useTOTPDisableMutation();
  const regenerate = useTOTPRecoveryCodesMutation();

  const [secret, setSecret] = useState("");
  const [uri, setUri] = useState("");
  const [code, setCode] = useState("");
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);
  const [copied, setCopied] = useState(false);
  const [isDisableConfirmOpen, setIsDisableConfirmOpen] = useState(false);

  const enabled = status?.enabled ?? false;

  const reset = () => {
    setSecret("");
    setUri("");
    setCode("");
  };

  const handleEnroll = () => {
    enroll.mutate(undefined, {
      onSuccess: (data) => {
        setSecret(data.secret);
        setUri(data.provisioning_uri);
        setRecoveryCodes([]);
      },
      onError: (err) => toast.error(err.message),
    });
  };

  const handleConfirm = () => {
    confirm.mutate(code.trim(), {
      onSuccess: (data) => {
        setRecoveryCodes(data.codes);
        reset();
        toast.success(t("totp.enabled_toast", "Two-factor authentication is on."));
      },
      onError: (err) => toast.error(err.message),
    });
  };

  const handleDisable = () => {
    setIsDisableConfirmOpen(true);
  };

  const handleConfirmDisable = () => {
    setIsDisableConfirmOpen(false);
    disable.mutate(code.trim(), {
      onSuccess: () => {
        setRecoveryCodes([]);
        reset();
        toast.success(t("totp.disabled_toast", "Two-factor authentication is off."));
      },
      onError: (err) => toast.error(err.message),
    });
  };

  const handleRegenerate = () => {
    regenerate.mutate(code.trim(), {
      onSuccess: (data) => {
        setRecoveryCodes(data.codes);
        setCode("");
        toast.success(t("totp.recovery_regenerated", "New recovery codes generated. The old ones no longer work."));
      },
      onError: (err) => toast.error(err.message),
    });
  };

  const copyRecoveryCodes = () => {
    navigator.clipboard.writeText(recoveryCodes.join("\n"));
    setCopied(true);
    toast.success(t("common.copied", "Copied to clipboard"));
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="rounded-2xl border border-base-300 bg-base-100 p-6 shadow-sm space-y-4">
      <div className="flex items-center justify-between border-b border-base-200 pb-3">
        <div className="flex items-center gap-3">
          <div className="grid h-10 w-10 place-items-center rounded-xl bg-primary/10 text-primary">
            <ShieldCheck className="h-5 w-5" />
          </div>
          <div>
            <h3 className="text-base font-bold flex items-center gap-2">
              {t("totp.title", "Two-Factor Authentication")}
              {enabled ? (
                <span className="badge badge-success badge-sm">{t("totp.on", "On")}</span>
              ) : (
                <span className="badge badge-ghost badge-sm">{t("totp.off", "Off")}</span>
              )}
            </h3>
            <p className="text-xs text-base-content/60">
              {t("totp.subtitle", "Require a code from your authenticator app when signing in.")}
            </p>
          </div>
        </div>
      </div>

      {isLoading && <span className="loading loading-spinner loading-sm" />}

      {recoveryCodes.length > 0 && (
        <div className="space-y-2 rounded-xl border border-warning/40 bg-warning/10 p-4">
          <div className="flex items-start gap-2">
            <AlertTriangle className="h-4 w-4 shrink-0 text-warning mt-0.5" />
            <p className="text-xs">
              {t(
                "totp.recovery_warning",
                "Save these recovery codes now. Each works once and they are the only way in if you lose your phone. They are not shown again."
              )}
            </p>
          </div>
          <div className="grid grid-cols-2 gap-1 font-mono text-sm">
            {recoveryCodes.map((entry) => (
              <span key={entry}>{entry}</span>
            ))}
          </div>
          <button className="btn btn-xs btn-outline gap-1" onClick={copyRecoveryCodes}>
            {copied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
            {t("totp.copy_codes", "Copy codes")}
          </button>
        </div>
      )}

      {!enabled && !uri && !isLoading && (
        <button className="btn btn-primary btn-sm gap-2" onClick={handleEnroll} disabled={enroll.isPending}>
          {enroll.isPending ? <span className="loading loading-spinner loading-xs" /> : <KeyRound className="h-4 w-4" />}
          {t("totp.begin", "Set up two-factor authentication")}
        </button>
      )}

      {!enabled && uri && (
        <div className="space-y-3">
          <p className="text-xs text-base-content/70">
            {t("totp.scan_instructions", "Scan this with your authenticator app, then enter the code it shows.")}
          </p>
          <CustomQRCode value={uri} size={180} />
          <div className="text-xs">
            <span className="text-base-content/60">{t("totp.manual_entry", "Or enter this key manually:")}</span>
            <code className="ml-2 font-mono break-all">{secret}</code>
          </div>
          <div className="flex gap-2">
            <input
              value={code}
              onChange={(event) => setCode(event.target.value)}
              maxLength={6}
              inputMode="numeric"
              placeholder="000000"
              className="input input-bordered input-sm font-mono tracking-[0.3em] text-center w-36"
            />
            <button className="btn btn-primary btn-sm" onClick={handleConfirm} disabled={code.trim().length !== 6 || confirm.isPending}>
              {confirm.isPending ? <span className="loading loading-spinner loading-xs" /> : null}
              {t("totp.confirm", "Confirm")}
            </button>
            <button className="btn btn-ghost btn-sm" onClick={reset}>
              {t("common.cancel", "Cancel")}
            </button>
          </div>
        </div>
      )}

      {enabled && (
        <div className="space-y-3">
          <p className="text-xs text-base-content/60">
            {t("totp.recovery_remaining", "{{count}} recovery codes left", {
              count: status?.recovery_codes_remaining ?? 0,
            })}
          </p>
          <div className="flex flex-wrap items-center gap-2">
            <input
              value={code}
              onChange={(event) => setCode(event.target.value)}
              maxLength={16}
              placeholder={t("totp.current_code", "Current code")}
              className="input input-bordered input-sm font-mono w-40"
            />
            <button
              className="btn btn-outline btn-sm"
              onClick={handleRegenerate}
              disabled={code.trim().length < 6 || regenerate.isPending}
            >
              {t("totp.regenerate_codes", "New recovery codes")}
            </button>
            <button
              className="btn btn-error btn-outline btn-sm"
              onClick={handleDisable}
              disabled={code.trim().length < 6 || disable.isPending}
            >
              {t("totp.disable", "Turn off")}
            </button>
          </div>
        </div>
      )}

      <ConfirmModal
        open={isDisableConfirmOpen}
        title={t("totp.disable", "Turn off")}
        message={t("totp.disable_confirm", "Turn off two-factor authentication for your account?")}
        onClose={() => setIsDisableConfirmOpen(false)}
        onConfirm={handleConfirmDisable}
        variant="danger"
        loading={disable.isPending}
      />
    </div>
  );
};
