import { useLoginFlow, usePublicSettings } from "@/hooks";
import { TOTPCodeStep } from "./TOTPCodeStep";
import { useAuthStore } from "@/stores";
import { BookOpen, LogIn } from "lucide-react";
import { SyntheticEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { useShallow } from "zustand/react/shallow";
import { Link } from "react-router-dom";

export function LoginView() {
  const { isLoginModalOpen, setLoginModalOpen, setRegisterModalOpen } = useAuthStore(
    useShallow((state) => ({
      isLoginModalOpen: state.isLoginModalOpen,
      setLoginModalOpen: state.setLoginModalOpen,
      setRegisterModalOpen: state.setRegisterModalOpen,
    }))
  );

  const settings = usePublicSettings();
  const { mutation: loginMutation, needsCode, resetCode, submit } = useLoginFlow();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const { t } = useTranslation();

  async function handleSubmit(event: SyntheticEvent) {
    event.preventDefault();
    submit(email, password);
  }

  return (
    <dialog className={`modal ${isLoginModalOpen ? "modal-open" : ""}`}>
      <div className="modal-box">
        <button 
          onClick={() => setLoginModalOpen(false)}
          className="btn btn-sm btn-circle btn-ghost absolute right-2 top-2"
        >
          ✕
        </button>
        <div className="flex flex-col items-center gap-4 mb-8 mt-2 text-center">
          <div className="w-14 h-14 rounded-2xl bg-primary/10 flex items-center justify-center">
            <BookOpen size={28} className="text-primary" />
          </div>
          <div>
            <h3 className="text-2xl font-bold">NovelHub</h3>
            <p className="text-base-content/60 font-medium text-sm mt-1">{t("auth.login_to_account")}</p>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div className="form-control w-full">
            <label className="label">
              <span className="label-text font-semibold">{t("auth.email")}</span>
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
              <span className="label-text font-semibold">{t("auth.password")}</span>
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
              <span>{loginMutation.error instanceof Error ? loginMutation.error.message : String(loginMutation.error)}</span>
            </div>
          )}

          {needsCode ? (
            <TOTPCodeStep
              pending={loginMutation.isPending}
              onSubmit={(code) => submit(email, password, code)}
            />
          ) : (
            <button
              className="btn btn-primary mt-4 w-full"
              disabled={loginMutation.isPending}
            >
              {loginMutation.isPending ? <span className="loading loading-spinner"></span> : <LogIn size={20} />}
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
      <form method="dialog" className="modal-backdrop">
        <button onClick={() => setLoginModalOpen(false)}>close</button>
      </form>
    </dialog>
  );
}
