import { OTPCodeStep, PasswordStrength } from "@/components/common";
import { usePublicSettings, useRegisterMutation } from "@/hooks";
import { useAuthStore } from "@/stores";
import { BookOpen, Loader2 } from "lucide-react";
import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { useShallow } from "zustand/react/shallow";

export function RegisterView() {
  const { isRegisterModalOpen, setRegisterModalOpen, setLoginModalOpen } = useAuthStore(
    useShallow((state) => ({
      isRegisterModalOpen: state.isRegisterModalOpen,
      setRegisterModalOpen: state.setRegisterModalOpen,
      setLoginModalOpen: state.setLoginModalOpen,
    }))
  );

  const { t } = useTranslation();
  const settings = usePublicSettings();
  const registerMutation = useRegisterMutation();

  const [form, setForm] = useState({ email: "", password: "", full_name: "" });
  const [ticket, setTicket] = useState("");

  const verifyRequired = settings?.require_email_verify ?? false;
  const needsVerification = verifyRequired && !ticket;

  const handleSubmit = (e: React.SyntheticEvent) => {
    e.preventDefault();
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
        },
      }
    );
  };

  const openLogin = () => {
    setRegisterModalOpen(false);
    setLoginModalOpen(true);
  };

  return (
    <dialog className={`modal ${isRegisterModalOpen ? "modal-open" : ""}`}>
      <div className="modal-box max-w-md">
        <button
          onClick={() => setRegisterModalOpen(false)}
          className="btn btn-sm btn-circle btn-ghost absolute right-2 top-2"
        >
          ✕
        </button>
        <div className="flex flex-col items-center gap-2 mb-6 mt-2 text-center">
          <div className="w-14 h-14 rounded-2xl bg-primary/10 flex items-center justify-center">
            <BookOpen size={28} className="text-primary" />
          </div>
          <h3 className="text-2xl font-bold">{t("auth.create_account", "Create Account")}</h3>
          <p className="text-xs text-base-content/60">
            {t("auth.register_desc", "Register to access the library features.")}
          </p>
        </div>

        {settings && !settings.registration_enabled ? (
          <div className="text-center py-6">
            <p className="text-sm text-base-content/60">
              {t("auth.registration_disabled_desc", "Public registration is currently disabled by the administrator.")}
            </p>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <div className="form-control w-full">
              <label className="label">
                <span className="label-text font-semibold">{t("auth.email", "Email")}</span>
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
              <OTPCodeStep email={form.email} purpose="email_verify" onVerified={setTicket} />
            ) : (
              <>
                {verifyRequired && (
                  <div className="alert alert-success py-2 text-sm rounded-lg">
                    {t("auth.otp_verified", "Email verified")}
                  </div>
                )}
                <div className="form-control w-full">
                  <label className="label">
                    <span className="label-text font-semibold">{t("auth.full_name", "Full Name")}</span>
                  </label>
                  <input
                    type="text"
                    placeholder={t("auth.optional", "(optional)")}
                    className="input input-bordered w-full focus:input-primary"
                    value={form.full_name}
                    onChange={(e) => setForm({ ...form, full_name: e.target.value })}
                  />
                </div>
                <div className="form-control w-full">
                  <label className="label">
                    <span className="label-text font-semibold">{t("auth.password", "Password")}</span>
                  </label>
                  <input
                    type="password"
                    placeholder={t("auth.password_min", "Minimum 8 characters")}
                    className="input input-bordered w-full focus:input-primary"
                    value={form.password}
                    onChange={(e) => setForm({ ...form, password: e.target.value })}
                    required
                    minLength={8}
                    autoComplete="new-password"
                  />
                  {form.password.length > 0 && <PasswordStrength password={form.password} />}
                </div>

                {registerMutation.error && (
                  <div className="alert alert-error py-2 text-sm rounded-lg">
                    <span>
                      {registerMutation.error instanceof Error
                        ? registerMutation.error.message
                        : String(registerMutation.error)}
                    </span>
                  </div>
                )}

                <button className="btn btn-primary mt-2 w-full" disabled={registerMutation.isPending}>
                  {registerMutation.isPending ? <Loader2 className="animate-spin" size={20} /> : null}
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
          <button onClick={openLogin} className="text-sm link link-primary font-semibold">
            {t("auth.sign_in", "Sign in")}
          </button>
        </div>
      </div>
      <form method="dialog" className="modal-backdrop">
        <button onClick={() => setRegisterModalOpen(false)}>close</button>
      </form>
    </dialog>
  );
}
