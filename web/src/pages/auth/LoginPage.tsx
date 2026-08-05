import { TOTPCodeStep } from "@/components/common";
import { useLoginFlow, usePublicSettings } from "@/hooks";
import { useAuthStore } from "@/stores";
import { BookOpen, Loader2, LogIn } from "lucide-react";
import { SyntheticEvent, useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";

export function LoginPage() {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const user = useAuthStore((state) => state.user);
  const settings = usePublicSettings();
  const { mutation: loginMutation, needsCode, resetCode, submit } = useLoginFlow();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  useEffect(() => {
    if (user) {
      navigate("/", { replace: true });
    }
  }, [user, navigate]);

  async function handleSubmit(event: SyntheticEvent) {
    event.preventDefault();
    submit(email, password);
  }

  return (
    <div className="min-h-screen bg-base-200 flex items-center justify-center p-4">
      <div className="card w-full max-w-md bg-base-100 shadow-xl">
        <div className="card-body">
          <div className="flex flex-col items-center gap-2 mb-6 text-center">
            <div className="w-14 h-14 rounded-2xl bg-primary/10 flex items-center justify-center">
              <BookOpen size={28} className="text-primary" />
            </div>
            <h2 className="text-2xl font-bold">NovelHub</h2>
            <p className="text-sm text-base-content/60">
              {t("auth.login_desc", "Sign in to access the library.")}
            </p>
          </div>

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div className="form-control">
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
                placeholder="account@example.com"
                autoComplete="email"
                className="input input-bordered w-full focus:input-primary"
                required
              />
            </div>
            <div className="form-control">
              <label className="label">
                <span className="label-text font-semibold">{t("auth.password")}</span>
                {settings?.password_reset_enabled && (
                  <Link to="/forgot-password" className="label-text-alt link link-hover">
                    {t("auth.forgot_password", "Forgot password?")}
                  </Link>
                )}
              </label>
              <input
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                type="password"
                placeholder="********"
                autoComplete="current-password"
                className="input input-bordered w-full focus:input-primary"
                required
              />
            </div>

            {loginMutation.error && (
              <div className="alert alert-error mt-2 py-2 text-sm rounded-lg flex items-center gap-2">
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
                className="btn btn-primary mt-4 w-full"
                disabled={loginMutation.isPending}
              >
                {loginMutation.isPending ? (
                  <Loader2 className="animate-spin" size={20} />
                ) : (
                  <LogIn size={20} />
                )}
                {t("auth.sign_in")}
              </button>
            )}
          </form>

          {settings?.registration_enabled && (
            <div className="text-center mt-6 pt-4 border-t border-base-200">
              <span className="text-sm text-base-content/60 mr-1.5">
                {t("auth.dont_have_account", "Don't have an account?")}
              </span>
              <Link to="/register" className="text-sm link link-primary font-semibold">
                {t("auth.register_now", "Register now")}
              </Link>
            </div>
          )}

          <div className="text-center mt-4">
            <Link to="/" className="text-sm link link-hover text-base-content/50">
              {t("auth.back_to_library", "Back to library")}
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
