import { OTPCodeStep, PasswordStrength } from "@/components/common";
import { usePublicSettings, useRegisterMutation } from "@/hooks";
import { useAuthStore } from "@/stores";
import { BookOpen, Loader2, X } from "lucide-react";
import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { useShallow } from "zustand/react/shallow";

export function RegisterView() {
  const { isRegisterModalOpen, setRegisterModalOpen, setLoginModalOpen } =
    useAuthStore(
      useShallow((state) => ({
        isRegisterModalOpen: state.isRegisterModalOpen,
        setRegisterModalOpen: state.setRegisterModalOpen,
        setLoginModalOpen: state.setLoginModalOpen,
      })),
    );

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
    if (form.password.length < 8) {
      setValidationError(t("auth.password_min", "Minimum 8 characters"));
      return;
    }
    if (form.password !== confirmPassword) {
      setValidationError(
        t("auth.passwords_do_not_match", "New passwords do not match"),
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
      {
        onSuccess: () => {
          setRegisterModalOpen(false);
          setConfirmPassword("");
        },
      },
    );
  };

  const openLogin = () => {
    setRegisterModalOpen(false);
    setLoginModalOpen(true);
  };

  if (!isRegisterModalOpen) return null;

  return (
    <dialog className="modal modal-open">
      <div className="modal-box max-w-md max-h-[80dvh] sm:max-h-[85vh] p-0 overflow-hidden flex flex-col rounded-2xl sm:rounded-3xl border border-base-300 shadow-2xl bg-base-100">
        {/* Fixed Header */}
        <div className="px-4 sm:px-6 py-3.5 sm:py-4 border-b border-base-200 flex items-center justify-between gap-3 shrink-0 bg-base-100">
          <div className="flex items-center gap-2.5 min-w-0">
            <div className="w-7 h-7 rounded-lg bg-primary/10 flex items-center justify-center shrink-0">
              <BookOpen size={16} className="text-primary" />
            </div>
            <h3 className="font-bold text-base sm:text-lg leading-tight truncate">
              {t("auth.create_account", "Create Account")}
            </h3>
          </div>
          <button
            type="button"
            onClick={() => setRegisterModalOpen(false)}
            className="btn btn-ghost btn-circle btn-sm -mr-1.5 text-base-content/70 hover:text-base-content shrink-0"
            aria-label={t("common.close", "Close")}
          >
            <X className="w-4 h-4 sm:w-5 sm:h-5 stroke-[2.5]" />
          </button>
        </div>

        {/* Scrollable Body */}
        <div className="p-4 sm:p-6 overflow-y-auto flex-1 min-h-0 space-y-4">
          <div className="text-center mb-1">
            <h4 className="text-xl font-bold">
              {t("auth.create_account", "Create Account")}
            </h4>
            <p className="text-xs text-base-content/60 mt-0.5">
              {t(
                "auth.register_desc",
                "Register to access the library features.",
              )}
            </p>
          </div>

        {settings && !settings.registration_enabled ? (
          <div className="text-center py-6">
            <p className="text-sm text-base-content/60">
              {t(
                "auth.registration_disabled_desc",
                "Public registration is currently disabled by the administrator.",
              )}
            </p>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <div className="form-control w-full">
              <label className="label">
                <span className="label-text font-semibold">
                  {t("auth.email", "Email")}
                </span>
              </label>
              <input
                type="email"
                placeholder="account@example.com"
                className="input input-bordered w-full focus:input-primary"
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
                <div className="form-control w-full">
                  <label className="label">
                    <span className="label-text font-semibold">
                      {t("auth.full_name", "Full Name")}
                    </span>
                  </label>
                  <input
                    type="text"
                    placeholder={t("auth.optional", "(optional)")}
                    className="input input-bordered w-full focus:input-primary"
                    value={form.full_name}
                    onChange={(e) =>
                      setForm({ ...form, full_name: e.target.value })
                    }
                  />
                </div>
                <div className="form-control w-full">
                  <label className="label">
                    <span className="label-text font-semibold">
                      {t("auth.password", "Password")}
                    </span>
                  </label>
                  <input
                    type="password"
                    placeholder={t("auth.password_min", "Minimum 8 characters")}
                    className="input input-bordered w-full focus:input-primary"
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

                <div className="form-control w-full">
                  <label className="label">
                    <span className="label-text font-semibold">
                      {t("settings.confirm_password", "Confirm new password")}
                    </span>
                  </label>
                  <input
                    type="password"
                    placeholder={t("auth.password_min", "Minimum 8 characters")}
                    className="input input-bordered w-full focus:input-primary"
                    value={confirmPassword}
                    onChange={(e) => {
                      setConfirmPassword(e.target.value);
                      setValidationError("");
                    }}
                    required
                    minLength={8}
                    autoComplete="new-password"
                  />
                </div>

                {(validationError || registerMutation.error) && (
                  <div className="alert alert-error py-2 text-sm rounded-lg">
                    <span>
                      {validationError ||
                        (registerMutation.error instanceof Error
                          ? registerMutation.error.message
                          : String(registerMutation.error))}
                    </span>
                  </div>
                )}

                <button
                  className="btn btn-primary mt-2 w-full"
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
        )}

        <div className="text-center mt-4 pt-3 border-t border-base-200">
          <span className="text-sm text-base-content/60 mr-1.5">
            {t("auth.already_have_account", "Already have an account?")}
          </span>
          <button
            onClick={openLogin}
            className="text-sm link link-primary font-semibold"
          >
            {t("auth.sign_in", "Sign in")}
          </button>
        </div>
        </div>
      </div>
      <form method="dialog" className="modal-backdrop" onClick={() => setRegisterModalOpen(false)}>
        <button type="button">{t("common.close", "Close")}</button>
      </form>
    </dialog>
  );
}
