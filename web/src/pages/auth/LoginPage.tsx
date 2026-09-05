import { TOTPCodeStep, TopNav } from "@/components/common";
import { useLoginFlow, usePublicSettings } from "@/hooks";
import { useAuthStore } from "@/stores";
import { BookOpen, Loader2, LogIn, Shield } from "lucide-react";
import { SyntheticEvent, useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { API_BASE } from "@/config/api";

export function LoginPage() {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const user = useAuthStore((state) => state.user);
  const settings = usePublicSettings();
  const {
    mutation: loginMutation,
    needsCode,
    resetCode,
    submit,
  } = useLoginFlow();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [urlError, setUrlError] = useState<string | null>(null);

  useEffect(() => {
    if (user) {
      navigate("/", { replace: true });
    }
  }, [user, navigate]);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const err = params.get("error");
    if (err) {
      setUrlError(err);
      window.history.replaceState({}, document.title, window.location.pathname);
    }
  }, []);

  async function handleSubmit(event: SyntheticEvent) {
    event.preventDefault();
    setUrlError(null);
    submit(email, password);
  }

  return (
    <div className="min-h-screen bg-base-100 flex flex-col font-sans">
      <TopNav showSidebarToggle={false} />
      <div className="flex-1 flex items-center justify-center p-4">
        <div className="card w-full max-w-md bg-base-100 shadow-xl border border-base-200">
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
                  <span className="label-text font-semibold">
                    {t("auth.email")}
                  </span>
                </label>
                <input
                  value={email}
                  onChange={(event) => {
                    setEmail(event.target.value);
                    resetCode();
                    setUrlError(null);
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
                  <span className="label-text font-semibold">
                    {t("auth.password")}
                  </span>
                  {settings?.password_reset_enabled && (
                    <Link
                      to="/forgot-password"
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
                  placeholder="********"
                  autoComplete="current-password"
                  className="input input-bordered w-full focus:input-primary"
                  required
                />
              </div>

              {(loginMutation.error || urlError) && (
                <div className="alert alert-error mt-2 py-2 text-sm rounded-lg flex items-center gap-2">
                  <span>
                    {urlError ||
                      (loginMutation.error instanceof Error
                        ? loginMutation.error.message
                        : String(loginMutation.error))}
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

            {settings?.oauth?.providers &&
              settings.oauth.providers.some((p) => p.enabled) && (
                <div className="flex flex-col gap-2 mt-4">
                  <div className="divider text-xs text-base-content/40 uppercase font-semibold">
                    {t("auth.or_sign_in_with", "Or sign in with")}
                  </div>
                  <div className="flex flex-col gap-2">
                    {settings.oauth.providers
                      .filter((p) => p.enabled)
                      .map((p) => {
                        let href = `${API_BASE}/auth/oauth2/${p.id}/login?redirect=${encodeURIComponent(window.location.origin)}`;

                        let icon = null;
                        if (p.id === "google") {
                          icon = (
                            <svg
                              className="h-4 w-4 shrink-0"
                              viewBox="0 0 24 24"
                              fill="currentColor"
                            >
                              <path
                                d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
                                fill="#4285F4"
                              />
                              <path
                                d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                                fill="#34A853"
                              />
                              <path
                                d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.06H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.94l2.85-2.22.81-.63z"
                                fill="#FBBC05"
                              />
                              <path
                                d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.06l3.66 2.84c.87-2.6 3.3-4.52 6.16-4.52z"
                                fill="#EA4335"
                              />
                            </svg>
                          );
                        } else if (p.id === "github") {
                          icon = (
                            <svg
                              className="h-4 w-4 shrink-0"
                              viewBox="0 0 24 24"
                              fill="currentColor"
                            >
                              <path
                                fillRule="evenodd"
                                clipRule="evenodd"
                                d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482C19.138 20.193 22 16.44 22 12.017 22 6.484 17.522 2 12 2z"
                              />
                            </svg>
                          );
                        } else if (p.id === "discord") {
                          icon = (
                            <svg
                              className="h-4 w-4 shrink-0"
                              viewBox="0 0 24 24"
                              fill="currentColor"
                            >
                              <path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028c.462-.63.874-1.295 1.226-1.994.021-.041.001-.09-.041-.106a13.094 13.094 0 0 1-1.873-.894.077.077 0 0 1-.008-.128c.126-.093.252-.19.372-.287a.075.075 0 0 1 .077-.011c3.92 1.793 8.18 1.793 12.061 0a.073.073 0 0 1 .078.009c.12.099.246.195.373.289a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.894.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.156-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.156 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.156-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.156 2.418z" />
                            </svg>
                          );
                        } else {
                          icon = <Shield size={16} className="shrink-0" />;
                        }

                        return (
                          <a
                            key={p.id}
                            href={href}
                            className="btn btn-outline btn-sm h-10 w-full gap-2 border-base-300 font-semibold normal-case"
                          >
                            {icon}
                            {t(
                              "auth.sign_in_with_provider",
                              `Sign in with ${p.display_name}`,
                              { provider: p.display_name },
                            )}
                          </a>
                        );
                      })}
                  </div>
                </div>
              )}

            {settings?.registration_enabled && (
              <div className="text-center mt-6 pt-4 border-t border-base-200">
                <span className="text-sm text-base-content/60 mr-1.5">
                  {t("auth.dont_have_account", "Don't have an account?")}
                </span>
                <Link
                  to="/register"
                  className="text-sm link link-primary font-semibold"
                >
                  {t("auth.register_now", "Register now")}
                </Link>
              </div>
            )}

            <div className="text-center mt-4">
              <Link
                to="/"
                className="text-sm link link-hover text-base-content/50"
              >
                {t("auth.back_to_library", "Back to library")}
              </Link>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
