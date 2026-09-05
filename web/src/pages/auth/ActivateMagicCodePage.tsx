import React, { useEffect, useState } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import {
  QrCode,
  Smartphone,
  CheckCircle2,
  AlertCircle,
  ArrowLeft,
} from "lucide-react";
import { useActivateMagicCodeMutation } from "@/hooks";
import { useAuthStore } from "@/stores";

export const ActivateMagicCodePage: React.FC = () => {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { user } = useAuthStore();
  const isAuthenticated = !!user;

  const codeFromUrl = searchParams.get("code") || "";
  const [code, setCode] = useState(codeFromUrl);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const activateMutation = useActivateMagicCodeMutation();

  useEffect(() => {
    if (codeFromUrl) {
      setCode(codeFromUrl);
    }
  }, [codeFromUrl]);

  const handleActivate = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    if (!code.trim()) return;

    setError(null);
    setSuccess(false);

    try {
      const data = await activateMutation.mutateAsync(code.trim());
      if (data?.status) {
        setSuccess(true);
      } else {
        setError(
          data?.message ||
            t(
              "profile.ereader_activate_failed",
              "Failed to activate eReader code",
            ),
        );
      }
    } catch (err: any) {
      setError(
        err.response?.data?.message ||
          err.message ||
          t(
            "profile.ereader_activate_failed",
            "Failed to activate eReader code",
          ),
      );
    }
  };

  return (
    <div className="min-h-screen bg-base-200 flex flex-col items-center justify-center p-4">
      <div className="w-full max-w-md card bg-base-100 shadow-xl border border-base-300">
        <div className="card-body p-6 text-center">
          <div className="mx-auto grid h-14 w-14 place-items-center rounded-2xl bg-primary/10 text-primary mb-2">
            <QrCode className="h-7 w-7" />
          </div>

          <h2 className="card-title text-xl justify-center font-bold">
            {t("profile.ereader_login_title", "eReader Quick Login")}
          </h2>

          <p className="text-sm text-base-content/70">
            {t(
              "profile.ereader_login_desc",
              "Log in to your Kindle, Kobo, Boox or mobile eReader without typing your password.",
            )}
          </p>

          {!isAuthenticated ? (
            <div className="alert alert-warning text-xs mt-4 text-left flex items-start gap-2">
              <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
              <div>
                <p className="font-semibold">
                  {t("auth.login_required", "Login Required")}
                </p>
                <p>
                  {t(
                    "auth.login_to_activate_ereader",
                    "Please log in to your NovelHub account first to activate this eReader.",
                  )}
                </p>
                <button
                  onClick={() =>
                    navigate(
                      `/login?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`,
                    )
                  }
                  className="btn btn-primary btn-xs mt-2"
                >
                  {t("auth.login_now", "Log In Now")}
                </button>
              </div>
            </div>
          ) : success ? (
            <div className="my-6 space-y-4">
              <div className="mx-auto grid h-16 w-16 place-items-center rounded-full bg-success/20 text-success">
                <CheckCircle2 className="h-10 w-10" />
              </div>
              <h3 className="text-lg font-bold text-success">
                {t("profile.device_activated_heading", "Device Activated!")}
              </h3>
              <p className="text-sm text-base-content/80">
                {t(
                  "profile.device_activated_desc",
                  "Your eReader device is now authenticated and logged in. You can close this window.",
                )}
              </p>
              <button
                onClick={() => navigate("/library")}
                className="btn btn-outline btn-sm gap-2 mt-4"
              >
                <ArrowLeft className="w-4 h-4" />
                {t("common.back_to_library", "Back to Library")}
              </button>
            </div>
          ) : (
            <form
              onSubmit={handleActivate}
              className="mt-4 space-y-4 text-left"
            >
              <div>
                <label className="label text-xs font-semibold text-base-content/70">
                  {t(
                    "profile.enter_6_digit_code",
                    "Enter 6-Digit Code displayed on eReader",
                  )}
                </label>
                <input
                  type="text"
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                  placeholder="849-102"
                  maxLength={10}
                  className="input input-bordered w-full text-center font-mono text-xl tracking-widest uppercase font-bold"
                  autoFocus
                />
              </div>

              {error && (
                <div className="alert alert-error text-xs p-3 flex items-center gap-2">
                  <AlertCircle className="w-4 h-4 shrink-0" />
                  <span>{error}</span>
                </div>
              )}

              <button
                type="submit"
                disabled={activateMutation.isPending || !code.trim()}
                className="btn btn-primary w-full gap-2"
              >
                {activateMutation.isPending ? (
                  <span className="loading loading-spinner loading-sm"></span>
                ) : (
                  <>
                    <Smartphone className="w-4 h-4" />
                    {t(
                      "profile.confirm_activation",
                      "Confirm & Connect eReader",
                    )}
                  </>
                )}
              </button>
            </form>
          )}
        </div>
      </div>
    </div>
  );
};
