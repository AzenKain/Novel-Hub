import { Loader2, ShieldCheck } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";

type Props = {
  onSubmit: (code: string) => void;
  pending?: boolean;
};

// Accepts a recovery code as well as a 6-digit one, which is why the input is not numeric-only.
export function TOTPCodeStep({ onSubmit, pending }: Props) {
  const { t } = useTranslation();
  const [code, setCode] = useState("");

  const trimmed = code.trim();

  return (
    <div className="flex flex-col gap-3">
      <p className="text-xs text-base-content/60">
        {t("auth.totp_prompt", "Enter the 6-digit code from your authenticator app, or a recovery code.")}
      </p>
      <div className="form-control">
        <label className="label">
          <span className="label-text font-semibold">{t("auth.totp_code", "Authentication code")}</span>
        </label>
        <input
          type="text"
          inputMode="text"
          autoComplete="one-time-code"
          autoFocus
          maxLength={16}
          value={code}
          onChange={(event) => setCode(event.target.value)}
          placeholder="000000"
          className="input input-bordered w-full tracking-[0.3em] text-center font-mono"
        />
      </div>
      <button
        type="button"
        className="btn btn-primary w-full gap-2"
        disabled={trimmed.length < 6 || pending}
        onClick={() => onSubmit(trimmed)}
      >
        {pending ? <Loader2 className="animate-spin" size={18} /> : <ShieldCheck size={18} />}
        {t("auth.totp_verify", "Verify and sign in")}
      </button>
    </div>
  );
}
