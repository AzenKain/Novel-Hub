import { useRequestOTPMutation, useVerifyOTPMutation } from "@/hooks";
import type { OTPPurpose } from "@/types";
import { Loader2, MailCheck } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

type Props = {
  email: string;
  purpose: OTPPurpose;
  onVerified: (ticket: string) => void;
};

export function OTPCodeStep({ email, purpose, onVerified }: Props) {
  const { t } = useTranslation();
  const requestMutation = useRequestOTPMutation();
  const verifyMutation = useVerifyOTPMutation();
  const [code, setCode] = useState("");
  const [cooldown, setCooldown] = useState(0);

  const sendCode = () => {
    requestMutation.mutate(
      { email, purpose },
      { onSuccess: (data) => setCooldown(data.cooldown_seconds) }
    );
  };

  useEffect(() => {
    if (cooldown <= 0) return;
    const timer = setInterval(() => setCooldown((value) => (value <= 1 ? 0 : value - 1)), 1000);
    return () => clearInterval(timer);
  }, [cooldown]);

  const sent = requestMutation.isSuccess;
  const error = requestMutation.error || verifyMutation.error;

  return (
    <div className="flex flex-col gap-3">
      {!sent ? (
        <button
          type="button"
          className="btn btn-outline w-full gap-2"
          onClick={sendCode}
          disabled={requestMutation.isPending || !email}
        >
          {requestMutation.isPending ? <Loader2 className="animate-spin" size={18} /> : <MailCheck size={18} />}
          {t("auth.otp_send", "Send verification code")}
        </button>
      ) : (
        <>
          <p className="text-xs text-base-content/60">
            {t("auth.otp_sent_to", "We sent a 6-digit code to {{email}}.", { email })}
          </p>
          <div className="form-control">
            <label className="label">
              <span className="label-text font-semibold">{t("auth.otp_code", "Verification code")}</span>
            </label>
            <input
              type="text"
              inputMode="numeric"
              autoComplete="one-time-code"
              maxLength={6}
              value={code}
              onChange={(event) => setCode(event.target.value.replace(/\D/g, ""))}
              placeholder="000000"
              className="input input-bordered w-full tracking-[0.5em] text-center font-mono"
            />
          </div>
          <button
            type="button"
            className="btn btn-primary w-full"
            disabled={code.length !== 6 || verifyMutation.isPending}
            onClick={() =>
              verifyMutation.mutate(
                { email, purpose, code },
                { onSuccess: (data) => onVerified(data.otp_ticket) }
              )
            }
          >
            {verifyMutation.isPending ? <Loader2 className="animate-spin" size={18} /> : null}
            {t("auth.otp_verify", "Verify code")}
          </button>
          <button
            type="button"
            className="btn btn-ghost btn-xs"
            onClick={sendCode}
            disabled={cooldown > 0 || requestMutation.isPending}
          >
            {cooldown > 0
              ? t("auth.otp_resend_in", "Resend in {{seconds}}s", { seconds: cooldown })
              : t("auth.otp_resend", "Resend code")}
          </button>
        </>
      )}

      {error && (
        <div className="alert alert-error py-2 text-sm rounded-lg">
          {error instanceof Error ? error.message : String(error)}
        </div>
      )}
    </div>
  );
}
