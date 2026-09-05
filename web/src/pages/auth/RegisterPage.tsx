import { OTPCodeStep, PasswordStrength, TopNav } from "@/components/common";
import { usePublicSettings, useRegisterMutation } from "@/hooks";
import { BookOpen, Loader2 } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useNavigate } from "react-router-dom";

export function RegisterPage() {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const settings = usePublicSettings();
  const registerMutation = useRegisterMutation();

  const [form, setForm] = useState({ email: "", password: "", full_name: "" });
  const [confirmPassword, setConfirmPassword] = useState("");
  const [validationError, setValidationError] = useState("");
  const [ticket, setTicket] = useState("");

  const verifyRequired = settings?.require_email_verify ?? false;
  const needsVerification = verifyRequired && !ticket;

  const handleSubmit = (e: React.SyntheticEvent) => {
    e.preventDefault();
    setValidationError("");
    if (form.password !== confirmPassword) {
      setValidationError(
        t("settings.password_mismatch", "New passwords do not match"),
      );
      return;
    }
    registerMutation.mutate(
      {
        email: form.email,
        password: form.password,
        full_name: form.full_name || undefined,
        otp_ticket: ticket || undefined,
      },
      { onSuccess: () => navigate("/", { replace: true }) },
    );
  };

  if (settings && !settings.registration_enabled) {
    return (
      <div className="min-h-screen bg-base-100 flex flex-col font-sans">
        <TopNav showSidebarToggle={false} />
        <div className="flex-1 flex items-center justify-center p-4">
          <div className="card w-full max-w-md bg-base-100 shadow-xl border border-base-200 text-center">
            <div className="card-body items-center gap-4">
              <BookOpen size={40} className="text-base-content/30" />
              <h2 className="text-xl font-bold">
                {t("auth.registration_disabled", "Registration Disabled")}
              </h2>
              <p className="text-sm text-base-content/60">
                {t(
                  "auth.registration_disabled_desc",
                  "Public registration is currently disabled by the administrator.",
                )}
              </p>
              <Link to="/" className="btn btn-primary btn-sm">
                {t("auth.go_home", "Go Home")}
              </Link>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-base-100 flex flex-col font-sans">
      <TopNav showSidebarToggle={false} />
      <div className="flex-1 flex items-center justify-center p-4">
        <div className="card w-full max-w-md bg-base-100 shadow-xl border border-base-200">
          <div className="card-body">
            <div className="flex flex-col items-center gap-2 mb-4">
              <div className="w-14 h-14 rounded-2xl bg-primary/10 flex items-center justify-center">
                <BookOpen size={28} className="text-primary" />
              </div>
              <h2 className="text-2xl font-bold">
                {t("auth.create_account", "Create Account")}
              </h2>
              <p className="text-sm text-base-content/60">
                {t(
                  "auth.register_desc",
                  "Register to access the library features.",
                )}
              </p>
            </div>

            <form onSubmit={handleSubmit} className="flex flex-col gap-3">
              <div className="form-control">
                <label className="label">
                  <span className="label-text font-semibold">
                    {t("auth.email", "Email")}
                  </span>
                </label>
                <input
                  type="email"
                  placeholder="account@example.com"
                  className="input input-bordered w-full"
                  value={form.email}
                  onChange={(e) => {
                    setForm({ ...form, email: e.target.value });
                    setTicket("");
                  }}
                  required
                  autoComplete="email"
                />
              </div>

              {needsVerification ? (
                <OTPCodeStep
                  email={form.email}
                  purpose="email_verify"
                  onVerified={setTicket}
                />
              ) : (
                <>
                  {verifyRequired && (
                    <div className="alert alert-success py-2 text-sm rounded-lg">
                      {t("auth.otp_verified", "Email verified")}
                    </div>
                  )}
                  <div className="form-control">
                    <label className="label">
                      <span className="label-text font-semibold">
                        {t("auth.full_name", "Full Name")}
                      </span>
                    </label>
                    <input
                      type="text"
                      placeholder={t("auth.optional", "(optional)")}
                      className="input input-bordered w-full"
                      value={form.full_name}
                      onChange={(e) =>
                        setForm({ ...form, full_name: e.target.value })
                      }
                    />
                  </div>
                  <div className="form-control">
                    <label className="label">
                      <span className="label-text font-semibold">
                        {t("auth.password", "Password")}
                      </span>
                    </label>
                    <input
                      type="password"
                      placeholder={t(
                        "auth.password_min",
                        "Minimum 8 characters",
                      )}
                      className="input input-bordered w-full"
                      value={form.password}
                      onChange={(e) =>
                        setForm({ ...form, password: e.target.value })
                      }
                      required
                      minLength={8}
                      autoComplete="new-password"
                    />
                    {form.password.length > 0 && (
                      <PasswordStrength password={form.password} />
                    )}
                  </div>
                  <div className="form-control">
                    <label className="label">
                      <span className="label-text font-semibold">
                        {t("settings.confirm_password", "Confirm new password")}
                      </span>
                    </label>
                    <input
                      type="password"
                      placeholder={t(
                        "auth.password_min",
                        "Minimum 8 characters",
                      )}
                      className="input input-bordered w-full"
                      value={confirmPassword}
                      onChange={(e) => setConfirmPassword(e.target.value)}
                      required
                      minLength={8}
                      autoComplete="new-password"
                    />
                  </div>

                  {(validationError || registerMutation.error) && (
                    <div className="alert alert-error py-2 text-sm rounded-lg">
                      {validationError ||
                        (registerMutation.error instanceof Error
                          ? registerMutation.error.message
                          : String(registerMutation.error))}
                    </div>
                  )}

                  <button
                    className="btn btn-primary mt-2"
                    disabled={registerMutation.isPending}
                  >
                    {registerMutation.isPending ? (
                      <Loader2 className="animate-spin" size={20} />
                    ) : null}
                    {t("auth.register", "Register")}
                  </button>
                </>
              )}
            </form>

            <div className="text-center mt-2">
              <Link to="/" className="text-sm link link-hover">
                {t("auth.back_to_library", "Back to library")}
              </Link>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
