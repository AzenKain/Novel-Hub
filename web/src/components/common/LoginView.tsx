import { useLoginFlow, usePublicSettings } from "@/hooks";
import { TOTPCodeStep } from "./TOTPCodeStep";
import { useAuthStore } from "@/stores";
import { BookOpen, LogIn, X } from "lucide-react";
import { SyntheticEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { useShallow } from "zustand/react/shallow";
import { Link } from "react-router-dom";

export function LoginView() {
  const { isLoginModalOpen, setLoginModalOpen, setRegisterModalOpen } =
    useAuthStore(
      useShallow((state) => ({
        isLoginModalOpen: state.isLoginModalOpen,
        setLoginModalOpen: state.setLoginModalOpen,
        setRegisterModalOpen: state.setRegisterModalOpen,
      })),
    );

  const settings = usePublicSettings();
  const siteLogo = settings?.site?.logo || "/logo.svg";
  const siteTitle = settings?.site?.title || "NovelHub";
  const {
    mutation: loginMutation,
    needsCode,
    resetCode,
    submit,
  } = useLoginFlow();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const { t } = useTranslation();

  async function handleSubmit(event: SyntheticEvent) {
    event.preventDefault();
    submit(email, password);
  }

  if (!isLoginModalOpen) return null;

  return (
    <dialog className="modal modal-open">
      <div className="modal-box max-w-md max-h-[80dvh] sm:max-h-[85vh] p-0 overflow-hidden flex flex-col rounded-2xl sm:rounded-3xl border border-base-300 shadow-2xl bg-base-100">
        {/* Fixed Header */}
        <div className="px-4 sm:px-6 py-3.5 sm:py-4 border-b border-base-200 flex items-center justify-between gap-3 shrink-0 bg-base-100">
          <div className="flex items-center gap-2.5 min-w-0">
            {siteLogo ? (
              <img
                src={siteLogo}
                alt={t("common.alt_logo", "Logo")}
                className="w-7 h-7 object-contain drop-shadow-xs shrink-0"
              />
            ) : (
              <div className="w-7 h-7 rounded-lg bg-primary/10 flex items-center justify-center shrink-0">
                <BookOpen size={16} className="text-primary" />
              </div>
            )}
            <h3 className="font-bold text-base sm:text-lg leading-tight truncate">
              {siteTitle}
            </h3>
          </div>
          <button
            type="button"
            onClick={() => setLoginModalOpen(false)}
            className="btn btn-ghost btn-circle btn-sm -mr-1.5 text-base-content/70 hover:text-base-content shrink-0"
            aria-label={t("common.close", "Close")}
          >
            <X className="w-4 h-4 sm:w-5 sm:h-5 stroke-[2.5]" />
          </button>
        </div>

        {/* Scrollable Body */}
        <div className="p-4 sm:p-6 overflow-y-auto flex-1 min-h-0 space-y-4">
          <div className="text-center mb-1">
            <h4 className="text-xl font-bold">{t("auth.sign_in")}</h4>
            <p className="text-base-content/60 font-medium text-xs sm:text-sm mt-0.5">
              {t("auth.login_to_account")}
            </p>
          </div>

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div className="form-control w-full">
              <label className="label">
                <span className="label-text font-semibold">
                  {t("auth.email")}
                </span>
              </label>
              <input
                value={email}
                onChange={(event) => {
                  setEmail(event.target.value);
                  resetCode();
                }}
                type="email"
                placeholder={"account@example.com"}
                autoComplete="email"
                className="input input-bordered w-full focus:input-primary"
              />
            </div>
            <div className="form-control w-full">
              <label className="label">
                <span className="label-text font-semibold">
                  {t("auth.password")}
                </span>
                {settings?.password_reset_enabled && (
                  <Link
                    to="/forgot-password"
                    onClick={() => setLoginModalOpen(false)}
                    className="label-text-alt link link-hover"
                  >
                    {t("auth.forgot_password", "Forgot password?")}
                  </Link>
                )}
              </label>
              <input
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                type="password"
                placeholder={"********"}
                autoComplete="current-password"
                className="input input-bordered w-full focus:input-primary"
              />
            </div>

            {loginMutation.error && (
              <div className="alert alert-error mt-2 py-2 text-sm rounded-lg">
                <span>
                  {loginMutation.error instanceof Error
                    ? loginMutation.error.message
                    : String(loginMutation.error)}
                </span>
              </div>
            )}

            {needsCode ? (
              <TOTPCodeStep
                pending={loginMutation.isPending}
                onSubmit={(code) => submit(email, password, code)}
              />
            ) : (
              <button
                className="btn btn-primary mt-2 w-full"
                disabled={loginMutation.isPending}
              >
                {loginMutation.isPending ? (
                  <span className="loading loading-spinner"></span>
                ) : (
                  <LogIn size={20} />
                )}
                {t("auth.sign_in")}
              </button>
            )}
          </form>

          {settings?.registration_enabled && (
            <div className="text-center mt-4 pt-3 border-t border-base-200">
              <span className="text-sm text-base-content/60 mr-1.5">
                {t("auth.dont_have_account", "Don't have an account?")}
              </span>
              <button
                onClick={() => {
                  setLoginModalOpen(false);
                  setRegisterModalOpen(true);
                }}
                className="text-sm link link-primary font-semibold"
              >
                {t("auth.register_now", "Register now")}
              </button>
            </div>
          )}
        </div>
      </div>
      <form method="dialog" className="modal-backdrop" onClick={() => setLoginModalOpen(false)}>
        <button type="button">{t("common.close", "Close")}</button>
      </form>
    </dialog>
  );
}
