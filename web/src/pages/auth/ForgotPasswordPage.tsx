import { OTPCodeStep, PasswordStrength } from "@/components/common";
import { usePublicSettings, useResetPasswordWithOTPMutation } from "@/hooks";
import { BookOpen, KeyRound, Loader2 } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useNavigate } from "react-router-dom";

export function ForgotPasswordPage() {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const settings = usePublicSettings();
  const resetMutation = useResetPasswordWithOTPMutation();

  const [email, setEmail] = useState("");
  const [ticket, setTicket] = useState("");
  const [password, setPassword] = useState("");

  const handleSubmit = (e: React.SyntheticEvent) => {
    e.preventDefault();
    resetMutation.mutate(
      { email, otp_ticket: ticket, new_password: password },
      { onSuccess: () => navigate("/login", { replace: true }) }
    );
  };

  if (settings && !settings.password_reset_enabled) {
    return (
      <div className="min-h-screen bg-base-200 flex items-center justify-center p-4">
        <div className="card w-full max-w-md bg-base-100 shadow-xl text-center">
          <div className="card-body items-center gap-4">
            <BookOpen size={40} className="text-base-content/30" />
            <h2 className="text-xl font-bold">{t("auth.reset_disabled", "Password Reset Disabled")}</h2>
            <p className="text-sm text-base-content/60">
              {t("auth.reset_disabled_desc", "Password reset by email is currently disabled. Ask an administrator to reset your password.")}
            </p>
            <Link to="/login" className="btn btn-primary btn-sm">{t("auth.login", "Sign in")}</Link>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-base-200 flex items-center justify-center p-4">
      <div className="card w-full max-w-md bg-base-100 shadow-xl">
        <div className="card-body">
          <div className="flex flex-col items-center gap-2 mb-4">
            <div className="w-14 h-14 rounded-2xl bg-primary/10 flex items-center justify-center">
              <KeyRound size={28} className="text-primary" />
            </div>
            <h2 className="text-2xl font-bold">{t("auth.reset_title", "Reset Password")}</h2>
            <p className="text-sm text-base-content/60">
              {t("auth.reset_desc", "We'll email you a code to confirm it's your account.")}
            </p>
          </div>

          <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <div className="form-control">
              <label className="label"><span className="label-text font-semibold">{t("auth.email", "Email")}</span></label>
              <input
                type="email"
                placeholder="account@example.com"
                className="input input-bordered w-full"
                value={email}
                onChange={(e) => {
                  setEmail(e.target.value);
                  setTicket("");
                }}
                required
                autoComplete="email"
              />
            </div>

            {!ticket ? (
              <OTPCodeStep email={email} purpose="password_reset" onVerified={setTicket} />
            ) : (
              <>
                <div className="alert alert-success py-2 text-sm rounded-lg">
                  {t("auth.otp_verified", "Email verified")}
                </div>
                <div className="form-control">
                  <label className="label">
                    <span className="label-text font-semibold">{t("auth.new_password", "New password")}</span>
                  </label>
                  <input
                    type="password"
                    placeholder={t("auth.password_min", "Minimum 8 characters")}
                    className="input input-bordered w-full"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                    minLength={8}
                    autoComplete="new-password"
                  />
                  {password.length > 0 && <PasswordStrength password={password} />}
                </div>

                {resetMutation.error && (
                  <div className="alert alert-error py-2 text-sm rounded-lg">
                    {resetMutation.error instanceof Error
                      ? resetMutation.error.message
                      : String(resetMutation.error)}
                  </div>
                )}

                <button className="btn btn-primary mt-2" disabled={resetMutation.isPending}>
                  {resetMutation.isPending ? <Loader2 className="animate-spin" size={20} /> : null}
                  {t("auth.reset_submit", "Set new password")}
                </button>
              </>
            )}
          </form>

          <div className="text-center mt-2">
            <Link to="/login" className="text-sm link link-hover">{t("auth.back_to_login", "Back to sign in")}</Link>
          </div>
        </div>
      </div>
    </div>
  );
}
